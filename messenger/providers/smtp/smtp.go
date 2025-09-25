package smtp

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/yaoapp/yao/messenger/types"
)

// SMTPProvider implements the Provider interface for SMTP email sending
type SMTPProvider struct {
	config   types.ProviderConfig
	host     string
	port     int
	username string
	password string
	from     string
	useTLS   bool
	useSSL   bool
}

// NewSMTPProvider creates a new SMTP provider
func NewSMTPProvider(config types.ProviderConfig) (*SMTPProvider, error) {
	provider := &SMTPProvider{
		config: config,
		useTLS: true, // Default to TLS
	}

	// Extract options
	options := config.Options
	if options == nil {
		return nil, fmt.Errorf("SMTP provider requires options")
	}

	// Required options
	if host, ok := options["host"].(string); ok {
		provider.host = host
	} else {
		return nil, fmt.Errorf("SMTP provider requires 'host' option")
	}

	if port, ok := options["port"]; ok {
		switch p := port.(type) {
		case int:
			provider.port = p
		case float64:
			provider.port = int(p)
		case string:
			var err error
			provider.port, err = strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", p)
			}
		default:
			return nil, fmt.Errorf("invalid port type")
		}
	} else {
		provider.port = 587 // Default SMTP port
	}

	if username, ok := options["username"].(string); ok {
		provider.username = username
	} else {
		return nil, fmt.Errorf("SMTP provider requires 'username' option")
	}

	if password, ok := options["password"].(string); ok {
		provider.password = password
	} else {
		return nil, fmt.Errorf("SMTP provider requires 'password' option")
	}

	if from, ok := options["from"].(string); ok {
		provider.from = from
	} else {
		return nil, fmt.Errorf("SMTP provider requires 'from' option")
	}

	// Optional options
	if useTLS, ok := options["use_tls"].(bool); ok {
		provider.useTLS = useTLS
	}

	if useSSL, ok := options["use_ssl"].(bool); ok {
		provider.useSSL = useSSL
	}

	return provider, nil
}

// Send sends a message using SMTP
func (p *SMTPProvider) Send(message *types.Message) error {
	if message.Type != types.MessageTypeEmail {
		return fmt.Errorf("SMTP provider only supports email messages")
	}

	// Create message content
	content, err := p.buildMessage(message)
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

	// Send the email
	return p.sendEmail(message.To, content)
}

// SendBatch sends multiple messages in batch
func (p *SMTPProvider) SendBatch(messages []*types.Message) error {
	for _, message := range messages {
		if err := p.Send(message); err != nil {
			return fmt.Errorf("failed to send message to %v: %w", message.To, err)
		}
	}
	return nil
}

// GetType returns the provider type
func (p *SMTPProvider) GetType() string {
	return "smtp"
}

// GetName returns the provider name
func (p *SMTPProvider) GetName() string {
	return p.config.Name
}

// Validate validates the provider configuration
func (p *SMTPProvider) Validate() error {
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

// Close closes the provider connection (no-op for SMTP)
func (p *SMTPProvider) Close() error {
	return nil
}

// buildMessage builds the email message content
func (p *SMTPProvider) buildMessage(message *types.Message) (string, error) {
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

// sendEmail sends the email using SMTP
func (p *SMTPProvider) sendEmail(to []string, content string) error {
	addr := fmt.Sprintf("%s:%d", p.host, p.port)

	// Create auth
	auth := smtp.PlainAuth("", p.username, p.password, p.host)

	// Send email
	if p.useSSL {
		// Use SSL/TLS connection
		return p.sendWithTLS(addr, auth, to, content)
	} else {
		// Use standard SMTP with STARTTLS
		return smtp.SendMail(addr, auth, p.from, to, []byte(content))
	}
}

// sendWithTLS sends email with explicit TLS connection
func (p *SMTPProvider) sendWithTLS(addr string, auth smtp.Auth, to []string, content string) error {
	// Create TLS connection
	tlsConfig := &tls.Config{
		ServerName:         p.host,
		InsecureSkipVerify: false,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to create TLS connection: %w", err)
	}
	defer conn.Close()

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

	// Set sender
	if err := client.Mail(p.from); err != nil {
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
