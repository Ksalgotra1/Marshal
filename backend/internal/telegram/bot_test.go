package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/realtime"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

type fakeGroupStore struct {
	claimGroupFunc         func(ctx context.Context, groupID, driverID string) (int64, error)
	getByIDWithMembersFunc func(ctx context.Context, id string) (*store.GroupDetail, error)
	getActiveForDriverFunc func(ctx context.Context, driverID string) (*models.RideGroup, error)
}

func (f *fakeGroupStore) ClaimGroup(ctx context.Context, groupID, driverID string) (int64, error) {
	return f.claimGroupFunc(ctx, groupID, driverID)
}

func (f *fakeGroupStore) GetByIDWithMembers(ctx context.Context, id string) (*store.GroupDetail, error) {
	return f.getByIDWithMembersFunc(ctx, id)
}

func (f *fakeGroupStore) GetActiveForDriver(ctx context.Context, driverID string) (*models.RideGroup, error) {
	if f.getActiveForDriverFunc != nil {
		return f.getActiveForDriverFunc(ctx, driverID)
	}
	return nil, nil
}

type fakeChatStore struct {
	addMessageFunc func(ctx context.Context, groupID, senderType, content string) (*models.ChatMessage, error)
}

func (f *fakeChatStore) AddMessage(ctx context.Context, groupID, senderType, content string) (*models.ChatMessage, error) {
	if f.addMessageFunc != nil {
		return f.addMessageFunc(ctx, groupID, senderType, content)
	}
	return &models.ChatMessage{GroupID: groupID, SenderType: senderType, Content: content}, nil
}

type fakeEventPublisher struct {
	broadcastMultiFunc func(rooms []string, event realtime.Event)
}

func (f *fakeEventPublisher) BroadcastMulti(rooms []string, event realtime.Event) {
	if f.broadcastMultiFunc != nil {
		f.broadcastMultiFunc(rooms, event)
	}
}

type fakeDriverStore struct {
	getByTelegramIDFunc func(ctx context.Context, telegramID int64) (*models.Driver, error)
	setTelegramChatFunc func(ctx context.Context, telegramID int64, chatID int64) error
	setStatusFunc       func(ctx context.Context, id, status string) error
}

func (f *fakeDriverStore) GetByTelegramID(ctx context.Context, telegramID int64) (*models.Driver, error) {
	return f.getByTelegramIDFunc(ctx, telegramID)
}

func (f *fakeDriverStore) SetTelegramChat(ctx context.Context, telegramID int64, chatID int64) error {
	return f.setTelegramChatFunc(ctx, telegramID, chatID)
}

func (f *fakeDriverStore) SetStatus(ctx context.Context, id, status string) error {
	return f.setStatusFunc(ctx, id, status)
}

func TestHandleUpdateStart(t *testing.T) {
	var chatSet bool
	ds := &fakeDriverStore{
		setTelegramChatFunc: func(ctx context.Context, telegramID, chatID int64) error {
			assert.Equal(t, int64(123), telegramID)
			assert.Equal(t, int64(456), chatID)
			chatSet = true
			return nil
		},
	}
	gs := &fakeGroupStore{}

	var msgSent bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "sendMessage") {
			var req sendMessageRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, int64(456), req.ChatID)
			assert.Contains(t, req.Text, "Registered")
			msgSent = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer server.Close()

	bot := New("token", 789, gs, ds, &fakeChatStore{}, &fakeEventPublisher{})
	bot.baseURL = server.URL

	bot.HandleUpdate(context.Background(), Update{
		Message: &Message{
			Chat: Chat{ID: 456},
			From: User{ID: 123},
			Text: "/start",
		},
	})

	assert.True(t, chatSet)
	assert.True(t, msgSent)
}

