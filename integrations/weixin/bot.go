package weixin

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/yaoapp/kun/log"
)

const defaultBaseURL = "https://ilinkai.weixin.qq.com"
const defaultCDNBaseURL = "https://novac2c.cdn.weixin.qq.com/c2c"
const channelVersion = "1.0.0"

const (
	UploadMediaImage = 1
	UploadMediaVideo = 2
	UploadMediaFile  = 3
	UploadMediaVoice = 4
)

const cdnUploadMaxRetries = 3

type Bot struct {
	token      string
	baseURL    string
	cdnBaseURL string
	httpClient *http.Client
}

func NewBot(token, baseURL, cdnBaseURL string) *Bot {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if cdnBaseURL == "" {
		cdnBaseURL = defaultCDNBaseURL
	}
	return &Bot{
		token:      token,
		baseURL:    strings.TrimRight(baseURL, "/"),
		cdnBaseURL: strings.TrimRight(cdnBaseURL, "/"),
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (b *Bot) Token() string      { return b.token }
func (b *Bot) BaseURL() string    { return b.baseURL }
func (b *Bot) CDNBaseURL() string { return b.cdnBaseURL }
func DefaultBaseURL() string      { return defaultBaseURL }
func DefaultCDNBaseURL() string   { return defaultCDNBaseURL }

func (b *Bot) GetUpdates(ctx context.Context, syncBuf string, timeoutMs int) (*GetUpdatesResp, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"get_updates_buf": syncBuf,
		"base_info":       BaseInfo{ChannelVersion: channelVersion},
	})

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs+5000)*time.Millisecond)
	defer cancel()

	raw, err := b.post(reqCtx, "ilink/bot/getupdates", body)
	if err != nil {
		if reqCtx.Err() != nil {
			return &GetUpdatesResp{GetUpdatesBuf: syncBuf}, nil
		}
		return nil, err
	}

	var resp GetUpdatesResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("weixin GetUpdates unmarshal: %w", err)
	}
	return &resp, nil
}

func (b *Bot) SendMessage(ctx context.Context, toUserID, contextToken, text string) error {
	if contextToken == "" {
		return fmt.Errorf("weixin SendMessage: contextToken is required for to=%s", toUserID)
	}
	clientID := randomClientID()
	req := map[string]interface{}{
		"msg": map[string]interface{}{
			"from_user_id":  "",
			"to_user_id":    toUserID,
			"client_id":     clientID,
			"message_type":  MessageTypeBot,
			"message_state": MessageStateFinish,
			"context_token": contextToken,
			"item_list": []map[string]interface{}{
				{
					"type":      ItemTypeText,
					"text_item": map[string]string{"text": text},
				},
			},
		},
		"base_info": BaseInfo{ChannelVersion: channelVersion},
	}
	body, _ := json.Marshal(req)
	_, err := b.post(ctx, "ilink/bot/sendmessage", body)
	return err
}

func (b *Bot) SendImageMessage(ctx context.Context, toUserID, contextToken string, uploaded *UploadedFileInfo) error {
	return b.sendMediaMessage(ctx, toUserID, contextToken, MsgItem{
		Type: ItemTypeImage,
		ImageItem: &ImageItem{
			Media: &CDNMedia{
				EncryptQueryParam: uploaded.DownloadParam,
				AesKey:            base64.StdEncoding.EncodeToString([]byte(uploaded.AesKeyHex)),
				EncryptType:       1,
			},
			MidSize: uploaded.FileSizeCiphertext,
		},
	})
}

func (b *Bot) SendVideoMessage(ctx context.Context, toUserID, contextToken string, uploaded *UploadedFileInfo) error {
	return b.sendMediaMessage(ctx, toUserID, contextToken, MsgItem{
		Type: ItemTypeVideo,
		VideoItem: &VideoItem{
			Media: &CDNMedia{
				EncryptQueryParam: uploaded.DownloadParam,
				AesKey:            base64.StdEncoding.EncodeToString([]byte(uploaded.AesKeyHex)),
				EncryptType:       1,
			},
			VideoSize: uploaded.FileSizeCiphertext,
		},
	})
}

