package utility

import (
	"chrononewsapi/internal/model"
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/shirou/gopsutil/process"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func HandleParallelContentProcessing(content *string) (string, []string, []string, error) {

	var once sync.Once
	errChan := make(chan error, 1)
	var startTime time.Time
	var mu sync.Mutex
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var fileDatas []model.FileData
	var oldFileNames []string

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(*content))
	if err != nil {
		return "", []string{}, []string{}, err
	}

	startTime = time.Now()
	doc.Find("img").Each(func(i int, g *goquery.Selection) {
		src, exists := g.Attr("src")
		if exists && strings.HasPrefix(src, "data:image/") {
			wg.Add(1)
			go func(src string, g *goquery.Selection) {
				defer wg.Done()
				select {
				case <-ctx.Done():
					return
				default:
					file, name, err := Base64ToFile(src)
					if err != nil {
						once.Do(func() {
							errChan <- err
							cancel()
						})
						return
					}
					mu.Lock()
					fileDatas = append(fileDatas, model.FileData{File: file, Name: name})
					g.SetAttr("src", name)
					mu.Unlock()
				}
			}(src, g)
		} else {
			oldFileNames = append(oldFileNames, src)
		}
	})
	wg.Wait()
	select {
	case err := <-errChan:
		return "", []string{}, []string{}, err
	default:
	}

	*content = ""
	newContent, err := doc.Html()
	if err != nil {
		return "", []string{}, []string{}, err
	}
	doc = nil
	LogResourceUsage("proses validasi file", startTime)

	startTime = time.Now()
	var fileNames []string
	if len(fileDatas) > 0 {

		for i, file := range fileDatas {

			wg.Add(1)
			go func(file model.FileData, i int) {
				defer wg.Done()

				select {
				case <-ctx.Done():
					return
				default:
					name, err := CompressImage(file, os.TempDir())
					if err != nil {
						once.Do(func() {
							errChan <- err
							cancel()
						})
						return
					}
					LogResourceUsage("proses kompresi file ke"+strconv.Itoa(i), startTime)
					mu.Lock()
					fileNames = append(fileNames, name)
					mu.Unlock()
				}
			}(file, i)
		}

		wg.Wait()
		select {
		case err := <-errChan:
			return "", []string{}, []string{}, err
		default:
		}
	}
	LogResourceUsage("proses kompresi file", startTime)
	fileDatas = nil
	return newContent, fileNames, oldFileNames, nil
}

func LogResourceUsage(operation string, startTime time.Time) {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		slog.Error("failed to get process info", err)
		return
	}

	cpuPercent, _ := proc.CPUPercent()
	memInfo, _ := proc.MemoryInfo()
	duration := time.Since(startTime)

	slog.Info(fmt.Sprintf("%s completed in %s | CPU: %.2f%% | RAM: %.2f MB", operation, duration, cpuPercent, float64(memInfo.RSS)/1024/1024))
}
