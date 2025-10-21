package compression

import (
	"chrononewsapi/internal/entity"
	"fmt"
	"path/filepath"
	"strings"

	"gorm.io/gorm"
)

func (cs *CompressionService) handleSuccess(task entity.File) {
	originalNameWithoutExt := strings.TrimSuffix(task.Name, filepath.Ext(task.Name))
	newWebPFileName := fmt.Sprintf("%s.webp", originalNameWithoutExt)

	sourceFilePath := filepath.Join(cs.Config.Storage.Upload, task.Name)

	err := cs.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&task).Updates(map[string]interface{}{
			"status":     "compressed",
			"last_error": nil,
			"name":       newWebPFileName,
		}).Error; err != nil {
			return err
		}

		deletionEntry := entity.SourceFileToDelete{
			FileID:     task.ID,
			SourcePath: sourceFilePath,
		}
		if err := tx.Create(&deletionEntry).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		cs.logError("CRITICAL: Failed to complete success transaction",
			"task_id", task.ID,
			"error", err,
		)
	} else {
		cs.logDebug("File compressed successfully",
			"task_id", task.ID,
			"original_name", task.Name,
			"new_name", newWebPFileName,
		)
	}
}

func (cs *CompressionService) handleFailure(task entity.File, taskErr error) {
	newAttempts := task.FailedAttempts + 1
	errorMessage := taskErr.Error()

	if newAttempts >= cs.Config.Compression.MaxRetries {
		cs.logError("Task permanently failed, moving to DLQ",
			"file_name", task.Name,
			"error", errorMessage,
			"attempts", newAttempts,
		)

		tx := cs.DB.Begin()

		err := tx.Model(&task).Updates(map[string]interface{}{
			"status":          "failed",
			"failed_attempts": newAttempts,
			"last_error":      &errorMessage,
		}).Error

		if err != nil {
			tx.Rollback()
			cs.logError("CRITICAL: Failed to update task status to FAILED",
				"task_id", task.ID,
				"error", err,
			)
			return
		}

		dlqEntry := entity.DeadLetterQueue{
			FileID:       task.ID,
			ErrorMessage: errorMessage,
		}
		err = tx.Create(&dlqEntry).Error

		if err != nil {
			tx.Rollback()
			cs.logError("CRITICAL: Failed to insert task into DLQ",
				"task_id", task.ID,
				"error", err,
			)
			return
		}

		if err := tx.Commit().Error; err != nil {
			cs.logError("CRITICAL: Failed to commit DLQ transaction",
				"task_id", task.ID,
				"error", err,
			)
		}
	} else {
		cs.logWarn("Task failed, will retry in next execution",
			"file_name", task.Name,
			"attempts", newAttempts,
			"error", errorMessage,
		)

		cs.DB.Model(&task).Updates(map[string]interface{}{
			"status":          "pending",
			"failed_attempts": newAttempts,
			"last_error":      &errorMessage,
		})
	}
}
