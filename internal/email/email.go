package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Sender sends emails via the Resend API.
type Sender struct {
	apiKey string
	to     string
}

// New creates a Sender. Pass the Resend API key and the recipient email.
func New(apiKey, to string) *Sender {
	return &Sender{apiKey: apiKey, to: to}
}

type resendRequest struct {
	From    string `json:"from"`
	To      []string `json:"to"`
	Subject string `json:"subject"`
	Text    string `json:"text"`
}

// Send sends an email via Resend.
func (s *Sender) Send(fromName, fromEmail, message string) error {
	body := resendRequest{
		From:    "Terminal Portfolio <onboarding@resend.dev>",
		To:      []string{s.to},
		Subject: fmt.Sprintf("Message from %s (%s) via tewodros.me", fromName, fromEmail),
		Text:    fmt.Sprintf("From: %s <%s>\n\n%s", fromName, fromEmail, message),
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
