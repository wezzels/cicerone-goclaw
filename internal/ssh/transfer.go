package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
)

// Transfer handles file transfers over SFTP
type Transfer struct {
	client  *Client
	sftpCli *sftp.Client
}

// NewTransfer creates a new transfer handler
func NewTransfer(client *Client) (*Transfer, error) {
	if !client.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	sftpCli, err := sftp.NewClient(client.RawClient())
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	return &Transfer{
		client:  client,
		sftpCli: sftpCli,
	}, nil
}

// Close closes the SFTP client
func (t *Transfer) Close() error {
	if t.sftpCli != nil {
		return t.sftpCli.Close()
	}
	return nil
}

// Push uploads a local file to remote
func (t *Transfer) Push(localPath, remotePath string) error {
	// Open local file
	local, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer local.Close()

	// Create remote directory if needed
	remoteDir := filepath.Dir(remotePath)
	if err := t.sftpCli.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Create remote file
	remote, err := t.sftpCli.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer remote.Close()

	// Copy
	if _, err := io.Copy(remote, local); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// Pull downloads a remote file to local
func (t *Transfer) Pull(remotePath, localPath string) error {
	// Open remote file
	remote, err := t.sftpCli.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remote.Close()

	// Create local directory if needed
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Create local file
	local, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer local.Close()

	// Copy
	if _, err := io.Copy(local, remote); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// PushDir uploads a directory recursively
func (t *Transfer) PushDir(localDir, remoteDir string) error {
	return filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		rel, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}

		remotePath := filepath.Join(remoteDir, rel)

		if info.IsDir() {
			return t.sftpCli.MkdirAll(remotePath)
		}

		return t.Push(path, remotePath)
	})
}

// PullDir downloads a directory recursively
func (t *Transfer) PullDir(remoteDir, localDir string) error {
	walker := t.sftpCli.Walk(remoteDir)

	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}

		path := walker.Path()
		info := walker.Stat()

		rel, err := filepath.Rel(remoteDir, path)
		if err != nil {
			return err
		}

		localPath := filepath.Join(localDir, rel)

		if info.IsDir() {
			if err := os.MkdirAll(localPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := t.Pull(path, localPath); err != nil {
			return err
		}
	}

	return nil
}

// ListRemote lists files in a remote directory
func (t *Transfer) ListRemote(remotePath string) ([]string, error) {
	files, err := t.sftpCli.ReadDir(remotePath)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, f := range files {
		names = append(names, f.Name())
	}
	return names, nil
}

// StatRemote gets file info for remote path
func (t *Transfer) StatRemote(remotePath string) (os.FileInfo, error) {
	return t.sftpCli.Stat(remotePath)
}

// RemoveRemote removes a remote file
func (t *Transfer) RemoveRemote(remotePath string) error {
	return t.sftpCli.Remove(remotePath)
}

// RenameRemote renames a remote file
func (t *Transfer) RenameRemote(oldPath, newPath string) error {
	return t.sftpCli.Rename(oldPath, newPath)
}

// ExistsRemote checks if a remote path exists
func (t *Transfer) ExistsRemote(remotePath string) bool {
	_, err := t.sftpCli.Stat(remotePath)
	return err == nil
}

// MkdirRemote creates remote directory
func (t *Transfer) MkdirRemote(remotePath string) error {
	return t.sftpCli.MkdirAll(remotePath)
}

// RmdirRemote removes remote directory
func (t *Transfer) RmdirRemote(remotePath string) error {
	return t.sftpCli.RemoveAll(remotePath)
}