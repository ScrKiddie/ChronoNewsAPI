package compression

import (
	"chrononewsapi/internal/entity"
	"context"
)

func (cs *CompressionService) runSequential(ctx context.Context, tasks []entity.File) (successCount, failedCount int, fileResults []FileProcessResult) {
	fileResults = make([]FileProcessResult, 0, len(tasks))

	for _, task := range tasks {
		select {
		case <-ctx.Done():
			cs.logInfo("Sequential process cancelled")
			return
		default:
			cs.logDebug("Processing file", "mode", "sequential", "filename", task.Name)
			err := cs.compressImage(task)

			if err != nil {
				failedCount++
				cs.handleFailure(task, err)

				fileResults = append(fileResults, FileProcessResult{
					FileID:  task.ID,
					Success: false,
					Error:   err,
				})
			} else {
				successCount++
				cs.handleSuccess(task)

				fileResults = append(fileResults, FileProcessResult{
					FileID:  task.ID,
					Success: true,
					Error:   nil,
				})
			}
		}
	}
	return
}
