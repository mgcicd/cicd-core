package connection

import (
	"strings"
)

type KafkaConfig struct {
	Brokers []string
}

func NewKafkaConfig() KafkaConfig {
	s := Value("kafka/log/brokers")

	brokers := strings.Split(s, ",")

	var config = KafkaConfig{
		Brokers: brokers,
	}

	return config
}
