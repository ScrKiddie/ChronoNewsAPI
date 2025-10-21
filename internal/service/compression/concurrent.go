package compression

import (
	"chrononewsapi/internal/entity"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type readJob struct {
	task      entity.File
	sourceDir string
}

type processJob struct {
	task   entity.File
	reader io.ReadCloser
	err    error
}

type writeJob struct {
	task        entity.File
	reader      io.ReadCloser
	destination string
	err         error
}

type processResult struct {
	task entity.File
	err  error
}

func (cs *CompressionService) runConcurrent(ctx context.Context, tasks []entity.File) (successCount, failedCount int, fileResults []FileProcessResult) {
	sourceDir := cs.Config.Storage.Upload
	destDir := cs.Config.Storage.Compressed

	numIOWorkers := cs.Config.Compression.NumWorkers / 2
	if numIOWorkers < 1 {
		numIOWorkers = 1
	}
	numCPUWorkers := cs.Config.Compression.NumWorkers

	readJobs := make(chan readJob, len(tasks))
	processQueue := make(chan processJob, numIOWorkers)
	writeQueue := make(chan writeJob, numCPUWorkers)
	results := make(chan processResult, len(tasks))

	var readerWg, processorWg, writerWg sync.WaitGroup

	readerWg.Add(numIOWorkers)
	for i := 1; i <= numIOWorkers; i++ {
		go cs.readerWorker(ctx, readJobs, processQueue, &readerWg, i)
	}

	processorWg.Add(numCPUWorkers)
	for i := 1; i <= numCPUWorkers; i++ {
		go cs.processorWorker(ctx, processQueue, writeQueue, &processorWg, destDir, i)
	}

	writerWg.Add(numIOWorkers)
	for i := 1; i <= numIOWorkers; i++ {
		go cs.writerWorker(ctx, writeQueue, results, &writerWg, i)
	}

SendLoop:
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			cs.logWarn("shutdown requested, stopping task dispatch")
			break SendLoop
		case readJobs <- readJob{task: task, sourceDir: sourceDir}:
		}
	}
	close(readJobs)

	go func() {
		readerWg.Wait()
		close(processQueue)
		processorWg.Wait()
		close(writeQueue)
		writerWg.Wait()
		close(results)
	}()

	fileResults = make([]FileProcessResult, 0, len(tasks))

	for result := range results {
		if result.err != nil {
			failedCount++
			cs.handleFailure(result.task, result.err)

			fileResults = append(fileResults, FileProcessResult{
				FileID:  result.task.ID,
				Success: false,
				Error:   result.err,
			})
		} else {
			successCount++
			cs.handleSuccess(result.task)

			fileResults = append(fileResults, FileProcessResult{
				FileID:  result.task.ID,
				Success: true,
				Error:   nil,
			})
		}
	}

	cs.logInfo("concurrent pipeline completed", "successful", successCount, "failed", failedCount)
	return
}

func (cs *CompressionService) readerWorker(
	ctx context.Context,
	jobs <-chan readJob,
	processQueue chan<- processJob,
	wg *sync.WaitGroup,
	workerID int,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}

			sourceFile := filepath.Join(job.sourceDir, job.task.Name)
			file, err := os.Open(sourceFile)
			if err != nil {
				err = fmt.Errorf("reader %d: %w", workerID, err)
				cs.logWarn(err.Error(), "file_name", job.task.Name)
				processQueue <- processJob{task: job.task, err: err}
				continue
			}

			cs.logDebug("file read",
				"worker", fmt.Sprintf("reader-%d", workerID),
				"file", job.task.Name,
			)
			processQueue <- processJob{task: job.task, reader: file}
		}
	}
}

func (cs *CompressionService) processorWorker(
	ctx context.Context,
	processQueue <-chan processJob,
	writeQueue chan<- writeJob,
	wg *sync.WaitGroup,
	destDir string,
	workerID int,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-processQueue:
			if !ok {
				return
			}

			if job.err != nil {
				writeQueue <- writeJob{task: job.task, err: job.err}
				continue
			}

			cs.logDebug("processing image",
				"worker", fmt.Sprintf("processor-%d", workerID),
				"file", job.task.Name,
			)

			processedReader, err := cs.processImageWithReader(job.reader)
			if err != nil {
				err = fmt.Errorf("processor %d: %w", workerID, err)
				cs.logWarn(err.Error(), "file_name", job.task.Name)
				writeQueue <- writeJob{task: job.task, err: err}
				continue
			}

			originalName := job.task.Name[:len(job.task.Name)-len(filepath.Ext(job.task.Name))]
			newFileName := fmt.Sprintf("%s.webp", originalName)
			outputFilePath := filepath.Join(destDir, newFileName)

			writeQueue <- writeJob{
				task:        job.task,
				reader:      processedReader,
				destination: outputFilePath,
			}
		}
	}
}

func (cs *CompressionService) writerWorker(
	ctx context.Context,
	writeQueue <-chan writeJob,
	results chan<- processResult,
	wg *sync.WaitGroup,
	workerID int,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-writeQueue:
			if !ok {
				return
			}

			if job.err != nil {
				results <- processResult{task: job.task, err: job.err}
				continue
			}

			cs.logDebug("writing file",
				"worker", fmt.Sprintf("writer-%d", workerID),
				"destination", job.destination,
			)

			outFile, err := os.Create(job.destination)
			if err != nil {
				results <- processResult{
					task: job.task,
					err:  fmt.Errorf("writer %d: failed to create file: %w", workerID, err),
				}
				job.reader.Close()
				continue
			}

			_, err = io.Copy(outFile, job.reader)
			closeErr := outFile.Close()
			job.reader.Close()

			if err != nil {
				results <- processResult{
					task: job.task,
					err:  fmt.Errorf("writer %d: failed to write: %w", workerID, err),
				}
				continue
			}

			if closeErr != nil {
				results <- processResult{
					task: job.task,
					err:  fmt.Errorf("writer %d: failed to close: %w", workerID, closeErr),
				}
				continue
			}

			results <- processResult{task: job.task, err: nil}
		}
	}
}
