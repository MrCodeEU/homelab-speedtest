package orchestrator

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHClient struct {
	client *ssh.Client
}

func ConnectSSH(user, host string, port int, authMethods []ssh.AuthMethod) (*SSHClient, error) {
	if len(authMethods) == 0 {
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
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
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

func (s *SSHClient) CopyFile(localPath, remotePath string, mode os.FileMode) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	session, err := s.client.NewSession()
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()

	session.Stdin = f
	cmd := fmt.Sprintf("rm -f %s && cat > %s && chmod %o %s", remotePath, remotePath, mode, remotePath)
	if err = session.Run(cmd); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

func (s *SSHClient) FileExists(path string) bool {
	session, err := s.client.NewSession()
	if err != nil {
		return false
	}
	defer func() { _ = session.Close() }()
	err = session.Run(fmt.Sprintf("test -f %s", path))
	return err == nil
}

// RunCommand executes a command and returns stdout and stderr separately.
func (s *SSHClient) RunCommand(cmd string) (string, string, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return "", "", err
	}
	defer func() { _ = session.Close() }()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(cmd)

	cleanOut := bytes.ReplaceAll(stdout.Bytes(), []byte{0}, []byte{})
	cleanErr := bytes.ReplaceAll(stderr.Bytes(), []byte{0}, []byte{})

	return string(bytes.TrimSpace(cleanOut)), string(bytes.TrimSpace(cleanErr)), err
}
