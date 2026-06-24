package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/dispatch"
	"github.com/Ksalgotra1/Marshal/internal/models"
	"github.com/Ksalgotra1/Marshal/internal/store"
)

type GroupStorer interface {
	ClaimGroup(ctx context.Context, groupID, driverID string) (int64, error)
	GetByIDWithMembers(ctx context.Context, id string) (*store.GroupDetail, error)
}

type DriverStorer interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*models.Driver, error)
	SetTelegramChat(ctx context.Context, telegramID int64, chatID int64) error
	SetStatus(ctx context.Context, id, status string) error
}

type Bot struct {
	token   string
	chatID  int64
	client  *http.Client
	baseURL string
	gs      GroupStorer
	ds      DriverStorer
}

func New(token string, chatID int64, gs GroupStorer, ds DriverStorer) *Bot {
	return &Bot{
		token:   token,
		chatID:  chatID,
		client:  &http.Client{Timeout: 60 * time.Second},
		baseURL: "https://api.telegram.org",
		gs:      gs,
		ds:      ds,
	}
}

func NewWithBaseURL(token string, chatID int64, gs GroupStorer, ds DriverStorer, baseURL string) *Bot {
	bot := New(token, chatID, gs, ds)
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

func (b *Bot) HandleUpdate(ctx context.Context, update Update) {
	if update.Message != nil && strings.HasPrefix(update.Message.Text, "/start") {
		if err := b.ds.SetTelegramChat(ctx, update.Message.From.ID, update.Message.Chat.ID); err != nil {
			slog.Error("telegram: failed to set chat", "error", err)
			return
		}

		req := sendMessageRequest{
			ChatID: update.Message.Chat.ID,
			Text:   "Registered. You'll receive ride dispatch requests here.",
		}
		_, _ = b.apiPost(ctx, "sendMessage", req)
		return
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
