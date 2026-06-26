package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/dispatch"
	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/realtime"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/jackc/pgx/v5"
)

type GroupStorer interface {
	ClaimGroup(ctx context.Context, groupID, driverID string) (int64, error)
	GetByIDWithMembers(ctx context.Context, id string) (*store.GroupDetail, error)
	GetActiveForDriver(ctx context.Context, driverID string) (*models.RideGroup, error)
	CompleteRide(ctx context.Context, groupID, driverID string) (bool, error)
}

type DriverStorer interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.Driver, error)
	SetTelegramChat(ctx context.Context, telegramID int64, chatID int64) error
	SetStatus(ctx context.Context, id, status string) error
	Touch(ctx context.Context, telegramID int64) error
}

type ChatStorer interface {
	AddMessage(ctx context.Context, groupID, senderType, content string) (*models.ChatMessage, error)
}

type EventPublisher interface {
	BroadcastMulti([]string, realtime.Event)
}

type Bot struct {
	token   string
	chatID  int64
	client  *http.Client
	baseURL string
	gs      GroupStorer
	ds      DriverStorer
	cs      ChatStorer
	events  EventPublisher
}

func New(token string, chatID int64, gs GroupStorer, ds DriverStorer, cs ChatStorer, events EventPublisher) *Bot {
	return &Bot{
		token:   token,
		chatID:  chatID,
		client:  &http.Client{Timeout: 60 * time.Second},
		baseURL: "https://api.telegram.org",
		gs:      gs,
		ds:      ds,
		cs:      cs,
		events:  events,
	}
}

func NewWithBaseURL(token string, chatID int64, gs GroupStorer, ds DriverStorer, cs ChatStorer, events EventPublisher, baseURL string) *Bot {
	bot := New(token, chatID, gs, ds, cs, events)
	bot.baseURL = baseURL
	return bot
}

func (b *Bot) apiPost(ctx context.Context, method string, body any) ([]byte, error) {
	url := fmt.Sprintf("%s/bot%s/%s", b.baseURL, b.token, method)
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

func (b *Bot) SendDispatch(ctx context.Context, groupID, msg string) (int, error) {
	req := sendMessageRequest{
		ChatID: b.chatID,
		Text:   msg,
		ReplyMarkup: &InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{
					{Text: "✅ Accept", CallbackData: "accept:" + groupID},
					{Text: "⏭ Pass", CallbackData: "pass:" + groupID},
				},
			},
		},
	}

	respBody, err := b.apiPost(ctx, "sendMessage", req)
	if err != nil {
		return 0, err
	}

	var res struct {
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &res); err != nil {
		return 0, err
	}
	return res.Result.MessageID, nil
}

func (b *Bot) SendMessage(ctx context.Context, chatID int64, text string) error {
	req := sendMessageRequest{ChatID: chatID, Text: text}
	_, err := b.apiPost(ctx, "sendMessage", req)
	return err
}

func (b *Bot) EditMessage(ctx context.Context, msgID int, text string) error {
	req := editMessageTextRequest{
		ChatID:    b.chatID,
		MessageID: msgID,
		Text:      text,
	}
	_, err := b.apiPost(ctx, "editMessageText", req)
	return err
}

func (b *Bot) AnswerCallback(ctx context.Context, callbackID, text string) error {
	req := answerCallbackQueryRequest{
		CallbackQueryID: callbackID,
		Text:            text,
	}
	_, err := b.apiPost(ctx, "answerCallbackQuery", req)
	return err
}

func isCommand(text, cmd string) bool {
	text = strings.TrimSpace(text)
	return text == cmd || strings.HasPrefix(text, cmd+" ")
}

func (b *Bot) HandleUpdate(ctx context.Context, update Update) {
	if update.Message != nil {
		_ = b.ds.Touch(ctx, update.Message.From.ID) // presence
	} else if update.CallbackQuery != nil {
		_ = b.ds.Touch(ctx, update.CallbackQuery.From.ID) // presence
	}

	if update.Message != nil && update.Message.Chat.Type == "private" {
		if isCommand(update.Message.Text, "/start") {
			if err := b.ds.SetTelegramChat(ctx, update.Message.From.ID, update.Message.Chat.ID); err != nil {
				slog.Error("telegram: failed to set chat", "error", err)
				return
			}
	
			req := sendMessageRequest{
				ChatID: update.Message.Chat.ID,
				Text:   "Registered. You'll receive ride dispatch requests here. Send /online when you're ready to drive.",
			}
			_, _ = b.apiPost(ctx, "sendMessage", req)
			return
		}

		if isCommand(update.Message.Text, "/complete") || isCommand(update.Message.Text, "/done") {
			b.handleCompleteRide(ctx, *update.Message)
			return
		}

		if isCommand(update.Message.Text, "/online") {
			b.handleOnline(ctx, *update.Message)
			return
		}

		if isCommand(update.Message.Text, "/offline") {
			b.handleOffline(ctx, *update.Message)
			return
		}

		if !strings.HasPrefix(strings.TrimSpace(update.Message.Text), "/") && strings.TrimSpace(update.Message.Text) != "" {
			b.handleDriverMessage(ctx, *update.Message)
			return
		}
	}

	if update.CallbackQuery != nil {
		if strings.HasPrefix(update.CallbackQuery.Data, "accept:") {
			b.handleAccept(ctx, *update.CallbackQuery)
		} else {
			_ = b.AnswerCallback(ctx, update.CallbackQuery.ID, "")
			if strings.HasPrefix(update.CallbackQuery.Data, "pass:") {
				// do nothing after AnswerCallback
			} else {
				slog.Info("telegram: unknown callback data", "data", update.CallbackQuery.Data)
			}
		}
	}
}

