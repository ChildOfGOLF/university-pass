package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"university-pass/internal/model"
	"university-pass/internal/repository"

	"github.com/redis/go-redis/v9"
)

type LogService struct {
	logRepo *repository.LogRepository
	rdb     *redis.Client
}

func NewLogService(logRepo *repository.LogRepository, rdb *redis.Client) *LogService {
	return &LogService{logRepo: logRepo, rdb: rdb}
}

const (
	logQueueKey   = "logs:queue"
	processingKey = "logs:processing"
	deadLetterKey = "logs:deadletter"
	batchSize     = 50
	flushEvery    = 1 * time.Second
	pollInterval  = 200 * time.Millisecond
)

func (ls *LogService) PublicAccessLogEvent(ctx context.Context, event *model.AccessLogEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return ls.rdb.RPush(ctx, logQueueKey, string(data)).Err()
}

func (ls *LogService) RecoverInFlight(ctx context.Context) error {
	for {
		val, err := ls.rdb.LMove(ctx, processingKey, logQueueKey, "left", "left").Result()
		if err == redis.Nil {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to recover in-flight logs: %w", err)
		}
		log.Printf("recovered in-flight log event from previous run: %s", val)
	}
}

func (ls *LogService) StartLogWorker(ctx context.Context) {
	flushTimer := time.NewTimer(flushEvery)
	ticker := time.NewTicker(pollInterval)
	defer flushTimer.Stop()
	defer ticker.Stop()

	doFlush := func(flushCtx context.Context) {
		events, raw, err := ls.popLogBatch(flushCtx, batchSize)
		if err != nil {
			log.Printf("Error fetching logs: %v", err)
			return
		}
		if len(events) == 0 {
			return
		}

		valid := make([]*model.AccessLogEvent, 0, len(events))
		validRaw := make([]string, 0, len(events))
		for i, event := range events {
			if event.AccessPointID == 0 {
				log.Printf("Warning: AccessPointID = 0, moving event to dead-letter")
				ls.deadLetter(context.Background(), raw[i], "access_point_id missing")
				continue
			}
			valid = append(valid, event)
			validRaw = append(validRaw, raw[i])
		}
		if len(valid) == 0 {
			return
		}

		buffer := make([]*model.AccessLog, 0, len(valid))
		for _, event := range valid {
			buffer = append(buffer, &model.AccessLog{
				UserID: event.UserID, GuestPassID: event.GuestPassID,
				AccessPointID: event.AccessPointID, Direction: event.Direction,
				IsAllowed: event.IsAllowed, Reason: event.Reason, LoggedAt: event.LoggedAt,
			})
		}

		if err := ls.logRepo.SaveAccessLogBatch(flushCtx, buffer); err != nil {
			log.Printf("Error saving log batch, isolating poison events: %v", err)
			ls.saveIndividuallyOrDeadLetter(context.Background(), valid, validRaw)
			return
		}

		ls.ackProcessed(context.Background(), validRaw)
		log.Printf("Successfully processed %d logs", len(buffer))
	}

	for {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			doFlush(shutdownCtx)
			cancel()
			log.Println("Log worker stopped")
			return

		case <-flushTimer.C:
			doFlush(ctx)
			flushTimer.Reset(flushEvery)

		case <-ticker.C:
			length, err := ls.rdb.LLen(ctx, logQueueKey).Result()
			if err != nil {
				log.Printf("Error checking queue length: %v", err)
				continue
			}
			if length >= batchSize {
				doFlush(ctx)
				flushTimer.Reset(flushEvery)
			}
		}
	}
}

func (ls *LogService) popLogBatch(ctx context.Context, size int) ([]*model.AccessLogEvent, []string, error) {
	if size <= 0 {
		return nil, nil, nil
	}

	raw := make([]string, 0, size)
	for i := 0; i < size; i++ {
		val, err := ls.rdb.LMove(ctx, logQueueKey, processingKey, "left", "right").Result()
		if err == redis.Nil {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to move logs to processing queue: %w", err)
		}
		raw = append(raw, val)
	}

	events := make([]*model.AccessLogEvent, 0, len(raw))
	keptRaw := make([]string, 0, len(raw))
	for _, r := range raw {
		var event model.AccessLogEvent
		if err := json.Unmarshal([]byte(r), &event); err != nil {
			log.Printf("Failed to unmarshal event, moving to dead-letter: %v", err)
			ls.deadLetter(context.Background(), r, "unmarshal error: "+err.Error())
			continue
		}
		events = append(events, &event)
		keptRaw = append(keptRaw, r)
	}
	return events, keptRaw, nil
}

func (ls *LogService) saveIndividuallyOrDeadLetter(ctx context.Context, events []*model.AccessLogEvent, raw []string) {
	for i, event := range events {
		single := []*model.AccessLog{{
			UserID: event.UserID, GuestPassID: event.GuestPassID,
			AccessPointID: event.AccessPointID, Direction: event.Direction,
			IsAllowed: event.IsAllowed, Reason: event.Reason, LoggedAt: event.LoggedAt,
		}}
		if err := ls.logRepo.SaveAccessLogBatch(ctx, single); err != nil {
			log.Printf("Poison event, moving to dead-letter: %v", err)
			ls.deadLetter(ctx, raw[i], err.Error())
			continue
		}
		ls.ackProcessed(ctx, []string{raw[i]})
	}
}

func (ls *LogService) ackProcessed(ctx context.Context, raw []string) {
	for _, r := range raw {
		if err := ls.rdb.LRem(ctx, processingKey, 1, r).Err(); err != nil {
			log.Printf("Failed to ack processed log event: %v", err)
		}
	}
}

func (ls *LogService) deadLetter(ctx context.Context, raw string, reason string) {
	if err := ls.rdb.RPush(ctx, deadLetterKey, raw).Err(); err != nil {
		log.Printf("Failed to move event to dead-letter queue (%s): %v", reason, err)
	}
	if err := ls.rdb.LRem(ctx, processingKey, 1, raw).Err(); err != nil {
		log.Printf("Failed to remove dead-lettered event from processing queue: %v", err)
	}
}
