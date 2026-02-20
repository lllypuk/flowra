// Package filestorage provides file storage implementations.
package filestorage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	ext := filepath.Ext(filepath.Base(originalName))
	storedName := fileID.String() + ext

	filePath := filepath.Join(s.baseDir, storedName)
	cleanPath := filepath.Clean(filePath)
	if !strings.HasPrefix(cleanPath, s.baseDir+string(filepath.Separator)) {
		return "", errors.New("invalid file name: resolved path is outside base directory")
	}
	filePath = cleanPath

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
// Returns an error if the resolved path escapes the base directory.
func (s *LocalStorage) FilePath(fileID uuid.UUID, fileName string) (string, error) {
	ext := filepath.Ext(fileName)
	fullPath := filepath.Join(s.baseDir, fileID.String()+ext)
	cleanPath := filepath.Clean(fullPath)

	if !strings.HasPrefix(cleanPath, s.baseDir+string(filepath.Separator)) && cleanPath != s.baseDir {
		return "", errors.New("path traversal detected: resolved path is outside base directory")
	}

	return cleanPath, nil
}

// Delete removes a stored file.
func (s *LocalStorage) Delete(fileID uuid.UUID, fileName string) error {
	path, err := s.FilePath(fileID, fileName)
	if err != nil {
		return err
	}
	if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
		return fmt.Errorf("failed to delete file: %w", removeErr)
	}
	return nil
}

// Exists checks if a file exists in storage.
func (s *LocalStorage) Exists(fileID uuid.UUID, fileName string) bool {
	path, err := s.FilePath(fileID, fileName)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
