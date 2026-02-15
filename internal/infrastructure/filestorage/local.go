// Package filestorage provides file storage implementations.
package filestorage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/lllypuk/flowra/internal/domain/uuid"
)

// LocalStorage stores files on the local filesystem.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a new local file storage.
// It ensures the base directory exists.
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("invalid upload directory: %w", err)
	}

	if mkErr := os.MkdirAll(absDir, 0o750); mkErr != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", mkErr)
	}

	return &LocalStorage{baseDir: absDir}, nil
}

// Save stores a file and returns the generated file ID.
func (s *LocalStorage) Save(reader io.Reader, originalName string) (uuid.UUID, error) {
	fileID := uuid.NewUUID()
	ext := filepath.Ext(originalName)
	storedName := fileID.String() + ext

	filePath := filepath.Join(s.baseDir, storedName)

	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	if _, copyErr := io.Copy(f, reader); copyErr != nil {
		_ = os.Remove(filePath)
		return "", fmt.Errorf("failed to write file: %w", copyErr)
	}

	return fileID, nil
}

// FilePath returns the full path to a stored file.
func (s *LocalStorage) FilePath(fileID uuid.UUID, fileName string) string {
	ext := filepath.Ext(fileName)
	return filepath.Join(s.baseDir, fileID.String()+ext)
}

// Delete removes a stored file.
func (s *LocalStorage) Delete(fileID uuid.UUID, fileName string) error {
	path := s.FilePath(fileID, fileName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// Exists checks if a file exists in storage.
func (s *LocalStorage) Exists(fileID uuid.UUID, fileName string) bool {
	path := s.FilePath(fileID, fileName)
	_, err := os.Stat(path)
	return err == nil
}