func (b *Bot) SendFileMessage(ctx context.Context, toUserID, contextToken, fileName string, uploaded *UploadedFileInfo) error {
	return b.sendMediaMessage(ctx, toUserID, contextToken, MsgItem{
		Type: ItemTypeFile,
		FileItem: &FileItem{
			FileName: fileName,
			Media: &CDNMedia{
				EncryptQueryParam: uploaded.DownloadParam,
				AesKey:            base64.StdEncoding.EncodeToString([]byte(uploaded.AesKeyHex)),
				EncryptType:       1,
			},
			Len: strconv.Itoa(uploaded.FileSize),
		},
	})
}

// SendVoiceMessage sends a voice message with a bubble UI.
// TODO(weixin-voice): The voice bubble displays correctly (with playtime) but
// audio playback does not work — the WeChat client reports "message still
// downloading". This affects all formats tested (SILK, Speex, OGG, MP3) and
// even echoing back an inbound voice's CDN reference verbatim. The iLink Bot
// API likely does not yet fully support outbound voice playback. For now,
// callers should fall back to SendFileMessage for audio attachments until
// WeChat officially supports voice playback via iLink Bot.
func (b *Bot) SendVoiceMessage(ctx context.Context, toUserID, contextToken string, uploaded *UploadedFileInfo, playtimeMs, sampleRate int) error {
	item := MsgItem{
		Type: ItemTypeVoice,
		VoiceItem: &VoiceItem{
			Media: &CDNMedia{
				EncryptQueryParam: uploaded.DownloadParam,
				AesKey:            base64.StdEncoding.EncodeToString([]byte(uploaded.AesKeyHex)),
			},
			PlayTime:   playtimeMs,
			SampleRate: sampleRate,
		},
	}
	return b.sendMediaMessage(ctx, toUserID, contextToken, item)
}

func (b *Bot) sendMediaMessage(ctx context.Context, toUserID, contextToken string, item MsgItem) error {
	if contextToken == "" {
		return fmt.Errorf("weixin sendMediaMessage: contextToken is required for to=%s", toUserID)
	}
	clientID := randomClientID()
	req := map[string]interface{}{
		"msg": map[string]interface{}{
			"from_user_id":  "",
			"to_user_id":    toUserID,
			"client_id":     clientID,
			"message_type":  MessageTypeBot,
			"message_state": MessageStateFinish,
			"context_token": contextToken,
			"item_list":     []MsgItem{item},
		},
		"base_info": BaseInfo{ChannelVersion: channelVersion},
	}
	body, _ := json.Marshal(req)
	_, err := b.post(ctx, "ilink/bot/sendmessage", body)
	return err
}

func (b *Bot) UploadMedia(ctx context.Context, plaintext []byte, toUserID string, mediaType int) (*UploadedFileInfo, error) {
	rawsize := len(plaintext)
	hash := md5.Sum(plaintext)
	rawfilemd5 := hex.EncodeToString(hash[:])
	filesize := aesEcbPaddedSize(rawsize)

	var filekeyBuf [16]byte
	rand.Read(filekeyBuf[:])
	filekey := hex.EncodeToString(filekeyBuf[:])

	var aeskeyBuf [16]byte
	rand.Read(aeskeyBuf[:])
	aeskeyHex := hex.EncodeToString(aeskeyBuf[:])

	uploadReq, _ := json.Marshal(map[string]interface{}{
		"filekey":       filekey,
		"media_type":    mediaType,
		"to_user_id":    toUserID,
		"rawsize":       rawsize,
		"rawfilemd5":    rawfilemd5,
		"filesize":      filesize,
		"no_need_thumb": true,
		"aeskey":        aeskeyHex,
		"base_info":     BaseInfo{ChannelVersion: channelVersion},
	})

	log.Info("[weixin:upload] getuploadurl request: media_type=%d to_user_id=%s rawsize=%d filesize=%d filekey=%s md5=%s",
		mediaType, toUserID, rawsize, filesize, filekey, rawfilemd5)

	raw, err := b.post(ctx, "ilink/bot/getuploadurl", uploadReq)
	if err != nil {
		return nil, fmt.Errorf("getUploadUrl: %w", err)
	}
	var uploadResp GetUploadUrlResp
	if err := json.Unmarshal(raw, &uploadResp); err != nil {
		return nil, fmt.Errorf("getUploadUrl unmarshal: %w (body: %s)", err, string(raw))
	}

	log.Info("[weixin:upload] getuploadurl response: ret=%d errcode=%d errmsg=%q upload_param_len=%d",
		uploadResp.Ret, uploadResp.ErrCode, uploadResp.ErrMsg, len(uploadResp.UploadParam))

	if uploadResp.Ret != 0 || uploadResp.ErrCode != 0 {
		return nil, fmt.Errorf("getUploadUrl: ret=%d errcode=%d errmsg=%q media_type=%d to_user_id=%s rawsize=%d filesize=%d rawfilemd5=%s",
			uploadResp.Ret, uploadResp.ErrCode, uploadResp.ErrMsg, mediaType, toUserID, rawsize, filesize, rawfilemd5)
	}
	if uploadResp.UploadParam == "" {
		return nil, fmt.Errorf("getUploadUrl: empty upload_param (body: %s)", string(raw))
	}

	ciphertext := encryptAES128ECB(plaintext, aeskeyBuf[:])
	log.Info("[weixin:upload] CDN uploading: ciphertext_len=%d filekey=%s", len(ciphertext), filekey)

	downloadParam, err := b.uploadBufferToCDN(ctx, ciphertext, uploadResp.UploadParam, filekey)
	if err != nil {
		return nil, fmt.Errorf("CDN upload: %w", err)
	}
	log.Info("[weixin:upload] CDN success: download_param_len=%d", len(downloadParam))

	return &UploadedFileInfo{
		Filekey:            filekey,
		DownloadParam:      downloadParam,
		AesKeyHex:          aeskeyHex,
		FileSize:           rawsize,
		FileSizeCiphertext: filesize,
	}, nil
}

