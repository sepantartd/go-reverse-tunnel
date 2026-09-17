package notify

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type EventType string

const (
	EventClientConnected    EventType = "CLIENT_CONNECTED"
	EventClientDisconnected EventType = "CLIENT_DISCONNECTED"
	EventAuthFailed         EventType = "AUTH_FAILED"
)

type WebhookPayload struct {
	Event     EventType `json:"event"`
	ClientID  string    `json:"client_id,omitempty"`
	RemoteAddr string   `json:"remote_addr,omitempty"`
	Message   string    `json:"message"`
	Timestamp string    `json:"timestamp"`
}

func SendWebhook(url string, event EventType, clientID, remoteAddr, message string) {
	if url == "" {
		return
	}

	payload := WebhookPayload{
		Event:      event,
		ClientID:   clientID,
		RemoteAddr: remoteAddr,
		Message:    message,
		Timestamp:  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
}
