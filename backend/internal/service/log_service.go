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
	return &LogService{
		logRepo: logRepo,
		rdb:     rdb,
	}
}

func (ls *LogService) PublicAccessLogEvent(ctx context.Context, event *model.AccessLogEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return ls.rdb.RPush(ctx, "logs:queue", string(data)).Err()
}

const (
	logQueueKey  = "logs:queue"
	batchSize    = 50
	flushEvery   = 5 * time.Minute
	pollInterval = 200 * time.Millisecond
)

func (ls *LogService) StartLogWorker(ctx context.Context) {
	flushTimer := time.NewTimer(flushEvery)
	ticker := time.NewTicker(pollInterval)
	defer flushTimer.Stop()
	defer ticker.Stop()

	doFlush := func(flushCtx context.Context) {
		events, err := ls.popLogBatch(flushCtx, batchSize)
		if err != nil {
			log.Printf("Error fetching logs: %v", err)
			return
		}
		if len(events) == 0 {
			return
		}

		buffer := make([]*model.AccessLog, 0, len(events))
		for _, event := range events {
			if event.AccessPointID == 0 {
				log.Printf("Warning: AccessPointID = 0 skipping event")
				continue
			}
			buffer = append(buffer, &model.AccessLog{
				UserID:        event.UserID,
				GuestPassID:   event.GuestPassID,
				AccessPointID: event.AccessPointID,
				Direction:     event.Direction,
				IsAllowed:     event.IsAllowed,
				Reason:        event.Reason,
				LoggedAt:      event.LoggedAt,
			})
		}

		if err := ls.logRepo.SaveAccessLogBatch(flushCtx, buffer); err != nil {
			log.Printf("Error saving logs to DB: %v", err)
			// атомарность, возвращаются обратно
			ls.requeue(context.Background(), events)
			return
		}

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

func (ls *LogService) requeue(ctx context.Context, events []*model.AccessLogEvent) {
	for i := len(events) - 1; i >= 0; i-- {
		data, err := json.Marshal(events[i])
		if err != nil {
			log.Printf("Failed to marshal event for requeue: %v", err)
			continue
		}
		if err := ls.rdb.LPush(ctx, logQueueKey, string(data)).Err(); err != nil {
			log.Printf("Failed to requeue event: %v", err)
		}
	}
}

func (ls *LogService) popLogBatch(ctx context.Context, size int) ([]*model.AccessLogEvent, error) {
	if size <= 0 {
		return nil, nil
	}

	results, err := ls.rdb.LPopCount(ctx, logQueueKey, size).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to pop logs from queue: %w", err)
	}

	events := make([]*model.AccessLogEvent, 0, len(results))
	for _, result := range results {
		var event model.AccessLogEvent
		if err := json.Unmarshal([]byte(result), &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			continue
		}
		events = append(events, &event)
	}
	return events, nil
}
