package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/messenger/types"
)

// IMAP configuration is now integrated into the main Provider struct

// MailReceiver handles email receiving via IMAP
type MailReceiver struct {
	provider     *Provider
	client       *client.Client
	stopChan     chan bool
	msgHandler   func(*types.Message) error
	startTime    time.Time // Only process emails received after this time
	lastCheckUID uint32    // Track last processed UID to avoid duplicates
}

// Note: The Receive method has been removed as it's replaced by TriggerWebhook
// For direct IMAP email receiving, use StartMailReceiver

// StartMailReceiver starts an IMAP-based email receiver with polling or IDLE support
func (p *Provider) StartMailReceiver(ctx context.Context, handler func(*types.Message) error) error {
	// Check if this provider supports receiving
	if !p.SupportsReceiving() {
		return fmt.Errorf("provider does not support receiving: IMAP not configured")
	}
	receiver := &MailReceiver{
		provider:     p,
		stopChan:     make(chan bool),
		msgHandler:   handler,
		startTime:    time.Now(), // Only process emails received after this moment
		lastCheckUID: 0,
	}

	// Mailbox is already set in provider initialization with default "INBOX"

	// Start receiving emails (connection will be handled in startReceiving)
	// This will block until the receiver stops
	receiver.startReceiving(ctx)

	return nil
}

// connect establishes connection to IMAP server
func (r *MailReceiver) connect() error {
	var c *client.Client
	var err error

	addr := fmt.Sprintf("%s:%d", r.provider.imapHost, r.provider.imapPort)

	if r.provider.imapUseSSL {
		// Connect with SSL/TLS
		c, err = client.DialTLS(addr, &tls.Config{ServerName: r.provider.imapHost})
	} else {
		// Connect without SSL (can upgrade with STARTTLS)
		c, err = client.Dial(addr)
		if err == nil {
			// Try to upgrade to TLS if available
			if caps, err := c.Capability(); err == nil {
				if caps["STARTTLS"] {
					c.StartTLS(&tls.Config{ServerName: r.provider.imapHost})
				}
			}
		}
	}

	if err != nil {
		return err
	}

	// Login
	if err := c.Login(r.provider.imapUsername, r.provider.imapPassword); err != nil {
		c.Close()
		return err
	}

	r.client = c
	return nil
}

// reconnect re-establishes connection to IMAP server
func (r *MailReceiver) reconnect() error {
	// Close existing connection if any
	if r.client != nil {
		r.client.Close()
		r.client = nil
	}

	// Establish new connection
	return r.connect()
}

// startReceiving starts the email receiving loop with retry mechanism
func (r *MailReceiver) startReceiving(ctx context.Context) {
	defer func() {
		if r.client != nil {
			r.client.Close()
		}
	}()

	maxRetries := 5
	retryDelay := time.Second * 5

	for retry := 0; retry < maxRetries; retry++ {
		select {
		case <-ctx.Done():
			log.Info("[Messenger] Context cancelled, stopping email receiver")
			return
		case <-r.stopChan:
			log.Info("[Messenger] Stop signal received, stopping email receiver")
			return
		default:
		}

		// Reconnect if needed
		if r.client == nil || r.client.State() != imap.SelectedState {
			if err := r.reconnect(); err != nil {
				log.Error("[Messenger] Failed to reconnect to IMAP server: %v", err)
				if retry < maxRetries-1 {
					time.Sleep(retryDelay)
					retryDelay *= 2 // Exponential backoff
					continue
				}
				return
			}
		}

		// Select mailbox
		_, err := r.client.Select(r.provider.imapMailbox, false)
		if err != nil {
			log.Error("[Messenger] Failed to select mailbox %s: %v", r.provider.imapMailbox, err)
			if retry < maxRetries-1 {
				time.Sleep(retryDelay)
				retryDelay *= 2
				continue
			}
			return
		}

		// Check if server supports IDLE
		caps, err := r.client.Capability()
		if err == nil && caps["IDLE"] {
			r.receiveWithIdle(ctx)
		} else {
			r.receiveWithPolling(ctx)
		}

		// If we reach here, the receiving loop ended, try to reconnect
		time.Sleep(retryDelay)
		retryDelay *= 2
	}
}

