package compression

import (
	"chrononewsapi/internal/entity"
	"context"
)

func (cs *CompressionService) runSequential(ctx context.Context, tasks []entity.File) (successCount, failedCount int) {
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			cs.logInfo("Sequential process cancelled")
			return
		default:
		}

		cs.logDebug("Processing file", "mode", "sequential", "file_name", task.Name)
		err := cs.compressImage(task)
		if err != nil {
			failedCount++
			cs.handleFailure(task, err)
		} else {
			successCount++
			cs.handleSuccess(task)
		}
	}
	return
}