func TestHandleUpdateAcceptWinsRace(t *testing.T) {
	var statusSet bool
	ds := &fakeDriverStore{
		getByTelegramIDFunc: func(ctx context.Context, telegramID int64) (*models.Driver, error) {
			return &models.Driver{ID: "driver1", Name: "Bob"}, nil
		},
		setStatusFunc: func(ctx context.Context, id, status string) error {
			assert.Equal(t, "driver1", id)
			assert.Equal(t, "busy", status)
			statusSet = true
			return nil
		},
	}
	gs := &fakeGroupStore{
		claimGroupFunc: func(ctx context.Context, groupID, driverID string) (int64, error) {
			assert.Equal(t, "group1", groupID)
			assert.Equal(t, "driver1", driverID)
			return 1, nil
		},
		getByIDWithMembersFunc: func(ctx context.Context, id string) (*store.GroupDetail, error) {
			return &store.GroupDetail{
				Members: []models.RideRequest{
					{ID: "req1", RequesterName: "Alice", PickupLat: 30.0, PickupLng: 76.0, DropoffLat: 30.1, DropoffLng: 76.1},
				},
			}, nil
		},
	}

	var edited bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "editMessageText") {
			var req editMessageTextRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Contains(t, req.Text, "✅ Accepted by Bob")
			edited = true
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	bot := New("token", 789, gs, ds, &fakeChatStore{}, &fakeEventPublisher{})
	bot.baseURL = server.URL

	bot.HandleUpdate(context.Background(), Update{
		CallbackQuery: &CallbackQuery{
			ID:   "cb1",
			From: User{ID: 123},
			Data: "accept:group1",
			Message: &Message{
				MessageID: 55,
			},
		},
	})

	assert.True(t, statusSet)
	assert.True(t, edited)
}

func TestHandleUpdateAcceptLosesRace(t *testing.T) {
	ds := &fakeDriverStore{
		getByTelegramIDFunc: func(ctx context.Context, telegramID int64) (*models.Driver, error) {
			return &models.Driver{ID: "driver1", Name: "Bob"}, nil
		},
	}
	gs := &fakeGroupStore{
		claimGroupFunc: func(ctx context.Context, groupID, driverID string) (int64, error) {
			return 0, nil // lost race
		},
	}

	var edited bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "editMessageText") {
			var req editMessageTextRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Contains(t, req.Text, "⚠️ Already taken by another driver")
			edited = true
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	bot := New("token", 789, gs, ds, &fakeChatStore{}, &fakeEventPublisher{})
	bot.baseURL = server.URL

	bot.HandleUpdate(context.Background(), Update{
		CallbackQuery: &CallbackQuery{
			ID:   "cb1",
			From: User{ID: 123},
			Data: "accept:group1",
			Message: &Message{
				MessageID: 55,
			},
		},
	})

	assert.True(t, edited)
}

func TestHandleUpdatePass(t *testing.T) {
	ds := &fakeDriverStore{}
	gs := &fakeGroupStore{}

	var answered bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "answerCallbackQuery") {
			answered = true
			w.WriteHeader(http.StatusOK)
		} else {
			t.Errorf("Unexpected API call to %s", r.URL.Path)
		}
	}))
	defer server.Close()

	bot := New("token", 789, gs, ds, &fakeChatStore{}, &fakeEventPublisher{})
	bot.baseURL = server.URL

	bot.HandleUpdate(context.Background(), Update{
		CallbackQuery: &CallbackQuery{
			ID:   "cb1",
			From: User{ID: 123},
			Data: "pass:group1",
		},
	})

	assert.True(t, answered)
}

func TestHandleUpdateUnregisteredDriver(t *testing.T) {
	ds := &fakeDriverStore{
		getByTelegramIDFunc: func(ctx context.Context, telegramID int64) (*models.Driver, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	gs := &fakeGroupStore{}

	var answeredWithText bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "answerCallbackQuery") {
			var req answerCallbackQueryRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Text == "You are not registered as a driver" {
				answeredWithText = true
			}
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	bot := New("token", 789, gs, ds, &fakeChatStore{}, &fakeEventPublisher{})
	bot.baseURL = server.URL

	bot.HandleUpdate(context.Background(), Update{
		CallbackQuery: &CallbackQuery{
			ID:   "cb1",
			From: User{ID: 123},
			Data: "accept:group1",
		},
	})

	assert.True(t, answeredWithText)
}

func TestHandleDriverMessage_GroupTypeFiltered(t *testing.T) {
	var routed bool
	ds := &fakeDriverStore{
		getByTelegramIDFunc: func(ctx context.Context, telegramID int64) (*models.Driver, error) {
			routed = true
			return nil, pgx.ErrNoRows
		},
	}
	bot := New("token", 789, &fakeGroupStore{}, ds, &fakeChatStore{}, &fakeEventPublisher{})
	
	bot.HandleUpdate(context.Background(), Update{
		Message: &Message{
			Chat: Chat{Type: "group", ID: 100},
			From: User{ID: 123},
			Text: "Hello",
		},
	})
	assert.False(t, routed)
}

func TestHandleDriverMessage_NoActiveRide(t *testing.T) {
	ds := &fakeDriverStore{
		getByTelegramIDFunc: func(ctx context.Context, telegramID int64) (*models.Driver, error) {
			return &models.Driver{ID: "driver1"}, nil
		},
	}
	gs := &fakeGroupStore{
		getActiveForDriverFunc: func(ctx context.Context, driverID string) (*models.RideGroup, error) {
			return nil, pgx.ErrNoRows
		},
	}
	var sent bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "sendMessage") {
			var req sendMessageRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "You don't have an active ride.", req.Text)
			sent = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
		}
	}))
	defer server.Close()

	var broadcast bool
	ep := &fakeEventPublisher{
		broadcastMultiFunc: func(rooms []string, event realtime.Event) {
			broadcast = true
		},
	}
	bot := New("token", 789, gs, ds, &fakeChatStore{}, ep)
	bot.baseURL = server.URL

	bot.HandleUpdate(context.Background(), Update{
		Message: &Message{
			Chat: Chat{Type: "private", ID: 100},
			From: User{ID: 123},
			Text: "Hello",
		},
	})
	assert.True(t, sent)
	assert.False(t, broadcast)
}

