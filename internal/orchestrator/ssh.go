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
	if len(authMethods) == 0 {
		// Default to loading id_rsa
		key, err := os.ReadFile("/root/.ssh/id_rsa")
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	}

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

func (s *SSHClient) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// CopyFile copies a local file to the remote destination.
func (s *SSHClient) CopyFile(localPath, remotePath string, mode os.FileMode) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

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
	defer func() { _ = session.Close() }()

	// SCP protocol details are tricky to implement manually via stdin.
	// A simpler approach is 'cat > remotePath' if the file is small enough,
	// OR use SFTP if available. Most unix systems have 'cat'.
	// Let's use 'cat' for simplicity with the worker binary.

	// Start remote execution of 'cat > remotePath'
	// We also chmod it afterwards.

	go func() {
		w, errInside := session.StdinPipe()
		if errInside != nil {
			return
		}
		defer func() { _ = w.Close() }()
		_, _ = io.Copy(w, f)
	}()

	if err = session.Run(fmt.Sprintf("cat > %s", remotePath)); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Chmod
	session2, err2 := s.client.NewSession()
	if err2 != nil {
		return err2
	}
	defer func() { _ = session2.Close() }()

	if err = session2.Run(fmt.Sprintf("chmod %o %s", mode, remotePath)); err != nil {
		return fmt.Errorf("failed to chmod: %w", err)
	}

	return nil
}

func (s *SSHClient) RunCommand(cmd string) (string, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", err
	}
	defer func() { _ = session.Close() }()

	var b bytes.Buffer
	session.Stdout = &b
	session.Stderr = &b // Combine stderr for debug

	if err = session.Run(cmd); err != nil {
		return b.String(), err
	}
	return b.String(), nil
}
