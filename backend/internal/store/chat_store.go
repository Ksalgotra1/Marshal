package store

import (
	"context"
	"fmt"

	"github.com/Ksalgotra1/Marshal/internal/models"
)

// ChatStore handles ride group chat messages.
type ChatStore struct{ DB DBTX }

// AddMessage inserts a new chat message into the group's chat.
func (s *ChatStore) AddMessage(ctx context.Context, groupID, senderType, senderName, content string) (*models.ChatMessage, error) {
	var m models.ChatMessage
	err := s.DB.QueryRow(ctx, `
		INSERT INTO chat_messages (group_id, sender_type, sender_name, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, group_id, sender_type, sender_name, content, created_at
	`, groupID, senderType, senderName, content).Scan(
		&m.ID, &m.GroupID, &m.SenderType, &m.SenderName, &m.Content, &m.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("ChatStore.AddMessage: %w", err)
	}
	return &m, nil
}

// ListMessages fetches recent messages for a group, ordered oldest-first.
func (s *ChatStore) ListMessages(ctx context.Context, groupID string) ([]models.ChatMessage, error) {
	rows, err := s.DB.Query(ctx, `
		SELECT id, group_id, sender_type, sender_name, content, created_at
		FROM chat_messages
		WHERE group_id = $1
		ORDER BY created_at ASC
		LIMIT 200
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("ChatStore.ListMessages: %w", err)
	}
	defer rows.Close()

	var messages []models.ChatMessage
	for rows.Next() {
		var m models.ChatMessage
		if err := rows.Scan(&m.ID, &m.GroupID, &m.SenderType, &m.SenderName, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}
