package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"
)

const (
	defaultSuccessTpl = "{{.service}} is up and running!"
	defaultErrorTpl   = "{{.service}} is down because of {{.error}}!"
)

// ChannelNotifier sends notifications using channel configs loaded from main config.
type ChannelNotifier struct {
	mu        sync.RWMutex
	channels  map[string]ChannelConfig
	client    *http.Client
	tmplCache map[string]*template.Template
}

func newChannelNotifier(channels map[string]ChannelConfig) *ChannelNotifier {
	c := &ChannelNotifier{
		channels:  make(map[string]ChannelConfig),
		client:    &http.Client{Timeout: 10 * time.Second},
		tmplCache: make(map[string]*template.Template),
	}
	for k, v := range channels {
		c.channels[k] = v

		success := v.SuccessNotification
		if strings.TrimSpace(success) == "" {
			success = defaultSuccessTpl
		}
		c.tmplCache[k+"_success"] = template.Must(template.New(k + "_success").Parse(success))

		errtpl := v.ErrorNotification
		if strings.TrimSpace(errtpl) == "" {
			errtpl = defaultErrorTpl
		}
		c.tmplCache[k+"_error"] = template.Must(template.New(k + "_error").Parse(errtpl))
	}
	return c
}

func (c *ChannelNotifier) renderTemplate(channel string, typ string, data map[string]string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	key := channel + "_" + typ
	t, ok := c.tmplCache[key]
	if !ok {
		return "", fmt.Errorf("no template for channel=%s type=%s", channel, typ)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}


func (c *ChannelNotifier) Notify(channelName string, msg string) error {
	c.mu.RLock()
	ch, ok := c.channels[channelName]
	c.mu.RUnlock()
	if !ok {
		return fmt.Errorf("channel not found: %s", channelName)
	}
	switch strings.ToLower(ch.Type) {
	case "telegram":
		channel_token_env_name := fmt.Sprintf("CHANNEL_%s_TG_BOT_TOKEN", strings.ToUpper(channelName))
		channel_chat_id_env_name := fmt.Sprintf("CHANNEL_%s_TG_CHAT_ID", strings.ToUpper(channelName))
		token := os.Getenv(channel_token_env_name)
		chatID := os.Getenv(channel_chat_id_env_name)
		if token == "" || chatID == "" {
			return fmt.Errorf(
				"telegram settings not configured via env: (%s/%s)",
				channel_token_env_name,
				channel_chat_id_env_name,
			)
		}
		return c.sendTelegram(token, chatID, msg)
	default:
		return fmt.Errorf("unsupported channel type: %s", ch.Type)
	}
}

func (c *ChannelNotifier) sendTelegram(token, chatID, msg string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload := fmt.Sprintf("chat_id=%s&text=%s", chatID, urlQueryEscape(msg))
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("telegram api returned status %d", resp.StatusCode)
	}
	return nil
}

func urlQueryEscape(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(s, "&", "%26"), "+", "%2B"), " ", "+")
}

// NotifyForServiceState composes and sends messages to given channel names.
func (c *ChannelNotifier) NotifyForServiceState(service string, res Result, channels []string) {
	data := map[string]string{
		"service": service,
		"error":   res.Error,
	}
	typ := "success"
	if !res.OK {
		typ = "error"
	}
	for _, ch := range channels {
		msg, err := c.renderTemplate(ch, typ, data)
		if err != nil {
			warningf("render template failed for %s: %v", ch, err)
			continue
		}
		if err := c.Notify(ch, msg); err != nil {
			warningf("notify failed for %s: %v", ch, err)
		} else {
			infof("notified %s about %s (ok=%v)", ch, service, res.OK)
		}
	}
}
