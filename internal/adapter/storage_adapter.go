package adapter

import (
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
)

type StorageAdapter struct {
}

func NewStorageAdapter() *StorageAdapter {
	return &StorageAdapter{}
}

func (f *StorageAdapter) Store(file *multipart.FileHeader, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	fileOpened, err := file.Open()
	if err != nil {
		return err
	}
	defer fileOpened.Close()
	fileStored, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fileStored.Close()
	_, err = io.Copy(fileStored, fileOpened)
	if err != nil {
		_ = os.Remove(path)
		return err
	}
	if err != nil {
		return err
	}
	return nil
}

func (f *StorageAdapter) Delete(path string) error {
	err := os.Remove(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Warn("File already deleted or not exists", "path", path)
			return nil
		}
		return err
	}
	return nil
}

func (f *StorageAdapter) Copy(fileName, tempDir, destDir string) error {
	tempPath := filepath.Join(tempDir, fileName)
	destPath := filepath.Join(destDir, fileName)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	tempFile, err := os.Open(tempPath)
	if err != nil {
		return err
	}
	defer tempFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, tempFile)
	if err != nil {
		return err
	}

	return nil
}
