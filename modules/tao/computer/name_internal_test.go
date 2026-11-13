package computer

import "testing"

func TestComputerNameFallbacks(t *testing.T) {
	if got := computerName(PurchaseRequest{}); got != "cloud-computer" {
		t.Fatalf("default=%q", got)
	}
	if got := computerName(PurchaseRequest{Name: "host"}); got != "host" {
		t.Fatalf("name=%q", got)
	}
	if got := computerName(PurchaseRequest{Name: "host", Computer: ComputerSpec{Name: "computer"}}); got != "computer" {
		t.Fatalf("computer name=%q", got)
	}
}
