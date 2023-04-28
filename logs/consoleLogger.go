package logs

import (
	"fmt"
	"sync"
	"time"
)

var DefaultConsoleLog iconsoleLogger = newDefaultConsoleLogger()

type defaultConsoleLogger struct {
	entries []*consoleLog
	mu      sync.Mutex
}

type consoleLog struct {
	module  string
	message string
	time    time.Time
}

type iconsoleLogger interface {
	Info(module string, message string)
	Warn(module string, message string)
	Error(module string, message string)
}

func newDefaultConsoleLogger() *defaultConsoleLogger {
	logger := &defaultConsoleLogger{}

	go func() {
		for {
			if len(logger.entries) > 0 {
				logger.mu.Lock()

				for _, entry := range logger.entries {
					fmt.Println(fmt.Sprintf("%v : %v By %v", entry.time.Format("2006-01-02 15:04:05"), entry.message, entry.module))
				}

				logger.entries = nil

				logger.mu.Unlock()
			} else {
				time.Sleep(5 * time.Second)
			}
		}
	}()

	return logger
}

func (l *defaultConsoleLogger) Info(module string, message string) {
	l.mu.Lock()

	defer l.mu.Unlock()

	log := &consoleLog{
		module: module,
		message: "[INFO] " + message,
		time: time.Now(),
	}

	l.entries = append(l.entries,log)
}

func (l *defaultConsoleLogger) Warn(module string, message string) {
	l.mu.Lock()

	defer l.mu.Unlock()

	log := &consoleLog{
		module: module,
		message: "[WARN] " + message,
		time: time.Now(),
	}

	l.entries = append(l.entries,log)
}

func (l *defaultConsoleLogger) Error(module string, message string) {
	l.mu.Lock()

	defer l.mu.Unlock()

	log := &consoleLog{
		module: module,
		message: "[Error] " + message,
		time: time.Now(),
	}

	l.entries = append(l.entries,log)
}