func TestHandleDriverMessage_Success(t *testing.T) {
	ds := &fakeDriverStore{
		getByTelegramIDFunc: func(ctx context.Context, telegramID int64) (*models.Driver, error) {
			return &models.Driver{ID: "driver1"}, nil
		},
	}
	gs := &fakeGroupStore{
		getActiveForDriverFunc: func(ctx context.Context, driverID string) (*models.RideGroup, error) {
			return &models.RideGroup{ID: "group1"}, nil
		},
	}
	var msgAdded bool
	cs := &fakeChatStore{
		addMessageFunc: func(ctx context.Context, groupID, senderType, content string) (*models.ChatMessage, error) {
			assert.Equal(t, "group1", groupID)
			assert.Equal(t, "driver", senderType)
			assert.Equal(t, "Hello student", content)
			msgAdded = true
			return &models.ChatMessage{GroupID: groupID, Content: content}, nil
		},
	}
	var broadcast bool
	ep := &fakeEventPublisher{
		broadcastMultiFunc: func(rooms []string, event realtime.Event) {
			assert.Equal(t, []string{"group1"}, rooms)
			broadcast = true
		},
	}
	bot := New("token", 789, gs, ds, cs, ep)
	bot.HandleUpdate(context.Background(), Update{
		Message: &Message{
			Chat: Chat{Type: "private", ID: 100},
			From: User{ID: 123},
			Text: "Hello student",
		},
	})
	assert.True(t, msgAdded)
	assert.True(t, broadcast)
}

func TestHandleDriverMessage_UnregisteredDriver(t *testing.T) {
	ds := &fakeDriverStore{
		getByTelegramIDFunc: func(ctx context.Context, telegramID int64) (*models.Driver, error) {
			return nil, pgx.ErrNoRows
		},
	}
	var routed bool
	gs := &fakeGroupStore{
		getActiveForDriverFunc: func(ctx context.Context, driverID string) (*models.RideGroup, error) {
			routed = true
			return nil, nil
		},
	}
	bot := New("token", 789, gs, ds, &fakeChatStore{}, &fakeEventPublisher{})
	bot.HandleUpdate(context.Background(), Update{
		Message: &Message{
			Chat: Chat{Type: "private", ID: 100},
			From: User{ID: 123},
			Text: "Hello",
		},
	})
	assert.False(t, routed)
}

func TestHandleDriverMessage_DBErrorLogs(t *testing.T) {
	ds := &fakeDriverStore{
		getByTelegramIDFunc: func(ctx context.Context, telegramID int64) (*models.Driver, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}
	bot := New("token", 789, &fakeGroupStore{}, ds, &fakeChatStore{}, &fakeEventPublisher{})

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	slog.SetDefault(logger)

	bot.HandleUpdate(context.Background(), Update{
		Message: &Message{
			Chat: Chat{Type: "private", ID: 100},
			From: User{ID: 123},
			Text: "Hello",
		},
	})

	assert.Contains(t, logBuf.String(), "handleDriverMessage: driver lookup failed")
	assert.Contains(t, logBuf.String(), "connection refused")
}
