package imgpool

import (
	"bufio"
	"context"
	"goibed/config"
	"goibed/database"
	"goibed/utils"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"
)

type Task struct {
	ID string
	// Absolute path because of immediate converting
	TempPath string
}

var taskQueue chan Task
var logger *slog.Logger

func init() {
	logger = slog.With("converter")
}

func InitPool() {
	taskQueue = make(chan Task, config.Config.MaxQueueLength)

	for i := range config.Config.MaxImgPool {
		go worker(i)
	}
}

func Submit(task Task) bool {
	select {
	case taskQueue <- task:
		return true
	default:
		return false
	}
}

func worker(workerID int) {
	for task := range taskQueue {
		logger.Debug("start converting",
			"workerID", workerID,
			"ID", task.ID,
			"src", task.TempPath,
		)
		task.convert()
	}
}

func runImageMagick(task *Task, arg ...string) error {
	cmd := exec.Command("magick", arg...)
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logger.Error("failed to get stderr pipe", "err", err)
		return err
	}
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			logger.Error("ImageMagick Error", "output", line, "src", task.TempPath)
		}

		if err := scanner.Err(); err != nil {
			logger.Error("Reading ImageMagick Error Error", "err", err, "src", task.TempPath)
		}
	}()

	if err := cmd.Start(); err != nil {
		logger.Error("failed to start magick", "err", err)
		return err
	}
	if err := cmd.Wait(); err != nil {
		logger.Error("magick doesn't work normally", "err", err)
		return err
	}
	return nil
}

func (task *Task) convert() {
	dstFile := strings.TrimSuffix(task.TempPath, filepath.Ext(task.TempPath)) + ".avif"

	if err := runImageMagick(task, task.TempPath, dstFile); err != nil {
		return
	}

	// remove original img after one minute
	time.AfterFunc(1*time.Minute, func() {
		if err := os.Remove(task.TempPath); err != nil {
			logger.Error("failed to remove temp img file", "src", task.TempPath)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	dstFileRelative, err := utils.ConvertToRelative(dstFile)
	if err != nil {
		logger.Error("failed to convert to relative")
		return
	}
	rowsAffected, err := gorm.G[database.Images](database.DB).Where("ID = ?", task.ID).Updates(ctx, database.Images{
		ImagePath: dstFileRelative,
	})
	if rowsAffected != 1 || err != nil {
		logger.Error("failed to update images table normally",
			"rowsAffected", rowsAffected,
			"err", err,
		)
		return
	}
}
