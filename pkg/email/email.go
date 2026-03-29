// Package email provides email sending functionality using Resend.
package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ResendClient handles sending emails via Resend API.
type ResendClient struct {
	apiKey    string
	fromEmail string
	client    *http.Client
}

// SendEmailRequest represents a Resend API request.
type SendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
	Text    string   `json:"text,omitempty"`
}

// SendEmailResponse represents a Resend API response.
type SendEmailResponse struct {
	ID string `json:"id"`
}

// NewResendClient creates a new Resend email client.
func NewResendClient(apiKey, fromEmail string) *ResendClient {
	return &ResendClient{
		apiKey:    apiKey,
		fromEmail: fromEmail,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendOTP sends an OTP code to the user's email.
func (r *ResendClient) SendOTP(toEmail, otp string, isSignUp bool) error {
	action := "sign in"
	if isSignUp {
		action = "sign up"
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Your OTP Code</title>
</head>
<body style="margin: 0; padding: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; background-color: #f4f4f5;">
  <table role="presentation" cellspacing="0" cellpadding="0" width="100%%" style="min-width: 100%%;">
    <tr>
      <td align="center" style="padding: 40px 20px;">
        <table role="presentation" cellspacing="0" cellpadding="0" width="100%%" style="max-width: 480px; background-color: #ffffff; border-radius: 12px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);">
          <tr>
            <td style="padding: 40px 40px 30px;">
              <div style="text-align: center; margin-bottom: 30px;">
                <div style="display: inline-block; background: linear-gradient(135deg, #3b82f6 0%%, #8b5cf6 100%%); padding: 12px 20px; border-radius: 8px;">
                  <span style="font-size: 24px; font-weight: 700; color: #ffffff; letter-spacing: -0.5px;">go2postgres</span>
                </div>
              </div>
              <h1 style="margin: 0 0 20px; font-size: 24px; font-weight: 600; color: #18181b; text-align: center;">
                Your verification code
              </h1>
              <p style="margin: 0 0 30px; font-size: 16px; line-height: 1.6; color: #52525b; text-align: center;">
                Use the code below to %s to go2postgres. This code expires in 10 minutes.
              </p>
              <div style="background-color: #f4f4f5; border-radius: 8px; padding: 20px; text-align: center; margin-bottom: 30px;">
                <span style="font-size: 32px; font-weight: 700; letter-spacing: 8px; color: #18181b; font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Mono', 'Droid Sans Mono', monospace;">%s</span>
              </div>
              <p style="margin: 0; font-size: 14px; line-height: 1.6; color: #71717a; text-align: center;">
                If you didn't request this code, you can safely ignore this email.
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding: 20px 40px; background-color: #f9fafb; border-radius: 0 0 12px 12px; border-top: 1px solid #e4e4e7;">
              <p style="margin: 0; font-size: 12px; color: #a1a1aa; text-align: center;">
                © 2026 go2postgres. PostgreSQL provisioning made simple.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>
`, action, otp)

	text := fmt.Sprintf("Your go2postgres verification code is: %s\n\nThis code expires in 10 minutes.\n\nIf you didn't request this code, you can safely ignore this email.", otp)

	return r.Send(toEmail, "Your go2postgres verification code", html, text)
}

// Send sends an email via Resend API.
func (r *ResendClient) Send(to, subject, html, text string) error {
	if r.apiKey == "" {
		return fmt.Errorf("Resend API key not configured")
	}

	req := SendEmailRequest{
		From:    r.fromEmail,
		To:      []string{to},
		Subject: subject,
		HTML:    html,
		Text:    text,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+r.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Resend API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
