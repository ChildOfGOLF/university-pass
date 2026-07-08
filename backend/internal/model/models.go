package model

import "time"

type User struct {
	ID         int       `json:"id"`
	Email      string    `json:"email"`
	LastName   string    `json:"last_name"`
	FirstName  string    `json:"first_name"`
	Patronymic string    `json:"patronymic"`
	AvatarURL  string    `json:"avatar_url,omitempty"`
	Role       string    `json:"role"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

type UserDevice struct {
	UserID       int       `json:"user_id"`
	DeviceID     string    `json:"device_id"`
	SecretKey    string    `json:"secret_key"`
	LastUsedStep *int64    `json:"last_used_step,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type GuestPass struct {
	ID         string    `json:"id"`
	LastName   string    `json:"last_name"`
	FirstName  string    `json:"first_name"`
	Patronymic string    `json:"patronymic"`
	Purpose    string    `json:"purpose,omitempty"`
	ValidFrom  time.Time `json:"valid_from"`
	ValidTo    time.Time `json:"valid_to"`
	IsUsed     bool      `json:"is_used"`
	IsEntered  bool      `json:"is_entered"` // TODO: think about this field
	CreatedAt  time.Time `json:"created_at"`
}

type AccessLog struct {
	ID            int64     `json:"id"`
	UserID        *int      `json:"user_id,omitempty"`
	GuestPassID   *string   `json:"guest_pass_id,omitempty"`
	AccessPointID int       `json:"access_point_id"`
	Direction     string    `json:"direction"`
	IsAllowed     bool      `json:"is_allowed"`
	Reason        string    `json:"reason,omitempty"`
	LoggedAt      time.Time `json:"logged_at"`
}

type AccessPoint struct {
	ID          int    `json:"id"`
	BuildingID  int    `json:"building_id"`
	ScannerID   string `json:"scanner_id"`
	GateNumber  string `json:"gate_number"`
	Description string `json:"description,omitempty"`
}

type Building struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address,omitempty"`
}

type ScanUserResult struct {
	IsAllowed bool   `json:"is_allowed"`
	Reason    string `json:"reason"`
	User      *User  `json:"user,omitempty"`
}

type AccessLogEvent struct {
	Type          string    `json:"type"` // eg scan
	UserID        *int      `json:"user_id,omitempty"`
	GuestPassID   *string   `json:"guest_pass_id,omitempty"`
	AccessPointID int       `json:"access_point_id,omitempty"`
	Direction     string    `json:"direction"`
	IsAllowed     bool      `json:"is_allowed"`
	Reason        string    `json:"reason,omitempty"`
	ScannerID     string    `json:"scanner_id"`
	LoggedAt      time.Time `json:"logged_at"`
}

type VerifyGuestResult struct {
	IsAllowed bool       `json:"is_allowed"`
	Reason    string     `json:"reason"`
	Guest     *GuestPass `json:"guest,omitempty"`
}

type CreateUserRequest struct {
	Email      string `json:"email"`
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	Patronymic string `json:"patronymic"`
	Phone      string `json:"phone"`
	Role       string `json:"role"`
	GroupID    *int   `json:"group_id,omitempty"` // not null для студента
	Password   string `json:"password"`
}

type UpdateUserRequest struct {
	LastName   *string `json:"last_name"`
	FirstName  *string `json:"first_name"`
	Patronymic *string `json:"patronymic"`
	Phone      *string `json:"phone"`
	IsActive   *bool   `json:"is_active"`
}

type CreateGuestPassRequest struct {
	LastName   string    `json:"last_name"`
	FirstName  string    `json:"first_name"`
	Patronymic string    `json:"patronymic"`
	Purpose    string    `json:"purpose"`
	ValidFrom  time.Time `json:"valid_from"`
	ValidTo    time.Time `json:"valid_to"`
}
