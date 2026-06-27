package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"crypto/subtle"
)

func NewWebhookHandler(bot *Bot, secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		var update Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		bot.HandleUpdate(r.Context(), update)
		w.WriteHeader(http.StatusOK)
	})
}

func (b *Bot) RegisterWebhook(ctx context.Context, webhookURL, secret string) error {
	req := map[string]string{
		"url":          webhookURL,
		"secret_token": secret,
	}
	respBody, err := b.apiPost(ctx, "setWebhook", req)
	if err != nil {
		return err
	}

	var res struct {
		Ok bool `json:"ok"`
	}
	if err := json.Unmarshal(respBody, &res); err != nil {
		return err
	}
	if !res.Ok {
		return fmt.Errorf("setWebhook returned ok=false")
	}
	return nil
}

func (b *Bot) DeleteWebhook(ctx context.Context) error {
	respBody, err := b.apiPost(ctx, "deleteWebhook", struct{}{})
	if err != nil {
		return err
	}

	var res struct {
		Ok bool `json:"ok"`
	}
	if err := json.Unmarshal(respBody, &res); err != nil {
		return err
	}
	if !res.Ok {
		return fmt.Errorf("deleteWebhook returned ok=false")
	}
	return nil
}
