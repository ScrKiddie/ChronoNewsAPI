package utility

import (
	"fmt"
	"github.com/shirou/gopsutil/process"
	"log/slog"
	"os"
	"time"
)

func LogResourceUsage(operation string, startTime time.Time) {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		slog.Error("failed to get process info", err)
		return
	}

	cpuPercent, _ := proc.CPUPercent()
	memInfo, _ := proc.MemoryInfo()
	duration := time.Since(startTime)

	slog.Info(fmt.Sprintf("%s completed in %s | CPU: %.2f%% | RAM: %.2f MB",
		operation, duration, cpuPercent, float64(memInfo.RSS)/1024/1024))
}
