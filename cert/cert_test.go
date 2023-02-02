package cert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/ssl"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	test.Prepare(t, config.Conf)
	defer test.Clean()

	Load(config.Conf)
	check(t)
}

func check(t *testing.T) {
	ids := map[string]bool{}
	for id := range ssl.Certificates {
		ids[id] = true
	}
	assert.True(t, ids["cert.pem"])
	assert.True(t, ids["cert.key"])
	assert.True(t, ids["cert.pub"])
}

func TestProcessSign(t *testing.T) {
	Load(config.Conf)
	args := []interface{}{"hello world", "cert.key", "SHA256"}
	signature, err := process.New("ssl.Sign", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "EDHf3C9TXEk7y8LzIk5czLefXZyGxcMDVMcbNuBBegDkTqnPsRQnhFtNOgCdox8lI3MzLatwjoljoMY4Qk+sHGd5mAHMpiREa1gRFSVYpA2xvXZ3+KsfOHAdICQrfUdy59QaJGo6iGPNGG8PQOXHPTVNn6LMfryat9+f4l21DPAZiT0RyCUgFZE3/Qv8Z/6J4AsIXMSKZD6BGPPHUxGe7UBrXZvcR5dX25EiNjuH2OO38YJnDiTRVw14UI5fk/mQrwRdezj5tSKFCyHt912BZExXtkHISiYFNTZ/2RhOup5Xx6o3GvrEOdshrnN80Lwu1Aaju+lnZp13hDz4P6hU7w==", signature)
}

func TestProcessVerify(t *testing.T) {
	Load(config.Conf)
	signature := "EDHf3C9TXEk7y8LzIk5czLefXZyGxcMDVMcbNuBBegDkTqnPsRQnhFtNOgCdox8lI3MzLatwjoljoMY4Qk+sHGd5mAHMpiREa1gRFSVYpA2xvXZ3+KsfOHAdICQrfUdy59QaJGo6iGPNGG8PQOXHPTVNn6LMfryat9+f4l21DPAZiT0RyCUgFZE3/Qv8Z/6J4AsIXMSKZD6BGPPHUxGe7UBrXZvcR5dX25EiNjuH2OO38YJnDiTRVw14UI5fk/mQrwRdezj5tSKFCyHt912BZExXtkHISiYFNTZ/2RhOup5Xx6o3GvrEOdshrnN80Lwu1Aaju+lnZp13hDz4P6hU7w=="
	args := []interface{}{"hello world", signature, "cert.pem", "SHA256"}
	res, err := process.New("ssl.Verify", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, res.(bool))
}
