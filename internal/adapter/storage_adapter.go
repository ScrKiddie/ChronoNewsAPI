package adapter

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"

	appConfig "chrononewsapi/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type StorageAdapter struct {
	mode   string
	client *s3.Client
	bucket string
}

func NewStorageAdapter(cfg *appConfig.Config, s3Client *s3.Client) *StorageAdapter {
	return &StorageAdapter{
		mode:   cfg.Storage.Mode,
		client: s3Client,
		bucket: cfg.Storage.S3.Bucket,
	}
}

func (s *StorageAdapter) Store(file *multipart.FileHeader, path string) error {
	if s.mode == "s3" {
		if s.client == nil {

			return errors.New("s3 client is not initialized")
		}
		fileOpened, err := file.Open()
		if err != nil {
			return err
		}

		defer func() {
			if cerr := fileOpened.Close(); cerr != nil {
				slog.Warn("Error closing S3 file source", "error", cerr)
			}
		}()

		s3Key := filepath.ToSlash(path)

		contentType := file.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		_, err = s.client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket:      aws.String(s.bucket),
			Key:         aws.String(s3Key),
			Body:        fileOpened,
			ContentType: aws.String(contentType),
		})
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	fileOpened, err := file.Open()
	if err != nil {
		return err
	}

	defer func() {
		if cerr := fileOpened.Close(); cerr != nil {
			slog.Warn("Error closing opened file", "error", cerr)
		}
	}()

	fileStored, err := os.Create(path)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := fileStored.Close(); cerr != nil {
			slog.Warn("Error closing stored file", "error", cerr)
		}
	}()

	_, err = io.Copy(fileStored, fileOpened)
	if err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func (s *StorageAdapter) Delete(path string) error {
	if s.mode == "s3" {
		if s.client == nil {

			return errors.New("s3 client is not initialized")
		}
		s3Key := filepath.ToSlash(path)
		_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(s3Key),
		})
		return err
	}

	err := os.Remove(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Warn("Attempted to delete a non-existent local file", "path", path)
			return nil
		}
		return err
	}

	return nil
}

func (s *StorageAdapter) Copy(fileName, tempDir, destDir string) error {
	if s.mode == "s3" {
		if s.client == nil {

			return errors.New("s3 client is not initialized")
		}
		sourceKey := filepath.ToSlash(filepath.Join(tempDir, fileName))
		destKey := filepath.ToSlash(filepath.Join(destDir, fileName))

		copySource := "/" + s.bucket + "/" + sourceKey
		encodedCopySource := url.PathEscape(copySource)

		_, err := s.client.CopyObject(context.TODO(), &s3.CopyObjectInput{
			Bucket:     aws.String(s.bucket),
			Key:        aws.String(destKey),
			CopySource: aws.String(encodedCopySource),
		})
		return err
	}

	tempPath := filepath.Join(tempDir, fileName)
	destPath := filepath.Join(destDir, fileName)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	tempFile, err := os.Open(tempPath)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := tempFile.Close(); cerr != nil {
			slog.Warn("Error closing temp file", "error", cerr)
		}
	}()

	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}

	defer func() {
		if cerr := destFile.Close(); cerr != nil {
			slog.Warn("Error closing destination file", "error", cerr)
		}
	}()

	_, err = io.Copy(destFile, tempFile)
	return err
}
