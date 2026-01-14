package notify

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/user/homelab-speedtest/internal/config"
)

type NtfyService struct {
	Config config.NtfyConfig
}

func New(cfg config.NtfyConfig) *NtfyService {
	return &NtfyService{Config: cfg}
}

func (n *NtfyService) Send(title, message, priority string) error {
	if !n.Config.Enabled {
		return nil
	}

	url := fmt.Sprintf("%s/%s", n.Config.Server, n.Config.Topic)

	// Ntfy supports headers for title, priority etc.
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(message))
	if err != nil {
		return err
	}

	req.Header.Set("Title", title)
	req.Header.Set("Priority", priority) // high, default, low
	if n.Config.Token != "" {
		req.Header.Set("Authorization", "Bearer "+n.Config.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return fmt.Errorf("ntfy returned status: %d", resp.StatusCode)
	}

	return nil
}
