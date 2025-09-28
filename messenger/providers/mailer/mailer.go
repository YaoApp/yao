package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/yaoapp/yao/messenger/types"
)

// Provider implements the Provider interface for SMTP email sending and IMAP receiving
type Provider struct {
	config   types.ProviderConfig
	host     string
	port     int
	username string
	password string
	from     string
	useTLS   bool
	useSSL   bool

	// IMAP configuration for receiving emails
	imapHost     string
	imapPort     int
	imapUsername string
	imapPassword string
	imapUseSSL   bool
	imapMailbox  string
}

// NewMailerProvider creates a new Mailer provider
func NewMailerProvider(config types.ProviderConfig) (*Provider, error) {
	provider := &Provider{
		config: config,
		useTLS: true, // Default to TLS
	}

	// Extract options
	options := config.Options
	if options == nil {
		return nil, fmt.Errorf("mailer provider requires options")
	}

	// Parse SMTP configuration (required)
	smtpConfig, ok := options["smtp"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("mailer provider requires 'smtp' configuration")
	}

	// Required SMTP options
	if host, ok := smtpConfig["host"].(string); ok {
		provider.host = host
	} else {
		return nil, fmt.Errorf("SMTP configuration requires 'host' option")
	}

	if port, ok := smtpConfig["port"]; ok {
		switch p := port.(type) {
		case int:
			provider.port = p
		case float64:
			provider.port = int(p)
		case string:
			var err error
			provider.port, err = strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("invalid SMTP port: %s", p)
			}
		default:
			return nil, fmt.Errorf("invalid SMTP port type")
		}
	} else {
		provider.port = 587 // Default SMTP port
	}

	if username, ok := smtpConfig["username"].(string); ok {
		provider.username = username
	} else {
		return nil, fmt.Errorf("SMTP configuration requires 'username' option")
	}

	if password, ok := smtpConfig["password"].(string); ok {
		provider.password = password
	} else {
		return nil, fmt.Errorf("SMTP configuration requires 'password' option")
	}

	if from, ok := smtpConfig["from"].(string); ok {
		provider.from = from
	} else {
		return nil, fmt.Errorf("SMTP configuration requires 'from' option")
	}

	// Optional SMTP options
	if useTLS, ok := smtpConfig["use_tls"].(bool); ok {
		provider.useTLS = useTLS
	}

	if useSSL, ok := smtpConfig["use_ssl"].(bool); ok {
		provider.useSSL = useSSL
	}

	// IMAP configuration (optional for receiving emails)
	if imapConfig, ok := options["imap"].(map[string]interface{}); ok {
		// IMAP is configured, parse it
		if imapHost, ok := imapConfig["host"].(string); ok {
			provider.imapHost = imapHost
		} else {
			// Default to same host as SMTP if not specified
			provider.imapHost = provider.host
		}

		if imapPort, ok := imapConfig["port"]; ok {
			switch p := imapPort.(type) {
			case int:
				provider.imapPort = p
			case float64:
				provider.imapPort = int(p)
			case string:
				var err error
				provider.imapPort, err = strconv.Atoi(p)
				if err != nil {
					return nil, fmt.Errorf("invalid IMAP port: %s", p)
				}
			default:
				return nil, fmt.Errorf("invalid IMAP port type")
			}
		} else {
			provider.imapPort = 993 // Default IMAP SSL port
		}

		if imapUsername, ok := imapConfig["username"].(string); ok {
			provider.imapUsername = imapUsername
		} else {
			// Default to same username as SMTP if not specified
			provider.imapUsername = provider.username
		}

		if imapPassword, ok := imapConfig["password"].(string); ok {
			provider.imapPassword = imapPassword
		} else {
			// Default to same password as SMTP if not specified
			provider.imapPassword = provider.password
		}

		if imapUseSSL, ok := imapConfig["use_ssl"].(bool); ok {
			provider.imapUseSSL = imapUseSSL
		} else {
			provider.imapUseSSL = true // Default to SSL for IMAP
		}

		if imapMailbox, ok := imapConfig["mailbox"].(string); ok {
			provider.imapMailbox = imapMailbox
		} else {
			provider.imapMailbox = "INBOX" // Default mailbox
		}
	} else {
		// IMAP not configured - this provider only supports sending
		provider.imapHost = ""
		provider.imapPort = 0
		provider.imapUsername = ""
		provider.imapPassword = ""
		provider.imapUseSSL = false
		provider.imapMailbox = ""
	}

	return provider, nil
}

// Send sends a message using SMTP
func (p *Provider) Send(ctx context.Context, message *types.Message) error {
	if message.Type != types.MessageTypeEmail {
		return fmt.Errorf("SMTP provider only supports email messages")
	}

	// Create message content
	content, err := p.buildMessage(message)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

	// Send the email
	return p.sendEmail(ctx, message.To, content)
}

// SendBatch sends multiple messages in batch
func (p *Provider) SendBatch(ctx context.Context, messages []*types.Message) error {
	for _, message := range messages {
		if err := p.Send(ctx, message); err != nil {
			return fmt.Errorf("failed to send message to %v: %w", message.To, err)
		}
	}
	return nil
}

// GetType returns the provider type
func (p *Provider) GetType() string {
	return "mailer"
}

// GetName returns the provider name
func (p *Provider) GetName() string {
	return p.config.Name
}