// receiveWithIdle uses IMAP IDLE for real-time email monitoring
func (r *MailReceiver) receiveWithIdle(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		default:
			// Check connection state
			if r.client == nil || r.client.State() == imap.LogoutState {
				return
			}

			// Process initial messages before starting IDLE
			r.processNewMessages()

			// Start IDLE with periodic message checking
			stop := make(chan struct{})
			idleDone := make(chan error, 1)

			go func() {
				err := r.client.Idle(stop, nil)
				idleDone <- err
			}()

			// Wait for IDLE to end or stop signals
			// Use shorter IDLE periods to check for messages more frequently
			idleTimeout := time.After(10 * time.Second) // Check every 10 seconds

		idleLoop:
			for {
				select {
				case err := <-idleDone:
					if err != nil {
						return
					}
					// Process messages after IDLE ends
					r.processNewMessages()
					break idleLoop

				case <-idleTimeout:
					close(stop)
					// Wait for IDLE to actually end
					<-idleDone
					// Process messages after stopping IDLE
					r.processNewMessages()
					break idleLoop

				case <-ctx.Done():
					close(stop)
					return

				case <-r.stopChan:
					close(stop)
					return
				}
			}
		}
	}
}

// receiveWithPolling uses periodic polling for email monitoring
func (r *MailReceiver) receiveWithPolling(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Poll every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-ticker.C:
			// Check connection state
			if r.client == nil || r.client.State() == imap.LogoutState {
				return
			}

			r.processNewMessages()
		}
	}
}

// processNewMessages fetches and processes new messages
func (r *MailReceiver) processNewMessages() {
	// Search for messages - use UID-based filtering instead of time-based
	criteria := imap.NewSearchCriteria()

	// If we have a lastCheckUID, only search for messages with higher UIDs
	if r.lastCheckUID > 0 {
		criteria.Uid = new(imap.SeqSet)
		criteria.Uid.AddRange(r.lastCheckUID+1, 0) // From last+1 to end
	} else {
		// For the first run, search for messages from today to avoid processing thousands of old emails
		today := time.Now().Truncate(24 * time.Hour)
		criteria.Since = today
	}

	uids, err := r.client.UidSearch(criteria)
	if err != nil {
		log.Error("[Messenger] Failed to search for new messages: %v", err)
		return
	}

	if len(uids) == 0 {
		return // No new messages
	}

	// Fetch messages using UID
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	// Fetch with UID and more complete data
	fetchItems := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchUid,
		imap.FetchInternalDate,
		imap.FetchBodyStructure,
		"BODY[TEXT]", // Get message body text
	}

	go func() {
		done <- r.client.UidFetch(seqset, fetchItems, messages)
	}()

	// Process each message and track highest UID
	var maxUID uint32
	processedCount := 0

	for msg := range messages {
		if msg.Uid > maxUID {
			maxUID = msg.Uid
		}

		// For the first run (lastCheckUID == 0), only process messages received after start time
		// For subsequent runs, process all messages (they're already filtered by UID)
		shouldProcess := true
		if r.lastCheckUID == 0 {
			// Convert both times to UTC for proper comparison
			msgTimeUTC := msg.InternalDate.UTC()
			startTimeUTC := r.startTime.UTC()

			if msgTimeUTC.Before(startTimeUTC) {
				shouldProcess = false
			}
		}

		if shouldProcess {
			if err := r.processMessage(msg); err != nil {
				log.Error("[Messenger] Failed to process message UID %d: %v", msg.Uid, err)
			} else {
				processedCount++
			}
		}
	}

	// Update last check UID to the highest UID we've seen (even if not processed)
	if maxUID > r.lastCheckUID {
		r.lastCheckUID = maxUID
	}

	if err := <-done; err != nil {
		log.Error("[Messenger] Failed to fetch messages: %v", err)
		return
	}
}