func (b *Bot) uploadBufferToCDN(ctx context.Context, ciphertext []byte, uploadParam, filekey string) (string, error) {
	cdnURL := b.cdnBaseURL + "/upload?encrypted_query_param=" +
		url.QueryEscape(uploadParam) + "&filekey=" + url.QueryEscape(filekey)

	var lastErr error
	for attempt := 1; attempt <= cdnUploadMaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cdnURL, bytes.NewReader(ciphertext))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/octet-stream")

		resp, err := b.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			resp.Body.Close()
			return "", fmt.Errorf("CDN upload client error %d", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("CDN upload server error %d", resp.StatusCode)
			continue
		}

		downloadParam := resp.Header.Get("x-encrypted-param")
		resp.Body.Close()
		if downloadParam == "" {
			lastErr = fmt.Errorf("CDN response missing x-encrypted-param")
			continue
		}
		return downloadParam, nil
	}
	return "", fmt.Errorf("CDN upload failed after %d attempts: %w", cdnUploadMaxRetries, lastErr)
}

func randomClientID() string {
	var buf [8]byte
	rand.Read(buf[:])
	return fmt.Sprintf("yao-weixin-%x", buf[:])
}

func (b *Bot) SendTyping(ctx context.Context, toUserID, typingTicket string, status int) error {
	body, _ := json.Marshal(map[string]interface{}{
		"ilink_user_id": toUserID,
		"typing_ticket": typingTicket,
		"status":        status,
		"base_info":     BaseInfo{ChannelVersion: channelVersion},
	})
	_, err := b.post(ctx, "ilink/bot/sendtyping", body)
	return err
}

func (b *Bot) GetConfig(ctx context.Context, ilinkUserID, contextToken string) (string, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"ilink_user_id": ilinkUserID,
		"context_token": contextToken,
		"base_info":     BaseInfo{ChannelVersion: channelVersion},
	})
	raw, err := b.post(ctx, "ilink/bot/getconfig", body)
	if err != nil {
		return "", err
	}
	var resp GetConfigResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", fmt.Errorf("weixin GetConfig unmarshal: %w", err)
	}
	return resp.TypingTicket, nil
}

func (b *Bot) post(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
	reqURL := b.baseURL + "/" + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AuthorizationType", HeaderAuthVal)
	req.Header.Set("Authorization", "Bearer "+b.token)
	req.Header.Set("Content-Length", strconv.Itoa(len(body)))
	req.Header.Set("X-WECHAT-UIN", randomWechatUin())

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weixin %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("weixin %s read body: %w", endpoint, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weixin %s HTTP %d: %s", endpoint, resp.StatusCode, string(raw))
	}
	return raw, nil
}

func randomWechatUin() string {
	var buf [4]byte
	rand.Read(buf[:])
	n := binary.BigEndian.Uint32(buf[:])
	return base64.StdEncoding.EncodeToString([]byte(strconv.FormatUint(uint64(n), 10)))
}
