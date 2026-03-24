package api

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	weixinapi "github.com/yaoapp/yao/integrations/weixin"
)

const (
	qrSessionTTL      = 5 * time.Minute
	maxQRRefreshCount = 3
)

type qrSession struct {
	qrcode    string
	apiHost   string
	startedAt time.Time
	refreshes int
}

var (
	qrSessions   = make(map[string]*qrSession)
	qrSessionsMu sync.Mutex
)

// WeixinQRCodeCreate creates a new QR code session for WeChat login.
// Returns the session key and QR code URL.
func WeixinQRCodeCreate(apiHost string) (sessionKey, qrcodeURL, qrcodeImg string, err error) {
	qrcode, qrcodeImgContent, err := weixinapi.GetQRCode(context.Background(), apiHost)
	if err != nil {
		return "", "", "", fmt.Errorf("get QR code: %w", err)
	}

	sessionKey = uuid.New().String()
	qrSessionsMu.Lock()
	qrSessions[sessionKey] = &qrSession{
		qrcode:    qrcode,
		apiHost:   apiHost,
		startedAt: time.Now(),
	}
	qrSessionsMu.Unlock()

	return sessionKey, qrcode, qrcodeImgContent, nil
}

// WeixinQRCodePoll polls the QR code status for a given session.
func WeixinQRCodePoll(sessionKey string) (status, botToken, accountID, baseURL, userID string, err error) {
	qrSessionsMu.Lock()
	session, ok := qrSessions[sessionKey]
	if !ok {
		qrSessionsMu.Unlock()
		return "", "", "", "", "", fmt.Errorf("session not found: %s", sessionKey)
	}

	if time.Since(session.startedAt) > qrSessionTTL {
		if session.refreshes < maxQRRefreshCount {
			session.refreshes++
			session.startedAt = time.Now()
			apiHost := session.apiHost
			qrSessionsMu.Unlock()

			newQR, _, refreshErr := weixinapi.GetQRCode(context.Background(), apiHost)
			if refreshErr != nil {
				qrSessionsMu.Lock()
				delete(qrSessions, sessionKey)
				qrSessionsMu.Unlock()
				return "expired", "", "", "", "", nil
			}

			qrSessionsMu.Lock()
			if s, ok := qrSessions[sessionKey]; ok {
				s.qrcode = newQR
			}
			qrSessionsMu.Unlock()
			return "refreshed", "", "", "", "", nil
		}
		delete(qrSessions, sessionKey)
		qrSessionsMu.Unlock()
		return "expired", "", "", "", "", nil
	}

	qrcode := session.qrcode
	apiHost := session.apiHost
	qrSessionsMu.Unlock()

	resp, err := weixinapi.PollQRStatus(context.Background(), apiHost, qrcode)
	if err != nil {
		return "wait", "", "", "", "", nil
	}

	if resp.Status == "confirmed" {
		qrSessionsMu.Lock()
		delete(qrSessions, sessionKey)
		qrSessionsMu.Unlock()
		return resp.Status, resp.BotToken, resp.IlinkBotID, resp.BaseURL, resp.UserID, nil
	}

	return resp.Status, "", "", "", "", nil
}
