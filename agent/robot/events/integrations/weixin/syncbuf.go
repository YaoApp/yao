package weixin

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/yaoapp/gou/application"
)

type syncBufData struct {
	GetUpdatesBuf string `json:"get_updates_buf"`
}

func syncBufPath(accountID string) string {
	root := application.App.Root()
	return filepath.Join(root, "data", "weixin", accountID+".sync.json")
}

func loadSyncBuf(accountID string) string {
	p := syncBufPath(accountID)
	data, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	var buf syncBufData
	if err := json.Unmarshal(data, &buf); err != nil {
		return ""
	}
	return buf.GetUpdatesBuf
}

func saveSyncBuf(accountID, syncBuf string) {
	p := syncBufPath(accountID)
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Error("weixin: mkdir for syncbuf: %v", err)
		return
	}
	data, _ := json.Marshal(syncBufData{GetUpdatesBuf: syncBuf})
	if err := os.WriteFile(p, data, 0644); err != nil {
		log.Error("weixin: write syncbuf: %v", err)
	}
}
