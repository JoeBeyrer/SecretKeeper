package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
)

// smtpCfg reads credentials from environment variables at call time.
// Set before starting server.

func smtpCfg() (host, port, user, pass, from, baseURL string) {
	host = "smtp.gmail.com"
	port = "587"
	user = os.Getenv("SMTP_USER")
	pass = os.Getenv("SMTP_PASS")
	from = user
	baseURL = os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4200"
	}
	return
}

var resetEmailTmpl = template.Must(template.New("reset").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>Reset your password</title>
</head>
<body style="margin:0;padding:0;background:#1a1e24;font-family:Helvetica,Arial,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="padding:40px 20px;">
    <tr>
      <td align="center">
        <table width="100%" style="max-width:480px;background:#303742;border-radius:8px;padding:40px;">
          <tr>
            <td>
              <h1 style="color:#ffffff;font-size:22px;font-weight:600;margin:0 0 16px;">
                Password Reset Request
              </h1>
              <p style="color:#aab0ba;font-size:14px;line-height:1.6;margin:0 0 24px;">
                We received a request to reset the password for your SecretKeeper account.
                Click the button below to choose a new password. This link expires in
                <strong style="color:#ffffff;">1 hour</strong>.
              </p>
              <a href="{{ . }}"
                 style="display:inline-block;padding:12px 28px;background:#1a702b;color:#ffffff;
                        text-decoration:none;border-radius:4px;font-size:15px;font-weight:600;">
                Reset Password
              </a>
              <p style="color:#aab0ba;font-size:12px;line-height:1.6;margin:24px 0 0;">
                If the button doesn't work, copy and paste this link into your browser:<br/>
                <a href="{{ . }}" style="color:#34c9eb;word-break:break-all;">{{ . }}</a>
              </p>
              <hr style="border:none;border-top:1px solid #4a5260;margin:28px 0;" />
              <p style="color:#666d7a;font-size:12px;margin:0;">
                If you did not request a password reset you can safely ignore this email.
                Your password will not change.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`))

func SendPasswordResetEmail(toAddress, token string) error {
	host, port, user, pass, from, baseURL := smtpCfg()
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)

	var htmlBody bytes.Buffer
	if err := resetEmailTmpl.Execute(&htmlBody, resetLink); err != nil {
		return fmt.Errorf("email: failed to render template: %w", err)
	}

	msg := buildMessage(from, toAddress, "Reset your SecretKeeper password", htmlBody.String(), resetLink)
	auth := smtp.PlainAuth("", user, pass, host)
	if err := smtp.SendMail(host+":"+port, auth, from, []string{toAddress}, msg); err != nil {
		return fmt.Errorf("email: smtp.SendMail failed: %w", err)
	}
	return nil
}

var verifyEmailTmpl = template.Must(template.New("verify").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <title>Verify your email</title>
</head>
<body style="margin:0;padding:0;background:#1a1e24;font-family:Helvetica,Arial,sans-serif;">
  <table width="100%" cellpadding="0" cellspacing="0" style="padding:40px 20px;">
    <tr>
      <td align="center">
        <table width="100%" style="max-width:480px;background:#303742;border-radius:8px;padding:40px;">
          <tr>
            <td>
              <h1 style="color:#ffffff;font-size:22px;font-weight:600;margin:0 0 16px;">
                Verify your email
              </h1>
              <p style="color:#aab0ba;font-size:14px;line-height:1.6;margin:0 0 24px;">
                Thanks for signing up for SecretKeeper! Click the button below to verify
                your email address. This link expires in
                <strong style="color:#ffffff;">24 hours</strong>.
              </p>
              <a href="{{ . }}"
                 style="display:inline-block;padding:12px 28px;background:#1a702b;color:#ffffff;
                        text-decoration:none;border-radius:4px;font-size:15px;font-weight:600;">
                Verify Email
              </a>
              <p style="color:#aab0ba;font-size:12px;line-height:1.6;margin:24px 0 0;">
                If the button doesn't work, copy and paste this link into your browser:<br/>
                <a href="{{ . }}" style="color:#34c9eb;word-break:break-all;">{{ . }}</a>
              </p>
              <hr style="border:none;border-top:1px solid #4a5260;margin:28px 0;" />
              <p style="color:#666d7a;font-size:12px;margin:0;">
                If you did not create a SecretKeeper account you can safely ignore this email.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`))

func SendVerificationEmail(toAddress, token string) error {
	host, port, user, pass, from, baseURL := smtpCfg()
	verifyLink := fmt.Sprintf("http://localhost:8080/api/verify-email?token=%s", token)

	var htmlBody bytes.Buffer
	if err := verifyEmailTmpl.Execute(&htmlBody, verifyLink); err != nil {
		return fmt.Errorf("email: failed to render template: %w", err)
	}

	msg := buildMessage(from, toAddress, "Verify your SecretKeeper email", htmlBody.String(), verifyLink)
	auth := smtp.PlainAuth("", user, pass, host)
	if err := smtp.SendMail(host+":"+port, auth, from, []string{toAddress}, msg); err != nil {
		return fmt.Errorf("email: smtp.SendMail failed: %w", err)
	}
	return nil
}


func buildMessage(from, to, subject, htmlBody, plainLink string) []byte {
	boundary := "==SecretKeeperBoundary=="

	var buf bytes.Buffer
	buf.WriteString("From: SecretKeeper <" + from + ">\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n")
	buf.WriteString("\r\n")

	// Plain-text fallback
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
	buf.WriteString(subject + "\r\n\r\n")
	buf.WriteString(plainLink + "\r\n\r\n")
	buf.WriteString("If you did not request this, you can safely ignore this email.\r\n")

	// HTML
	buf.WriteString("--" + boundary + "\r\n")
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
	buf.WriteString(htmlBody)
	buf.WriteString("\r\n")

	buf.WriteString("--" + boundary + "--\r\n")
	return buf.Bytes()
}
