package models

import (
	"encoding/json"
	"time"
)

const (
	PriorityHigh   = "high"
	PriorityNormal = "normal"
)

// RideRequest represents a student's ride request.
type RideRequest struct {
	ID            string    `json:"id"`
	RequesterName string    `json:"requester_name"`
	PickupLat     float64   `json:"pickup_lat"`
	PickupLng     float64   `json:"pickup_lng"`
	DropoffLat    float64   `json:"dropoff_lat"`
	DropoffLng    float64   `json:"dropoff_lng"`
	PickupH3      *int64    `json:"pickup_h3,omitempty"`
	DropoffH3     *int64    `json:"dropoff_h3,omitempty"`
	ArriveBy      time.Time `json:"arrive_by"`
	Status        string    `json:"status"`
	GroupID       *string   `json:"group_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateRequestPayload is the JSON body for POST /api/requests.
type CreateRequestPayload struct {
	RequesterName string    `json:"requester_name"`
	PickupLat     float64   `json:"pickup_lat"`
	PickupLng     float64   `json:"pickup_lng"`
	DropoffLat    float64   `json:"dropoff_lat"`
	DropoffLng    float64   `json:"dropoff_lng"`
	ArriveBy      time.Time `json:"arrive_by"`
}

// RideGroup represents a formed group of compatible ride requests.
type RideGroup struct {
	ID                string     `json:"id"`
	Status            string     `json:"status"`
	Priority          string     `json:"priority"`
	RouteScore        float64    `json:"route_score"`
	ArriveBy          time.Time  `json:"arrive_by"`
	ExpectedDeparture *time.Time `json:"expected_departure,omitempty"`
	DriverID          *string    `json:"driver_id,omitempty"`
	DispatchAttempts  int        `json:"dispatch_attempts"`
	TelegramMsgID     *int       `json:"telegram_msg_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// GroupMember links a request to a group.
type GroupMember struct {
	RequestID string    `json:"request_id"`
	GroupID   string    `json:"group_id"`
	JoinedAt  time.Time `json:"joined_at"`
}

// Driver represents a registered driver.
type Driver struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	TelegramID   int64     `json:"telegram_id"`
	TelegramChat *int64    `json:"telegram_chat,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// Job represents a unit of work in the Postgres-backed job queue.
type Job struct {
	ID          string          `json:"id"`
	JobType     string          `json:"job_type"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"`
	Priority    string          `json:"priority"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	RunAfter    time.Time       `json:"run_after"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ChatMessage represents a message in the Telegram <-> WebSocket relay.
type ChatMessage struct {
	ID         string    `json:"id"`
	GroupID    string    `json:"group_id"`
	SenderType string    `json:"sender_type"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}
