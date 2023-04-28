package logs

import (
	"testing"
	"time"
)

func TestConsoleLog(t *testing.T) {
	//fmt.Println("testconsolelog")
	DefaultConsoleLog.Info("Test","test info")
	DefaultConsoleLog.Warn("Test","test warn")
	DefaultConsoleLog.Error("Test","test error")

	time.Sleep(10 * time.Second)
}

