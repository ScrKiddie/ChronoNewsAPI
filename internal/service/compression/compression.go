package compression

import (
	"chrononewsapi/internal/config"
	"chrononewsapi/internal/entity"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/process"
	"gorm.io/gorm"
)

type CompressionService struct {
	DB     *gorm.DB
	Config *config.Config
}

type CompressionResult struct {
	TotalDuration time.Duration
	FilesCount    int
	SuccessCount  int
	FailedCount   int
	CPUPercent    float64
	PeakRAMMB     float64
}

func NewCompressionService(db *gorm.DB, cfg *config.Config) *CompressionService {
	return &CompressionService{
		DB:     db,
		Config: cfg,
	}
}

func (cs *CompressionService) shouldLog(level string) bool {
	logLevelOrder := map[string]int{
		"debug": 0,
		"info":  1,
		"warn":  2,
		"error": 3,
	}

	configLevel := cs.Config.Compression.LogLevel
	configLevelNum, ok := logLevelOrder[configLevel]
	if !ok {
		configLevelNum = 1
	}

	messageLevelNum, ok := logLevelOrder[level]
	if !ok {
		return true
	}

	return messageLevelNum >= configLevelNum
}

func (cs *CompressionService) logDebug(msg string, args ...any) {
	if cs.shouldLog("debug") {
		slog.Debug(msg, args...)
	}
}

func (cs *CompressionService) logInfo(msg string, args ...any) {
	if cs.shouldLog("info") {
		slog.Info(msg, args...)
	}
}

func (cs *CompressionService) logWarn(msg string, args ...any) {
	if cs.shouldLog("warn") {
		slog.Warn(msg, args...)
	}
}

func (cs *CompressionService) logError(msg string, args ...any) {
	if cs.shouldLog("error") {
		slog.Error(msg, args...)
	}
}

func (cs *CompressionService) ProcessFiles(ctx context.Context, fileIDs []int32) (*CompressionResult, error) {
	cs.logInfo("Starting compression process", "file_count", len(fileIDs), "mode", cs.getMode())

	var tasks []entity.File
	err := cs.DB.Where("id IN ? AND status = ?", fileIDs, "pending").Find(&tasks).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}

	if len(tasks) == 0 {
		cs.logInfo("No pending tasks found")
		return &CompressionResult{FilesCount: 0}, nil
	}

	cs.DB.Model(&entity.File{}).Where("id IN ?", fileIDs).Update("status", "processing")

	startTime := time.Now()
	p, _ := process.NewProcess(int32(os.Getpid()))
	cpuTimesBefore, _ := p.Times()
	cpuTimeBefore := cpuTimesBefore.User + cpuTimesBefore.System

	doneMonitoring := make(chan struct{})
	resultChan := make(chan uint64, 1)

	go func() {
		resultChan <- cs.monitorPeakRAM(p, doneMonitoring)
	}()

	var successCount, failedCount int
	if cs.Config.Compression.IsConcurrent {
		successCount, failedCount = cs.runConcurrent(ctx, tasks)
	} else {
		successCount, failedCount = cs.runSequential(ctx, tasks)
	}

	close(doneMonitoring)
	peakRAM := <-resultChan

	duration := time.Since(startTime)
	cpuTimesAfter, _ := p.Times()
	cpuTimeAfter := cpuTimesAfter.User + cpuTimesAfter.System
	cpuTimeUsed := cpuTimeAfter - cpuTimeBefore
	cpuPercent := 0.0
	if duration.Seconds() > 0 {
		cpuPercent = (cpuTimeUsed / duration.Seconds()) * 100.0
	}

	result := &CompressionResult{
		TotalDuration: duration,
		FilesCount:    len(tasks),
		SuccessCount:  successCount,
		FailedCount:   failedCount,
		CPUPercent:    cpuPercent,
		PeakRAMMB:     float64(peakRAM) / 1024 / 1024,
	}

	cs.logInfo("Compression completed",
		"duration", duration.String(),
		"cpu_percent", fmt.Sprintf("%.2f%%", cpuPercent),
		"peak_ram_mb", fmt.Sprintf("%.2f MB", result.PeakRAMMB),
		"success", successCount,
		"failed", failedCount,
	)

	return result, nil
}

func (cs *CompressionService) getMode() string {
	if cs.Config.Compression.IsConcurrent {
		return "concurrent"
	}
	return "sequential"
}

func (cs *CompressionService) monitorPeakRAM(p *process.Process, done <-chan struct{}) uint64 {
	var currentPeakRAM uint64
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return currentPeakRAM
		case <-ticker.C:
			memInfo, err := p.MemoryInfo()
			if err == nil {
				if memInfo.RSS > currentPeakRAM {
					currentPeakRAM = memInfo.RSS
				}
			}
		}
	}
}
