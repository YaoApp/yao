package weixin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultBotType = "3"

func GetQRCode(ctx context.Context, apiHost string) (qrcode, qrcodeImgURL string, err error) {
	if apiHost == "" {
		apiHost = defaultBaseURL
	}
	u := strings.TrimRight(apiHost, "/") + "/ilink/bot/get_bot_qrcode?bot_type=" + DefaultBotType

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", "", err
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("GetQRCode: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GetQRCode HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var r QRCodeResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", "", fmt.Errorf("GetQRCode unmarshal: %w", err)
	}
	return r.QRCode, r.QRCodeImgContent, nil
}

func PollQRStatus(ctx context.Context, apiHost, qrcode string) (*QRStatusResp, error) {
	if apiHost == "" {
		apiHost = defaultBaseURL
	}
	u := fmt.Sprintf("%s/ilink/bot/get_qrcode_status?qrcode=%s",
		strings.TrimRight(apiHost, "/"), qrcode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("iLink-App-ClientVersion", "1")

	client := &http.Client{Timeout: 35 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PollQRStatus: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PollQRStatus HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var r QRStatusResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("PollQRStatus unmarshal: %w", err)
	}
	return &r, nil
}
