package imgpool

import "goibed/config"

type Task struct {
	ID       string
	TempPath string
}

var taskQueue chan Task

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
		// handle

	}
}
