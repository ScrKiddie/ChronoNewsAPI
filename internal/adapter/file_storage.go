package adapter

import (
	"io"
	"mime/multipart"
	"os"
)

type FileStorage struct {
}

func NewFileStorage() *FileStorage {
	return &FileStorage{}
}

func (f *FileStorage) Store(file *multipart.FileHeader, path string) error {
	fileOpened, err := file.Open()
	if err != nil {
		return err
	}
	defer fileOpened.Close()
	fileStored, err := os.Create(path)
	if err != nil {
		return err
	}
	_, err = io.Copy(fileStored, fileOpened)
	if err != nil {
		return err
	}
	return nil
}

func (f *FileStorage) Delete(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	return nil
}

func (f *FileStorage) Copy(tempPath, destPath string) error {
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

	if err := os.Remove(tempPath); err != nil {
		return err
	}

	return nil
}
