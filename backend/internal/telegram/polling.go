package telegram

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

func (b *Bot) StartPolling(ctx context.Context) {
	offset := int64(0)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		req := map[string]any{
			"offset":  offset,
			"timeout": 30,
		}

		respBody, err := b.apiPost(ctx, "getUpdates", req)
		if err != nil {
			slog.Warn("telegram: getUpdates failed", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		var res struct {
			Ok     bool     `json:"ok"`
			Result []Update `json:"result"`
		}
		if err := json.Unmarshal(respBody, &res); err != nil {
			slog.Warn("telegram: getUpdates json decode failed", "error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if !res.Ok {
			slog.Warn("telegram: getUpdates returned ok=false")
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range res.Result {
			b.HandleUpdate(ctx, update)
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
		}
	}
}
