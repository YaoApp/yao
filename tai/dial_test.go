package tai

import (
	"testing"
)

func TestMaxGRPCMsgSize_500MB(t *testing.T) {
	want := 500 * 1024 * 1024
	if MaxGRPCMsgSizeExported != want {
		t.Errorf("maxGRPCMsgSize = %d, want %d (500 MB)", MaxGRPCMsgSizeExported, want)
	}
}

func TestDialGRPC_ReturnsConnection(t *testing.T) {
	conn, err := DialGRPC("passthrough:///127.0.0.1:0")
	if err != nil {
		t.Fatalf("dialGRPC returned error: %v", err)
	}
	defer conn.Close()

	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
}
