package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

type Store struct {
	Root string
}

func New(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("SMOI_DATA_DIR is empty")
	}

	directories := []string{
		"database",
		"attachments/users",
		"tmp",
		"tmp/deletions",
	}
	for _, directory := range directories {
		path := filepath.Join(root, directory)
		if err := os.MkdirAll(path, 0700); err != nil {
			return nil, fmt.Errorf("create %s: %w", path, err)
		}
	}

	if err := checkWritable(filepath.Join(root, "tmp")); err != nil {
		return nil, fmt.Errorf("persistent storage not writable: %w", err)
	}

	return &Store{Root: root}, nil
}

func checkWritable(directory string) error {
	file, err := os.CreateTemp(directory, "writable-")
	if err != nil {
		return err
	}

	filename := file.Name()
	_ = file.Close()
	return os.Remove(filename)
}

func NewID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes[:])
}

func (s *Store) Path(relative string) (string, error) {
	if !filepath.IsLocal(relative) || strings.Contains(relative, "\\") {
		return "", errors.New("invalid storage path")
	}

	resolved := filepath.Join(s.Root, relative)
	root, err := filepath.Abs(s.Root)
	if err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}

	if absolute != root && !strings.HasPrefix(absolute, root+string(os.PathSeparator)) {
		return "", errors.New("invalid storage path")
	}

	return resolved, nil
}

func (s *Store) ConversationDir(userID, conversationID string) string {
	return filepath.Join(s.Root, "attachments", "users", userID, conversationID)
}

func (s *Store) UserDir(userID string) string {
	return filepath.Join(s.Root, "attachments", "users", userID)
}

func (s *Store) Save(
	userID, conversationID, attachmentID, filename string, source io.Reader, maxBytes int64,
) (string, int64, error) {
	extension := filepath.Ext(filename)
	for _, compound := range []string{".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst"} {
		if strings.HasSuffix(strings.ToLower(filename), compound) {
			extension = filename[len(filename)-len(compound):]
			break
		}
	}
	if extension == "." {
		return "", 0, errors.New("invalid file extension")
	}
	for _, character := range extension {
		if unicode.IsControl(character) || character == '/' || character == '\\' {
			return "", 0, errors.New("invalid file extension")
		}
	}

	relative := filepath.Join("attachments", "users", userID, conversationID, attachmentID+extension)
	destination, err := s.Path(relative)
	if err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
		return "", 0, err
	}

	temporary, err := os.CreateTemp(filepath.Join(s.Root, "tmp"), "upload-")
	if err != nil {
		return "", 0, err
	}
	defer os.Remove(temporary.Name())

	size, copyErr := io.Copy(temporary, io.LimitReader(source, maxBytes+1))
	closeErr := temporary.Close()
	if copyErr != nil {
		return "", 0, copyErr
	}
	if closeErr != nil {
		return "", 0, closeErr
	}
	if size > maxBytes {
		return "", 0, fmt.Errorf("file exceeds %d byte limit", maxBytes)
	}

	if err := os.Rename(temporary.Name(), destination); err != nil {
		return "", 0, err
	}

	return relative, size, nil
}

func (s *Store) DeleteWithDatabase(path string, deleteRecords func() error) error {
	trash := filepath.Join(s.Root, "tmp", "deletions", NewID())
	renamed := false

	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, trash); err != nil {
			return fmt.Errorf("prepare attachment deletion: %w", err)
		}
		renamed = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := deleteRecords(); err != nil {
		if renamed {
			if restoreErr := os.Rename(trash, path); restoreErr != nil {
				return fmt.Errorf("database deletion: %v; restore: %w", err, restoreErr)
			}
		}
		return err
	}

	if renamed {
		if err := os.RemoveAll(trash); err != nil {
			return fmt.Errorf("database deleted; cleanup %s failed: %w", trash, err)
		}
	}

	return nil
}

func (s *Store) CleanupPendingFiles(paths []string) error {
	for _, relative := range paths {
		absolute, err := s.Path(relative)
		if err != nil {
			return err
		}
		if err := os.Remove(absolute); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	return nil
}

func (s *Store) CleanTemporary(maxAge time.Duration) {
	directory := filepath.Join(s.Root, "tmp")
	entries, err := os.ReadDir(directory)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err == nil && info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(directory, entry.Name()))
		}
	}
}
