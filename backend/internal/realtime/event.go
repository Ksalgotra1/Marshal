package realtime

import (
	"encoding/json"
	"time"
)

// Event is a real-time message pushed to live clients.
type Event struct {
	Type      string    `json:"type"`
	GroupID   string    `json:"group_id,omitempty"`
	Data      any       `json:"data,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Marshal serialises the event to JSON bytes.
func (e Event) Marshal() []byte {
	b, _ := json.Marshal(e)
	return b
}

func GroupFormed(groupID string, memberCount int, score float64) Event {
	return Event{
		Type:    "group:formed",
		GroupID: groupID,
		Data: map[string]any{
			"member_count": memberCount,
			"score":        score,
		},
		Timestamp: time.Now(),
	}
}

func RequestCreated(requestID, requesterName string) Event {
	return Event{
		Type: "request:created",
		Data: map[string]any{
			"request_id":     requestID,
			"requester_name": requesterName,
		},
		Timestamp: time.Now(),
	}
}

func DriverRegistered(driverID, driverName string) Event {
	return Event{
		Type: "driver:registered",
		Data: map[string]any{
			"driver_id":   driverID,
			"driver_name": driverName,
		},
		Timestamp: time.Now(),
	}
}

func GroupDispatching(groupID string, attempt int, score float64) Event {
	return Event{
		Type:    "group:dispatching",
		GroupID: groupID,
		Data: map[string]any{
			"attempt": attempt,
			"score":   score,
		},
		Timestamp: time.Now(),
	}
}

func GroupAssigned(groupID, driverID, driverName string) Event {
	return Event{
		Type:    "group:assigned",
		GroupID: groupID,
		Data: map[string]any{
			"driver_id":   driverID,
			"driver_name": driverName,
		},
		Timestamp: time.Now(),
	}
}

func GroupCancelled(groupID string, reason string) Event {
	return Event{
		Type:    "group:cancelled",
		GroupID: groupID,
		Data: map[string]any{
			"reason": reason,
		},
		Timestamp: time.Now(),
	}
}

func MemberJoined(groupID, requesterName string) Event {
	return Event{
		Type:    "member:joined",
		GroupID: groupID,
		Data: map[string]any{
			"requester_name": requesterName,
		},
		Timestamp: time.Now(),
	}
}

func SystemConnections(count int) Event {
	return Event{
		Type: "system:connections",
		Data: map[string]any{
			"connections": count,
		},
		Timestamp: time.Now(),
	}
}
