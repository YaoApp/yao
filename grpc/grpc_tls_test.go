package grpc

import (
	"path/filepath"
	"testing"
)

func TestResolveGRPCCertPath_Absolute(t *testing.T) {
	got := resolveGRPCCertPath("/absolute/path/cert.pem", "/root")
	if got != "/absolute/path/cert.pem" {
		t.Errorf("resolveGRPCCertPath absolute = %q, want /absolute/path/cert.pem", got)
	}
}

func TestResolveGRPCCertPath_Relative(t *testing.T) {
	got := resolveGRPCCertPath("grpc-cert.pem", "/app/data")
	want := filepath.Join("/app/data", "openapi", "certs", "grpc-cert.pem")
	if got != want {
		t.Errorf("resolveGRPCCertPath relative = %q, want %q", got, want)
	}
}

func TestResolveGRPCCertPath_Empty(t *testing.T) {
	got := resolveGRPCCertPath("", "/root")
	if got != "" {
		t.Errorf("resolveGRPCCertPath empty = %q, want empty", got)
	}
}
