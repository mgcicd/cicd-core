package logs

import (
	"cicd-core/config/connection"
	"cicd-core/util"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	v1 "cicd-core/pb/v1"

	"github.com/Shopify/sarama"
)

type kafkaLogger struct {
	sender sarama.AsyncProducer
}

var onceLogger = sync.Once{}
var oneKafkaLogger *kafkaLogger

func NewKafkaLogger() *kafkaLogger {
	if oneKafkaLogger != nil {
		return oneKafkaLogger
	}

	onceLogger.Do(func() {
		oneKafkaLogger = newKafkaLogger()
	})

	return oneKafkaLogger
}

func newKafkaLogger() *kafkaLogger {
	config := sarama.NewConfig()
	//等待服务器所有副本都保存成功后的响应
	config.Producer.RequiredAcks = sarama.WaitForAll
	//随机的分区类型
	config.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	//是否等待成功和失败后的响应,只有上面的RequireAcks设置不是NoReponse这里才有用.
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	// The maximum permitted size of a message (defaults to 1000000). Should be
	// set equal to or smaller than the broker's `message.max.bytes`.
	//set 2m
	config.Producer.MaxMessageBytes = 2000000
	//设置使用的kafka版本,如果低于V0_10_0_0版本,消息中的timestrap没有作用.需要消费和生产同时配置
	config.Version = sarama.V0_10_2_0

	kafkaConfig := connection.NewKafkaConfig()

	producer, e := sarama.NewAsyncProducer(kafkaConfig.Brokers, config)

	if e != nil {
		log.Println(e)
	}

	go func(p sarama.AsyncProducer) {
		for {
			select {
			case <-p.Successes():
				{
					//log.Println("success")
				}
			case err := <-p.Errors():
				{
					log.Println(util.StructToJson(err.Error()))
				}
			}
		}

	}(producer)

	var service = &kafkaLogger{
		sender: producer,
	}

	return service
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() &&
			!strings.Contains(ipnet.IP.String(), "10.250") &&
			!strings.Contains(ipnet.IP.String(), "10.251") &&
			!strings.Contains(ipnet.IP.String(), "10.252") &&
			!strings.Contains(ipnet.IP.String(), "10.253") &&
			!strings.Contains(ipnet.IP.String(), "10.254") &&
			!strings.Contains(ipnet.IP.String(), "10.255") &&
			!strings.Contains(ipnet.IP.String(), "10.256") {
			return ipnet.IP.String()
		}
	}
	return ""
}

var ip = ""

func info0(in *v1.LogInfo) error {

	if ip == "" {
		ip = getLocalIP()
	} else {
		in.UserIp = ip
	}
	if in == nil {
		return errors.New("in is nil")
	}
	if len(fmt.Sprint(in.Exception)) > 40960 {
		log.Println(time.Now().Format("2006-01-02 15:04:05 +08:00") + "LogType:" + in.LogType + "Logger:" + in.Logger)
		return nil
	}

	msg := &sarama.ProducerMessage{
		Key: sarama.StringEncoder(in.LogType),
	}

	logStr := util.StructToJson(in)
	logSize := len(fmt.Sprintf(logStr))
	if logSize >= 1048576 {
		log.Println(time.Now().Format("2006-01-02 15:04:05 +08:00") + "log size large than 1m LogType:" + in.LogType + "_Logger:" + in.Logger + "_size:" + strconv.Itoa(logSize) + "_Level:" + in.Level)
	}

	if in.Level == "ERROR" {
		msg.Topic = "monitor-error-log"
	} else if in.Level == "INFO" {
		msg.Topic = "monitor-info-log"
	} else if in.Level == "WARN" {
		msg.Topic = "monitor-warn-log"
	} else if in.Level == "DEBUG" {
		msg.Topic = "monitor-debug-log"
	} else if in.Level == "TRACE" {
		msg.Topic = "monitor-trace-log"
	} else {
		msg.Topic = "monitor-other-log"
	}

	msg.Value = sarama.ByteEncoder(util.StructToJson(in))

	//使用通道发送
	oneKafkaLogger.sender.Input() <- msg

	return nil
}

func (ls *kafkaLogger) Info(logType string, logger string, message string) {
	log := &v1.LogInfo{
		Message:    message,
		Level:      "INFO",
		LogType:    logType,
		Logger:     logger,
		CreateTime: time.Now().Format(time.RFC3339),
	}

	info0(log)
}

func (ls *kafkaLogger) Error(logType string, logger string, exception string) {
	log := &v1.LogInfo{
		Exception:  exception,
		Level:      "ERROR",
		LogType:    logType,
		Logger:     logger,
		CreateTime: time.Now().Format(time.RFC3339),
	}

	info0(log)
}

func (ls *kafkaLogger) Warn(logType string, logger string, message string) {
	log := &v1.LogInfo{
		Message:    message,
		Level:      "WARN",
		LogType:    logType,
		Logger:     logger,
		CreateTime: time.Now().Format(time.RFC3339),
	}

	info0(log)
}

func (ls *kafkaLogger) Debug(logType string, logger string, message string) {
	log := &v1.LogInfo{
		Message:    message,
		Level:      "DEBUG",
		LogType:    logType,
		Logger:     logger,
		CreateTime: time.Now().Format(time.RFC3339),
	}

	info0(log)
}

func (ls *kafkaLogger) Trace(logType string, logger string, message string) {
	log := &v1.LogInfo{
		Message:    message,
		Level:      "TRACE",
		LogType:    logType,
		Logger:     logger,
		CreateTime: time.Now().Format(time.RFC3339),
	}

	info0(log)
}
