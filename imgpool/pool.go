package imgpool

import (
	"context"
	"goibed/config"
	"goibed/database"
	"goibed/utils"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type Task struct {
	ID string
	// Absolute path because of immediate converting
	TempPath string
	done     func()
}

// SubmitBatchAndWait queues every task and returns after all tasks have been converted.
// Queue capacity only limits pending work; it does not limit the batch size.
func SubmitImagesBatchAndWait(images []database.Images) {
	logger.Debug("batch conversion requested", "count", len(images))
	if len(images) == 0 {
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(images))

	done := func() {
		wg.Done()
	}
	for _, image := range images {
		logger.Debug("queueing batch image", "ID", image.ID, "queue_length", len(taskQueue))
		taskQueue <- Task{
			ID:       image.ID,
			TempPath: utils.ConvertToAbsolute(image.ImagePath),
			done:     done,
		}
	}
	wg.Wait()
	logger.Debug("batch conversion completed", "count", len(images))
}

func worker(workerID int) {
	for task := range taskQueue {
		started := time.Now()
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("image conversion panicked",
						"workerID", workerID,
						"ID", task.ID,
						"panic", r,
					)
				}

				logger.Debug("conversion worker finished", "ID", task.ID, "workerID", workerID, "duration", time.Since(started))
				if task.done != nil {
					task.done()
				}
			}()

			task.convert()
		}()
	}
}

var taskQueue chan Task
var logger *slog.Logger

func InitPool() {
	logger = slog.With("component", "converter")
	taskQueue = make(chan Task, config.Config.MaxQueueLength)
	logger.Debug("conversion pool initialized", "workers", config.Config.MaxImgPool, "queue_capacity", config.Config.MaxQueueLength)

	for i := range config.Config.MaxImgPool {
		go worker(i)
	}
}

func Submit(task Task) bool {
	select {
	case taskQueue <- task:
		logger.Debug("conversion queued", "ID", task.ID, "queue_length", len(taskQueue))
		return true
	default:
		logger.Debug("conversion queue full", "ID", task.ID, "queue_length", len(taskQueue))
		return false
	}
}

func runImageMagick(task *Task, arg ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, config.Config.MagickPath, arg...)
	started := time.Now()
	logger.Debug("imagemagick started", "ID", task.ID, "args", arg)
	outputBytes, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		logger.Error("imagemagick timed out",
			"ID", task.ID,
			"tempPath", task.TempPath,
			"err", ctx.Err(),
		)
		return "", ctx.Err()
	}

	if err != nil {
		logger.Error("imagemagick doesn't work normally",
			"err", string(outputBytes),
			"ID", task.ID,
			"tempPath", task.TempPath,
		)
		return "", err
	}

	logger.Debug("imagemagick completed", "ID", task.ID, "duration", time.Since(started))
	return string(outputBytes), nil
}

func (task *Task) convert() {
	dstFile := strings.TrimSuffix(task.TempPath, filepath.Ext(task.TempPath)) + ".avif"
	logger.Debug("conversion started", "ID", task.ID, "src", task.TempPath, "dst", dstFile)

	if _, err := runImageMagick(task, task.TempPath, dstFile); err != nil {
		return
	}
	logger.Debug("image encoded", "ID", task.ID, "dst", dstFile)
	imageInfo, err := runImageMagick(task, "identify", "-precision", "16", "-format", "%w %h %b", "-ping", dstFile)
	if err != nil {
		return
	}
	infos := strings.Split(imageInfo, " ")
	if len(infos) != 3 {
		logger.Error("ImageMagick parse error", "dist", dstFile)
		return
	}
	width, err := strconv.Atoi(infos[0])
	if err != nil {
		logger.Error("ImageMagick parse error", "dist", dstFile)
		return
	}
	height, err := strconv.Atoi(infos[1])
	if err != nil {
		logger.Error("ImageMagick parse error", "dist", dstFile)
		return
	}
	size, err := strconv.Atoi(infos[2][:len(infos[2])-1])
	if err != nil {
		logger.Error("ImageMagick parse error", "dist", dstFile)
		return
	}
	logger.Debug("image metadata identified", "ID", task.ID, "width", width, "height", height, "size", size)

	if task.TempPath != dstFile {
		// remove original img after one minute
		time.AfterFunc(1*time.Minute, func() {
			if err := os.Remove(task.TempPath); err != nil {
				logger.Error("failed to remove temp img file", "src", task.TempPath)
			} else {
				logger.Debug("original removed", "ID", task.ID, "src", task.TempPath)
			}
		})
		logger.Debug("original removal scheduled", "ID", task.ID, "src", task.TempPath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	dstFileRelative, err := utils.ConvertToRelative(dstFile)
	if err != nil {
		logger.Error("failed to convert to relative")
		return
	}
	rowsAffected, err := gorm.G[database.Images](database.DB).Where("ID = ?", task.ID).Updates(ctx, database.Images{
		ImagePath: dstFileRelative,
		Height:    uint32(height),
		Width:     uint32(width),
		Size:      uint64(size),
	})
	if rowsAffected != 1 || err != nil {
		logger.Error("failed to update images table normally",
			"rowsAffected", rowsAffected,
			"err", err,
		)
		return
	}
	logger.Debug("conversion completed", "ID", task.ID, "dst", dstFileRelative, "width", width, "height", height, "size", size)
}
