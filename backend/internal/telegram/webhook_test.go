package telegram

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookRejectsMissingSecret(t *testing.T) {
	bot := New("token", 123, &fakeGroupStore{}, &fakeDriverStore{})
	handler := NewWebhookHandler(bot, "secret")

	req := httptest.NewRequest("POST", "/telegram/webhook", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWebhookRejectsWrongSecret(t *testing.T) {
	bot := New("token", 123, &fakeGroupStore{}, &fakeDriverStore{})
	handler := NewWebhookHandler(bot, "secret")

	req := httptest.NewRequest("POST", "/telegram/webhook", nil)
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWebhookAcceptsCorrectSecret(t *testing.T) {
	var handled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "answerCallbackQuery") {
			handled = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	bot := New("token", 123, &fakeGroupStore{}, &fakeDriverStore{})
	bot.baseURL = server.URL

	handler := NewWebhookHandler(bot, "secret")

	update := Update{
		UpdateID: 1,
		CallbackQuery: &CallbackQuery{
			ID:   "cb1",
			Data: "pass:group1",
		},
	}
	body, _ := json.Marshal(update)
	req := httptest.NewRequest("POST", "/telegram/webhook", bytes.NewReader(body))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, handled) // verify HandleUpdate was called
}

func TestWebhookRejectsBadJSON(t *testing.T) {
	bot := New("token", 123, &fakeGroupStore{}, &fakeDriverStore{})
	handler := NewWebhookHandler(bot, "secret")

	req := httptest.NewRequest("POST", "/telegram/webhook", bytes.NewReader([]byte("{bad json")))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
