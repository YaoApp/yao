//go:build unit

package hostexec

import (
	"context"
	"strings"
	"testing"

	pb "github.com/yaoapp/yao/tai/hostexec/pb"
)

func TestLocalBidiStream_Echo(t *testing.T) {
	c := NewLocalClient("", Policy{FullAccess: true})
	ctx := context.Background()
	bidi, err := c.ExecStreamBidi(ctx)
	if err != nil {
		t.Fatalf("ExecStreamBidi: %v", err)
	}

	err = bidi.Send(&pb.ExecInput{Payload: &pb.ExecInput_Start{Start: &pb.ExecRequest{
		Command: "echo",
		Args:    []string{"hello-bidi"},
	}}})
	if err != nil {
		t.Fatalf("Send start: %v", err)
	}

	var stdout strings.Builder
	for {
		msg, err := bidi.Recv()
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		if msg.Done {
			if msg.ExitCode != 0 {
				t.Errorf("exit code = %d, want 0", msg.ExitCode)
			}
			break
		}
		if msg.Stream == pb.ExecOutput_STDOUT {
			stdout.Write(msg.Data)
		}
	}

	if got := strings.TrimSpace(stdout.String()); got != "hello-bidi" {
		t.Errorf("stdout = %q, want %q", got, "hello-bidi")
	}
}

func TestLocalBidiStream_StdinWrite(t *testing.T) {
	c := NewLocalClient("", Policy{FullAccess: true})
	ctx := context.Background()
	bidi, err := c.ExecStreamBidi(ctx)
	if err != nil {
		t.Fatalf("ExecStreamBidi: %v", err)
	}

	err = bidi.Send(&pb.ExecInput{Payload: &pb.ExecInput_Start{Start: &pb.ExecRequest{
		Command: "cat",
	}}})
	if err != nil {
		t.Fatalf("Send start: %v", err)
	}

	err = bidi.Send(&pb.ExecInput{Payload: &pb.ExecInput_StdinData{StdinData: []byte("from-stdin\n")}})
	if err != nil {
		t.Fatalf("Send stdin: %v", err)
	}

	err = bidi.Send(&pb.ExecInput{Payload: &pb.ExecInput_StdinEof{StdinEof: true}})
	if err != nil {
		t.Fatalf("Send eof: %v", err)
	}

	var stdout strings.Builder
	for {
		msg, err := bidi.Recv()
		if err != nil {
			t.Fatalf("Recv: %v", err)
		}
		if msg.Done {
			if msg.ExitCode != 0 {
				t.Errorf("exit code = %d", msg.ExitCode)
			}
			break
		}
		if msg.Stream == pb.ExecOutput_STDOUT {
			stdout.Write(msg.Data)
		}
	}

	if got := strings.TrimSpace(stdout.String()); got != "from-stdin" {
		t.Errorf("stdout = %q, want %q", got, "from-stdin")
	}
}