func (b *Bot) handleAccept(ctx context.Context, cq CallbackQuery) {
	driver, err := b.ds.GetByTelegramID(ctx, cq.From.ID)
	if err != nil || driver == nil {
		_ = b.AnswerCallback(ctx, cq.ID, "You are not registered as a driver")
		return
	}
	_ = b.AnswerCallback(ctx, cq.ID, "")

	groupID := strings.TrimPrefix(cq.Data, "accept:")

	rowsAffected, err := b.gs.ClaimGroup(ctx, groupID, driver.ID)
	if err != nil || rowsAffected == 0 {
		_ = b.EditMessage(ctx, cq.Message.MessageID, "⚠️ Already taken by another driver")
		return
	}

	_ = b.ds.SetStatus(ctx, driver.ID, "busy")

	detail, err := b.gs.GetByIDWithMembers(ctx, groupID)
	if err != nil {
		slog.Error("telegram: failed to get group details", "error", err)
		return
	}

	_, _, msg, err := dispatch.GenerateMessage(detail.Members)
	if err != nil {
		slog.Error("telegram: failed to compute optimal sequence", "error", err)
		return
	}

	finalText := fmt.Sprintf("✅ Accepted by %s\n\n%s", driver.Name, msg)
	_ = b.EditMessage(ctx, cq.Message.MessageID, finalText)
}

func (b *Bot) handleDriverMessage(ctx context.Context, msg Message) {
	driver, err := b.ds.GetByTelegramID(ctx, msg.From.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return // unregistered Telegram user DMing the bot — expected, no-op
		}
		slog.Error("handleDriverMessage: driver lookup failed", "error", err)
		return
	}

	group, err := b.gs.GetActiveForDriver(ctx, driver.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = b.SendMessage(ctx, msg.Chat.ID, "You don't have an active ride.")
			return
		}
		slog.Error("handleDriverMessage: active group lookup failed", "error", err)
		_ = b.SendMessage(ctx, msg.Chat.ID, "You don't have an active ride.")
		return
	}

	chatMsg, err := b.cs.AddMessage(ctx, group.ID, "driver", msg.Text)
	if err != nil {
		slog.Error("telegram: failed to add driver message to chat", "error", err)
		return
	}

	if b.events != nil {
		b.events.BroadcastMulti([]string{group.ID}, realtime.ChatMessageEvent(*chatMsg))
	}
}

func (b *Bot) handleCompleteRide(ctx context.Context, msg Message) {
	driver, err := b.ds.GetByTelegramID(ctx, msg.From.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = b.SendMessage(ctx, msg.Chat.ID, "You are not registered as a driver.")
			return
		}
		slog.Error("handleCompleteRide: driver lookup failed", "error", err)
		return
	}

	group, err := b.gs.GetActiveForDriver(ctx, driver.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = b.SendMessage(ctx, msg.Chat.ID, "You don't have an active ride.")
			return
		}
		slog.Error("handleCompleteRide: active group lookup failed", "error", err)
		return
	}

	completed, err := b.gs.CompleteRide(ctx, group.ID, driver.ID)
	if err != nil {
		slog.Error("handleCompleteRide: CompleteRide failed", "error", err)
		return
	}
	if !completed {
		_ = b.SendMessage(ctx, msg.Chat.ID, "Ride already closed out.")
		return
	}

	// UX Decision: Automatically return the driver to 'online' (auto-ready for next dispatch) 
	// rather than 'offline' so they don't have to manually spam /online after every single ride.
	_ = b.ds.SetStatus(ctx, driver.ID, "online") // presence

	_ = b.SendMessage(ctx, msg.Chat.ID, "✅ Ride completed! You are now available for new dispatch requests.")

	if b.events != nil {
		b.events.BroadcastMulti([]string{"global", group.ID}, realtime.GroupCompleted(group.ID, driver.ID))
	}
}

func (b *Bot) handleOnline(ctx context.Context, msg Message) {
	driver, err := b.ds.GetByTelegramID(ctx, msg.From.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = b.SendMessage(ctx, msg.Chat.ID, "You are not registered as a driver.")
			return
		}
		slog.Error("handleOnline: driver lookup failed", "error", err)
		return
	}

	if driver.Status == "busy" {
		_ = b.SendMessage(ctx, msg.Chat.ID, "You're on an active ride — finish that first.")
		return
	}

	_ = b.ds.SetStatus(ctx, driver.ID, "online")
	_ = b.SendMessage(ctx, msg.Chat.ID, "You're online. Dispatches will come through here.")
}

func (b *Bot) handleOffline(ctx context.Context, msg Message) {
	driver, err := b.ds.GetByTelegramID(ctx, msg.From.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			_ = b.SendMessage(ctx, msg.Chat.ID, "You are not registered as a driver.")
			return
		}
		slog.Error("handleOffline: driver lookup failed", "error", err)
		return
	}

	if driver.Status == "busy" {
		_ = b.SendMessage(ctx, msg.Chat.ID, "Finish your current ride before going offline.")
		return
	}

	_ = b.ds.SetStatus(ctx, driver.ID, "offline")
	_ = b.SendMessage(ctx, msg.Chat.ID, "You're offline.")
}
