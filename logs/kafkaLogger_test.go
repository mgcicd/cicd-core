package logs

import (
	"testing"
	"time"
)

func TestKafkaLogger_Error(t *testing.T) {
	for {
		logger := NewKafkaLogger()
		logger.Info("MonitorApi", "MonitorApi", "hello")

		time.Sleep(5 * time.Second)
	}

}
