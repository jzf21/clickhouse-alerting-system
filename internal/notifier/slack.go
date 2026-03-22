package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jozef/clickhouse-alerting-system/internal/model"
)

type SlackSender struct{}

func (s *SlackSender) Send(ctx context.Context, channelConfig json.RawMessage, alert Alert) error {
	var cfg model.SlackConfig
	if err := json.Unmarshal(channelConfig, &cfg); err != nil {
		return fmt.Errorf("parsing slack config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("no webhook URL configured")
	}

	color := "#36a64f" // green for resolved
	status := "Resolved"
	if alert.State == "firing" {
		switch alert.Severity {
		case "critical":
			color = "#e01e5a"
		case "warning":
			color = "#ecb22e"
		default:
			color = "#2eb67d"
		}
		status = "Firing"
	}

	fields := []map[string]string{
		{"type": "mrkdwn", "text": fmt.Sprintf("*Value:* `%.4g`", alert.Value)},
		{"type": "mrkdwn", "text": fmt.Sprintf("*Threshold:* %s %.4g", operatorSymbol(alert.Operator), alert.Threshold)},
		{"type": "mrkdwn", "text": fmt.Sprintf("*Severity:* %s", alert.Severity)},
		{"type": "mrkdwn", "text": fmt.Sprintf("*Status:* %s", status)},
	}

	blocks := []map[string]interface{}{
		{
			"type": "header",
			"text": map[string]string{
				"type": "plain_text",
				"text": fmt.Sprintf("[%s] %s - %s", alert.Severity, alert.RuleName, status),
			},
		},
		{
			"type":   "section",
			"fields": fields,
		},
	}

	if len(alert.Labels) > 0 {
		labelsText := ""
		for k, v := range alert.Labels {
			labelsText += fmt.Sprintf("`%s=%s` ", k, v)
		}
		blocks = append(blocks, map[string]interface{}{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Labels:* %s", labelsText),
			},
		})
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{"color": color, "blocks": blocks},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func operatorSymbol(op string) string {
	switch op {
	case "gt":
		return ">"
	case "gte":
		return ">="
	case "lt":
		return "<"
	case "lte":
		return "<="
	case "eq":
		return "=="
	case "neq":
		return "!="
	default:
		return op
	}
}
