package weixin

const (
	HeaderAuthType = "AuthorizationType"
	HeaderAuthVal  = "ilink_bot_token"
)

const SessionExpiredErrCode = -14

const (
	ItemTypeText  = 1
	ItemTypeImage = 2
	ItemTypeVoice = 3
	ItemTypeFile  = 4
	ItemTypeVideo = 5
)

const (
	MessageTypeNone = 0
	MessageTypeUser = 1
	MessageTypeBot  = 2
)

const (
	MessageStateNew        = 0
	MessageStateGenerating = 1
	MessageStateFinish     = 2
)

const (
	TypingStatusTyping = 1
	TypingStatusCancel = 2
)

type BaseInfo struct {
	ChannelVersion string `json:"channel_version,omitempty"`
}

type TextItem struct {
	Text string `json:"text,omitempty"`
}

type CDNMedia struct {
	EncryptQueryParam string `json:"encrypt_query_param"`
	AesKey            string `json:"aes_key"`
	EncryptType       int    `json:"encrypt_type,omitempty"`
}

type UploadedFileInfo struct {
	Filekey            string `json:"filekey"`
	DownloadParam      string `json:"download_encrypted_query_param"`
	AesKeyHex          string `json:"aeskey"`
	FileSize           int    `json:"file_size"`
	FileSizeCiphertext int    `json:"file_size_ciphertext"`
}

type GetUploadUrlResp struct {
	Ret              int    `json:"ret"`
	ErrCode          int    `json:"errcode"`
	ErrMsg           string `json:"errmsg,omitempty"`
	UploadParam      string `json:"upload_param"`
	ThumbUploadParam string `json:"thumb_upload_param,omitempty"`
}

type ImageItem struct {
	AesKey     string    `json:"aeskey,omitempty"`
	Media      *CDNMedia `json:"media,omitempty"`
	ThumbMedia *CDNMedia `json:"thumb_media,omitempty"`
	MidSize    int       `json:"mid_size,omitempty"`
	HdSize     int       `json:"hd_size,omitempty"`
}

type VoiceItem struct {
	Media      *CDNMedia `json:"media,omitempty"`
	EncodeType int       `json:"encode_type,omitempty"`
	SampleRate int       `json:"sample_rate,omitempty"`
	PlayTime   int       `json:"playtime,omitempty"`
	Text       string    `json:"text,omitempty"`
}

type FileItem struct {
	FileName string    `json:"file_name,omitempty"`
	Media    *CDNMedia `json:"media,omitempty"`
	Len      string    `json:"len,omitempty"`
}

type VideoItem struct {
	Media      *CDNMedia `json:"media,omitempty"`
	ThumbMedia *CDNMedia `json:"thumb_media,omitempty"`
	VideoSize  int       `json:"video_size,omitempty"`
}

type RefMessage struct {
	MessageItem *MsgItem `json:"message_item,omitempty"`
	Title       string   `json:"title,omitempty"`
}

type MsgItem struct {
	Type      int         `json:"type"`
	TextItem  *TextItem   `json:"text_item,omitempty"`
	ImageItem *ImageItem  `json:"image_item,omitempty"`
	VoiceItem *VoiceItem  `json:"voice_item,omitempty"`
	FileItem  *FileItem   `json:"file_item,omitempty"`
	VideoItem *VideoItem  `json:"video_item,omitempty"`
	RefMsg    *RefMessage `json:"ref_msg,omitempty"`
}

type WeixinMessage struct {
	Seq          int64     `json:"seq,omitempty"`
	MessageID    int64     `json:"message_id,omitempty"`
	FromUserID   string    `json:"from_user_id"`
	ToUserID     string    `json:"to_user_id"`
	ClientID     string    `json:"client_id,omitempty"`
	SessionID    string    `json:"session_id,omitempty"`
	GroupID      string    `json:"group_id,omitempty"`
	MessageType  int       `json:"message_type,omitempty"`
	MessageState int       `json:"message_state,omitempty"`
	ContextToken string    `json:"context_token"`
	CreateTimeMs int64     `json:"create_time_ms"`
	UpdateTimeMs int64     `json:"update_time_ms,omitempty"`
	DeleteTimeMs int64     `json:"delete_time_ms,omitempty"`
	ItemList     []MsgItem `json:"item_list"`
}

type GetUpdatesResp struct {
	Ret                  int             `json:"ret"`
	ErrCode              int             `json:"errcode"`
	ErrMsg               string          `json:"errmsg"`
	Msgs                 []WeixinMessage `json:"msgs"`
	GetUpdatesBuf        string          `json:"get_updates_buf"`
	LongPollingTimeoutMs int             `json:"longpolling_timeout_ms"`
}

type SendMessageReq struct {
	Msg      *WeixinMessage `json:"msg"`
	BaseInfo *BaseInfo      `json:"base_info,omitempty"`
}

type SendTypingReq struct {
	IlinkUserID  string    `json:"ilink_user_id"`
	TypingTicket string    `json:"typing_ticket"`
	Status       int       `json:"status"`
	BaseInfo     *BaseInfo `json:"base_info,omitempty"`
}

type GetConfigResp struct {
	Ret          int    `json:"ret"`
	ErrMsg       string `json:"errmsg"`
	TypingTicket string `json:"typing_ticket"`
}

type QRCodeResp struct {
	QRCode           string `json:"qrcode"`
	QRCodeImgContent string `json:"qrcode_img_content"`
}

type QRStatusResp struct {
	Status     string `json:"status"`
	BotToken   string `json:"bot_token"`
	IlinkBotID string `json:"ilink_bot_id"`
	BaseURL    string `json:"baseurl"`
	UserID     string `json:"ilink_user_id"`
}
