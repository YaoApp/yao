package weixin

import (
	"path/filepath"
	"strings"
)

var mimeMap = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".svg":  "image/svg+xml",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
	".ogg":  "audio/ogg",
	".silk": "audio/silk",
	".amr":  "audio/amr",
	".mp4":  "video/mp4",
	".mov":  "video/quicktime",
	".avi":  "video/x-msvideo",
	".webm": "video/webm",
	".pdf":  "application/pdf",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":  "application/vnd.ms-powerpoint",
	".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".zip":  "application/zip",
	".gz":   "application/gzip",
	".tar":  "application/x-tar",
	".txt":  "text/plain",
	".csv":  "text/csv",
	".json": "application/json",
	".xml":  "application/xml",
}

func MimeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if m, ok := mimeMap[ext]; ok {
		return m
	}
	return "application/octet-stream"
}
