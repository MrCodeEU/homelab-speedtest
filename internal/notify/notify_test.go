package notify

import (
	"testing"

	"github.com/user/homelab-speedtest/internal/config"
)

func TestNewNtfyService(t *testing.T) {
	cfg := config.NtfyConfig{
		Enabled: true,
		Server:  "https://ntfy.sh",
		Topic:   "test-topic",
	}

	svc := New(cfg)
	if svc == nil {
		t.Fatal("New returned nil")
	}

	if svc.Config.Server != "https://ntfy.sh" {
		t.Errorf("Expected server 'https://ntfy.sh', got '%s'", svc.Config.Server)
	}
}

func TestSendDisabled(t *testing.T) {
	cfg := config.NtfyConfig{
		Enabled: false,
	}

	svc := New(cfg)

	// Should return nil when disabled
	err := svc.Send("Test", "Message", "default")
	if err != nil {
		t.Errorf("Expected no error when disabled, got %v", err)
	}
}