// GetPublicInfo returns public information about the provider
func (p *Provider) GetPublicInfo() types.ProviderPublicInfo {
	description := "SMTP email provider"
	if p.config.Description != "" {
		description = p.config.Description
	}

	return types.ProviderPublicInfo{
		Name:         p.config.Name,
		Type:         "mailer",
		Description:  description,
		Capabilities: []string{"email"},
		Features: types.Features{
			SupportsWebhooks:   false,
			SupportsReceiving:  p.SupportsReceiving(),
			SupportsTracking:   false,
			SupportsScheduling: false,
		},
	}
}

// Validate validates the provider configuration
func (p *Provider) Validate() error {
	if p.host == "" {
		return fmt.Errorf("host is required")
	}
	if p.port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	if p.username == "" {
		return fmt.Errorf("username is required")
	}
	if p.password == "" {
		return fmt.Errorf("password is required")
	}
	if p.from == "" {
		return fmt.Errorf("from address is required")
	}
	return nil
}

// TriggerWebhook processes webhook requests - not supported for SMTP
func (p *Provider) TriggerWebhook(c interface{}) (*types.Message, error) {
	return nil, fmt.Errorf("TriggerWebhook not supported for SMTP/mailer provider")
}

// Close closes the provider connection (no-op for SMTP)
func (p *Provider) Close() error {
	return nil
}

// SupportsReceiving returns true if this provider supports receiving emails via IMAP
func (p *Provider) SupportsReceiving() bool {
	return p.imapHost != "" && p.imapPort > 0
}

// buildMessage builds the email message content
func (p *Provider) buildMessage(message *types.Message) (string, error) {
	var content strings.Builder

	// From header
	from := message.From
	if from == "" {
		from = p.from
	}
	content.WriteString(fmt.Sprintf("From: %s\r\n", from))

	// To header
	content.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(message.To, ", ")))

	// Subject header
	content.WriteString(fmt.Sprintf("Subject: %s\r\n", message.Subject))

	// Additional headers
	if message.Headers != nil {
		for key, value := range message.Headers {
			content.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
		}
	}

	// MIME headers for HTML content
	if message.HTML != "" {
		content.WriteString("MIME-Version: 1.0\r\n")
		if message.Body != "" {
			// Multipart message with both text and HTML
			content.WriteString("Content-Type: multipart/alternative; boundary=\"boundary123\"\r\n")
			content.WriteString("\r\n")
			content.WriteString("--boundary123\r\n")
			content.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
			content.WriteString("\r\n")
			content.WriteString(message.Body)
			content.WriteString("\r\n--boundary123\r\n")
			content.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			content.WriteString("\r\n")
			content.WriteString(message.HTML)
			content.WriteString("\r\n--boundary123--\r\n")
		} else {
			// HTML only
			content.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
			content.WriteString("\r\n")
			content.WriteString(message.HTML)
		}
	} else {
		// Plain text only
		content.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		content.WriteString("\r\n")
		content.WriteString(message.Body)
	}

	return content.String(), nil
}

// extractEmailAddress extracts the email address from a string that may contain display name
// e.g., "John Doe <john@example.com>" -> "john@example.com"
func extractEmailAddress(address string) string {
	// Regular expression to match email addresses in angle brackets
	re := regexp.MustCompile(`<([^>]+)>`)
	matches := re.FindStringSubmatch(address)
	if len(matches) > 1 {
		return matches[1]
	}
	// If no angle brackets, assume the whole string is the email address
	return strings.TrimSpace(address)
}

// sendEmail sends the email using SMTP
func (p *Provider) sendEmail(ctx context.Context, to []string, content string) error {
	addr := fmt.Sprintf("%s:%d", p.host, p.port)

	// Create auth
	auth := smtp.PlainAuth("", p.username, p.password, p.host)

	// Send email with context support
	if p.useSSL {
		// Use SSL/TLS connection
		return p.sendWithTLS(ctx, addr, auth, to, content)
	}
	// Use standard SMTP with STARTTLS and context support
	return p.sendWithContext(ctx, addr, auth, to, content)
}

// sendWithContext sends email using standard SMTP with context support
func (p *Provider) sendWithContext(ctx context.Context, addr string, auth smtp.Auth, to []string, content string) error {
	// Create a dialer with timeout from context
	d := &net.Dialer{
		Timeout: 30 * time.Second,
	}

	// Connect with context
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, p.host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Quit()

	// Start TLS if supported
	if p.useTLS {
		tlsConfig := &tls.Config{
			ServerName: p.host,
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	// Set sender (extract pure email address from potentially formatted from address)
	fromEmail := extractEmailAddress(p.from)
	if err = client.Mail(fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to write message content: %w", err)
	}

	return w.Close()
}

// sendWithTLS sends email with explicit TLS connection
func (p *Provider) sendWithTLS(ctx context.Context, addr string, auth smtp.Auth, to []string, content string) error {
	// Create TLS connection with context support
	tlsConfig := &tls.Config{
		ServerName:         p.host,
		InsecureSkipVerify: false,
	}

	// Use dialer with context for TLS connection
	d := &net.Dialer{
		Timeout: 30 * time.Second,
	}

	conn, err := tls.DialWithDialer(d, "tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to create TLS connection: %w", err)
	}
	defer conn.Close()

	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return fmt.Errorf("connection cancelled: %w", ctx.Err())
	default:
	}

	// Create SMTP client
	client, err := smtp.NewClient(conn, p.host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Authenticate
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender (extract pure email address from potentially formatted from address)
	fromEmail := extractEmailAddress(p.from)
	if err := client.Mail(fromEmail); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send data
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = writer.Write([]byte(content))
	if err != nil {
		writer.Close()
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	return client.Quit()
}
