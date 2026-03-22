package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jozef/clickhouse-alerting-system/internal/model"
)

type WebhookSender struct{}

func (w *WebhookSender) Send(ctx context.Context, channelConfig json.RawMessage, alert Alert) error {
	var cfg model.WebhookConfig
	if err := json.Unmarshal(channelConfig, &cfg); err != nil {
		return fmt.Errorf("parsing webhook config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("no webhook URL configured")
	}

	method := cfg.Method
	if method == "" {
		method = http.MethodPost
	}

	body, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("marshaling webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
