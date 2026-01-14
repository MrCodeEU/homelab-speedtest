package orchestrator

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	client *ssh.Client
}

func ConnectSSH(user, host string, port int, authMethods []ssh.AuthMethod) (*SSHClient, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: authMethods,
		// CAUTION: In a real environment, HostKeyCallback should be strict.
		// For this homelab simplified setup, we skip verification or user must configure known_hosts.
		// We'll use InsecureIgnoreHostKey for now as per "safe-ish" plan, but should ideally use known_hosts.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}

	return &SSHClient{client: client}, nil
}

func (s *SSHClient) Close() {
	if s.client != nil {
		s.client.Close()
	}
}

// CopyFile copies a local file to the remote destination.
func (s *SSHClient) CopyFile(localPath, remotePath string, mode os.FileMode) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}
	// Use actual file size/mode if not overridden, but we often want +x
	_ = stat

	session, err := s.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// SCP protocol details are tricky to implement manually via stdin.
	// A simpler approach is 'cat > remotePath' if the file is small enough,
	// OR use SFTP if available. Most unix systems have 'cat'.
	// Let's use 'cat' for simplicity with the worker binary.

	// Start remote execution of 'cat > remotePath'
	// We also chmod it afterwards.

	go func() {
		w, _ := session.StdinPipe()
		defer w.Close()
		io.Copy(w, f)
	}()

	if err := session.Run(fmt.Sprintf("cat > %s", remotePath)); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Chmod
	session2, err := s.client.NewSession()
	if err != nil {
		return err
	}
	defer session2.Close()

	if err := session2.Run(fmt.Sprintf("chmod %o %s", mode, remotePath)); err != nil {
		return fmt.Errorf("failed to chmod: %w", err)
	}

	return nil
}

func (s *SSHClient) RunCommand(cmd string) (string, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	session.Stderr = &b // Combine stderr for debug

	err = session.Run(cmd)
	return b.String(), err
}