// processMessage converts IMAP message to types.Message and calls handler
func (r *MailReceiver) processMessage(imapMsg *imap.Message) error {
	if imapMsg.Envelope == nil {
		return fmt.Errorf("message envelope is nil")
	}

	// Extract message body
	body, htmlBody := r.extractMessageBody(imapMsg)

	// Convert to types.Message
	msg := &types.Message{
		Type:    types.MessageTypeEmail,
		Subject: imapMsg.Envelope.Subject,
		From:    r.formatAddress(imapMsg.Envelope.From),
		To:      r.formatAddresses(imapMsg.Envelope.To),
		Body:    body,
		HTML:    htmlBody,
	}

	// Add comprehensive metadata
	msg.Metadata = map[string]interface{}{
		"uid":           imapMsg.Uid,
		"message_id":    imapMsg.Envelope.MessageId,
		"date":          imapMsg.Envelope.Date,
		"internal_date": imapMsg.InternalDate,
		"reply_to":      r.formatAddresses(imapMsg.Envelope.ReplyTo),
		"cc":            r.formatAddresses(imapMsg.Envelope.Cc),
		"bcc":           r.formatAddresses(imapMsg.Envelope.Bcc),
		"size":          imapMsg.Size,
		"flags":         imapMsg.Flags,
	}

	// Add headers if available
	if len(imapMsg.Envelope.InReplyTo) > 0 {
		msg.Metadata["in_reply_to"] = imapMsg.Envelope.InReplyTo
	}

	// Call the message handler
	if r.msgHandler != nil {
		return r.msgHandler(msg)
	}

	return nil
}

// formatAddress formats a single email address
func (r *MailReceiver) formatAddress(addrs []*imap.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	addr := addrs[0]
	if addr.PersonalName != "" {
		return fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName)
	}
	return fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
}

// formatAddresses formats multiple email addresses
func (r *MailReceiver) formatAddresses(addrs []*imap.Address) []string {
	result := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if addr.PersonalName != "" {
			result = append(result, fmt.Sprintf("%s <%s@%s>", addr.PersonalName, addr.MailboxName, addr.HostName))
		} else {
			result = append(result, fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName))
		}
	}
	return result
}

// Note: handleBounce, handleDelivery, and handleComplaint functions have been removed
// as they were only used by the deprecated Receive method.
// Webhook processing is now handled by TriggerWebhook method which is not implemented for SMTP providers.

// extractMessageBody extracts plain text and HTML body from IMAP message
func (r *MailReceiver) extractMessageBody(imapMsg *imap.Message) (plainText, htmlText string) {
	// Get the body from the message
	for _, body := range imapMsg.Body {
		if body == nil {
			continue
		}

		// Read the body content
		bodyBytes, err := io.ReadAll(body)
		if err != nil {
			continue
		}

		bodyStr := string(bodyBytes)

		// Try to parse as email message
		msg, err := mail.ReadMessage(strings.NewReader(bodyStr))
		if err != nil {
			// If parsing fails, treat as plain text
			plainText = bodyStr
			continue
		}

		// Get content type
		contentType := msg.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			// Default to plain text if parsing fails
			bodyContent, _ := io.ReadAll(msg.Body)
			plainText = string(bodyContent)
			continue
		}

		// Handle different content types
		switch {
		case strings.HasPrefix(mediaType, "text/plain"):
			bodyContent, _ := io.ReadAll(msg.Body)
			plainText = string(bodyContent)

		case strings.HasPrefix(mediaType, "text/html"):
			bodyContent, _ := io.ReadAll(msg.Body)
			htmlText = string(bodyContent)

		case strings.HasPrefix(mediaType, "multipart/"):
			// Handle multipart messages
			boundary := params["boundary"]
			if boundary != "" {
				plainText, htmlText = r.parseMultipartBody(msg.Body, boundary)
			}

		default:
			// For other types, try to read as plain text
			bodyContent, _ := io.ReadAll(msg.Body)
			plainText = string(bodyContent)
		}
	}

	// Clean up the extracted text
	plainText = strings.TrimSpace(plainText)
	htmlText = strings.TrimSpace(htmlText)

	return plainText, htmlText
}

// parseMultipartBody parses multipart email body
func (r *MailReceiver) parseMultipartBody(body io.Reader, boundary string) (plainText, htmlText string) {
	reader := multipart.NewReader(body, boundary)

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		// Get content type of this part
		contentType := part.Header.Get("Content-Type")
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			continue
		}

		// Read part content
		partContent, err := io.ReadAll(part)
		if err != nil {
			continue
		}

		content := string(partContent)

		// Assign content based on type
		switch {
		case strings.HasPrefix(mediaType, "text/plain"):
			if plainText == "" { // Use first plain text part
				plainText = content
			}
		case strings.HasPrefix(mediaType, "text/html"):
			if htmlText == "" { // Use first HTML part
				htmlText = content
			}
		}

		part.Close()
	}

	return plainText, htmlText
}

// Stop stops the email receiver
func (r *MailReceiver) Stop() {
	close(r.stopChan)
}
