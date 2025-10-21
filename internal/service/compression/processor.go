package compression

import (
	"chrononewsapi/internal/entity"
	"chrononewsapi/vips"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
)

func (cs *CompressionService) compressImage(task entity.File) error {
	sourceFile := filepath.Join(cs.Config.Storage.Upload, task.Name)
	file, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("gagal membuka file sumber: %w", err)
	}
	defer file.Close()

	processedReader, err := cs.processImageWithReader(file)
	if err != nil {
		return fmt.Errorf("gagal memproses gambar: %w", err)
	}
	defer processedReader.Close()

	originalName := task.Name[:len(task.Name)-len(filepath.Ext(task.Name))]
	newFileName := fmt.Sprintf("%s.webp", originalName)
	outputFilePath := filepath.Join(cs.Config.Storage.Compressed, newFileName)

	outFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("gagal membuat file tujuan: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, processedReader); err != nil {
		return fmt.Errorf("gagal menulis hasil ke file: %w", err)
	}

	return nil
}

func (cs *CompressionService) processImageWithReader(reader io.ReadCloser) (io.ReadCloser, error) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		defer reader.Close()

		source := vips.NewSource(reader)
		defer source.Close()

		img, err := vips.NewImageFromSource(source, &vips.LoadOptions{
			Access:      vips.AccessSequentialUnbuffered,
			FailOnError: true,
		})
		if err != nil {
			pw.CloseWithError(fmt.Errorf("gagal membuat image dari source: %w", err))
			return
		}
		defer img.Close()

		w, h := img.Width(), img.Height()
		scale := cs.calculateOptimalScale(w, h)

		if scale < 1.0 {
			if err = img.Resize(scale, nil); err != nil {
				pw.CloseWithError(fmt.Errorf("gagal resize: %w", err))
				return
			}
		}

		target := vips.NewTarget(pw)
		defer target.Close()

		err = img.WebpsaveTarget(target, &vips.WebpsaveTargetOptions{
			Q: cs.Config.Compression.WebPQuality,
		})
		if err != nil {
			pw.CloseWithError(fmt.Errorf("gagal save WebP: %w", err))
			return
		}
	}()

	return pr, nil
}

func (cs *CompressionService) calculateOptimalScale(w, h int) float64 {
	maxWidth := cs.Config.Compression.MaxWidth
	maxHeight := cs.Config.Compression.MaxHeight

	if w <= maxWidth && h <= maxHeight {
		return 1.0
	}

	return math.Min(float64(maxWidth)/float64(w), float64(maxHeight)/float64(h))
}
