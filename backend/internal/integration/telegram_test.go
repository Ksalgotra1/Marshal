package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Ksalgotra1/Marshal/internal/assigner"
	"github.com/Ksalgotra1/Marshal/internal/geo"
	"github.com/Ksalgotra1/Marshal/internal/store"
	"github.com/Ksalgotra1/Marshal/internal/telegram"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sendMessageCall struct {
	chatID      int64
	text        string
	replyMarkup telegram.InlineKeyboardMarkup
}

type editMessageCall struct {
	chatID      int64
	messageID   int
	text        string
	replyMarkup telegram.InlineKeyboardMarkup
}

func TestTelegramDispatchFlow(t *testing.T) {
	db, _ := setupTestDB(t) // reuse from pipeline_test.go

	// mock Telegram API server — records calls made to it
	var mu sync.Mutex
	var sentMessages []sendMessageCall // {chatID, text, replyMarkup}
	var editedMessages []editMessageCall
	mockTG := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// route by URL path: /sendMessage, /editMessageText, /answerCallbackQuery
		// record the call, return {"ok":true,"result":{"message_id":42}}
		// for sendMessage, return a fixed message_id of 42
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/bottest-token/sendMessage" {
			// decode body to get chatID, text, replyMarkup
			// since it uses Bot, body format matches bot.go definitions.
			// Actually we can parse it from the incoming request body
			// telegram uses json body or query params but bot.go uses json body for sendMessage
			// wait, our bot.go posts JSON? Let's check how apiPost formats the body.
			// Yes, it json.Marshals the body. We can use a generic map to parse it.
			// but for this test, we can just define a struct or use map[string]interface{}.
			// Wait, the prompt says:
			// wait, if we can't easily import bot's internal structs, let's just parse what we can
			// Actually, the test uses telegram types!
			// We can define the struct to match telegram's format if it's not exported.
			// We'll just define it inline:
			var body struct {
				ChatID      int64                         `json:"chat_id"`
				Text        string                        `json:"text"`
				ReplyMarkup telegram.InlineKeyboardMarkup `json:"reply_markup"`
			}
			importJsonErr := json.NewDecoder(r.Body).Decode(&body)
			if importJsonErr == nil {
				sentMessages = append(sentMessages, sendMessageCall{
					chatID:      body.ChatID,
					text:        body.Text,
					replyMarkup: body.ReplyMarkup,
				})
			}
			w.Write([]byte(`{"ok":true,"result":{"message_id":42}}`))
		} else if r.URL.Path == "/bottest-token/editMessageText" {
			var body struct {
				ChatID      int64                         `json:"chat_id"`
				MessageID   int                           `json:"message_id"`
				Text        string                        `json:"text"`
				ReplyMarkup telegram.InlineKeyboardMarkup `json:"reply_markup"`
			}
			importJsonErr := json.NewDecoder(r.Body).Decode(&body)
			if importJsonErr == nil {
				editedMessages = append(editedMessages, editMessageCall{
					chatID:      body.ChatID,
					messageID:   body.MessageID,
					text:        body.Text,
					replyMarkup: body.ReplyMarkup,
				})
			}
			w.Write([]byte(`{"ok":true,"result":{"message_id":42}}`))
		} else if r.URL.Path == "/bottest-token/answerCallbackQuery" {
			w.Write([]byte(`{"ok":true}`))
		} else {
			w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer mockTG.Close()

	// create bot pointing at mock server instead of real Telegram
	gs := &store.GroupStore{DB: db}
	ds := &store.DriverStore{DB: db}
	bot := telegram.NewWithBaseURL("test-token", -1001234, gs, ds, mockTG.URL)

	// register a driver with known telegram_id
	driverTelegramID := int64(6031420785)
	driverID, _ := ds.Register(context.Background(), "Test Driver", driverTelegramID)
	ds.SetStatus(context.Background(), driverID, "online")
	ds.SetTelegramChat(context.Background(), driverTelegramID, -1001234)

	// insert 2 fast-track requests
	pickup := geo.LatLng{Lat: 30.0, Lng: 76.0}
	dropoff := geo.LatLng{Lat: 30.1, Lng: 76.1}
	arriveBy := time.Now().Add(12 * time.Minute)
	insertRequest(db, pickup, dropoff, arriveBy)
	insertRequest(db, pickup, dropoff, arriveBy)

	// run grouper to form the group
	runGrouperOnce(db)

	// run assigner with real bot (points at mock Telegram)
	pub := &recordingPublisher{}
	assigner.Run(context.Background(), db, pub, bot)

	// assert SendDispatch was called
	mu.Lock()
	require.Len(t, sentMessages, 1, "expected 1 dispatch message sent to Telegram")
	assert.Contains(t, sentMessages[0].text, "▶ Pickup")
	assert.Contains(t, sentMessages[0].text, "👉 Route")
	// assert inline keyboard has Accept and Pass buttons
	require.Len(t, sentMessages[0].replyMarkup.InlineKeyboard, 1)
	assert.Equal(t, "✅ Accept", sentMessages[0].replyMarkup.InlineKeyboard[0][0].Text)
	assert.Equal(t, "⏭ Pass", sentMessages[0].replyMarkup.InlineKeyboard[0][1].Text)
	mu.Unlock()

	// assert telegram_msg_id stored on group
	// group is now 'dispatching', use ListFiltered
	all, _ := gs.ListFiltered(context.Background(), store.GroupFilter{Status: "dispatching"})
	require.Len(t, all, 1)
	require.NotNil(t, all[0].TelegramMsgID)
	assert.Equal(t, 42, *all[0].TelegramMsgID)

	// simulate driver tapping Accept
	groupID := all[0].ID
	bot.HandleUpdate(context.Background(), telegram.Update{
		CallbackQuery: &telegram.CallbackQuery{
			ID:      "cq-123",
			From:    telegram.User{ID: driverTelegramID},
			Message: &telegram.Message{MessageID: 42, Chat: telegram.Chat{ID: -1001234}},
			Data:    "accept:" + groupID,
		},
	})

	// assert EditMessage called with acceptance text
	mu.Lock()
	require.Len(t, editedMessages, 1)
	assert.Contains(t, editedMessages[0].text, "✅ Accepted by Test Driver")
	assert.Contains(t, editedMessages[0].text, "👉 Route")
	mu.Unlock()

	// assert group is now 'assigned' and driver is 'busy'
	claimed, _ := gs.ListFiltered(context.Background(), store.GroupFilter{Status: "assigned"})
	require.Len(t, claimed, 1)
	driver, _ := ds.GetByID(context.Background(), driverID)
	assert.Equal(t, "busy", driver.Status)
}
