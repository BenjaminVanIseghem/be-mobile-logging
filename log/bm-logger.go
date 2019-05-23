package log

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/sirupsen/logrus"
)

//LFile is an exported struct with a the buffer to which logs are written and extra info for making a write file
type LFile struct {
	buffer        *bytes.Buffer
	serviceName   string
	serviceInfo   string
	errorHappened bool
	port          int
	host          string
}

var (
	//MaxNumberOfBuffers var
	MaxNumberOfBuffers = 300
	bufSlice           = []LFile{}
	entrySlice         = []*logrus.Entry{}
)

//Flush flushes the buffer to the file which will be send to Loki via Fluentd
func (logFile LFile) Flush() {
	//Only flush if error has occurred
	if logFile.errorHappened {
		start := time.Now()

		//Tag for Loki, easily filterable in Grafana
		tag := logFile.serviceName + "." + logFile.serviceInfo

		fluent := initFluent(logFile.port, logFile.host)

		//Close the fluent connection
		defer fluent.Close()

		//Iterate through the buffer using a scanner
		scanner := bufio.NewScanner(logFile.buffer)
		for scanner.Scan() {
			data := scanner.Text()
			log := make(map[string]interface{})

			//Unmarshal data into log
			err := json.Unmarshal([]byte(data), &log)
			if err != nil {
				logrus.Error("Unmarshalling error", err)
			}
			//Send every line to Fluentd
			error := fluent.Post(tag, log)
			if error != nil {
				panic(error)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
		}

		//Get amount of log lines
		n := logFile.buffer.Len()

		logrus.Printf("Copied %v logs\n", n)

		//Reset buffer
		logFile.buffer.Reset()

		//Calculate flush time
		logrus.WithFields(
			logrus.Fields{
				"serviceName": logFile.serviceName,
				"serviceInfo": logFile.serviceInfo,
			}).Info("Flushing took: ", time.Since(start))
	} else {
		logrus.WithFields(
			logrus.Fields{
				"serviceName": logFile.serviceName,
				"serviceInfo": logFile.serviceInfo,
			}).Info("Buffer cleared without flushing to file")
	}

}

//CreateLogBuffer creates an in-memory buffer to temporarily store logs
func CreateLogBuffer(serviceName string, serviceInfo string, fluentPort int, fluentHost string) (LFile, *logrus.Entry) {
	//Check if there is already an LFile with these credentials
	if checkBufSlice(serviceName, serviceInfo) {
		//if LFile already exists, return it
		logrus.Warn("Buffer already exists, returning existing buffer")
		var logFile, entry = GetLogBufferAndLogger(serviceName, serviceInfo)
		if entry == nil {
			logrus.Warn("Nil buffer")
		}
		return logFile, entry
	}
	//If it's a new LFile, return it and append it in the slice
	memLog := &bytes.Buffer{}
	logger := logrus.New()
	multiWriter := io.MultiWriter(os.Stdout, memLog)
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(multiWriter)

	//Create logrus.Entry
	entry := logrus.NewEntry(logger)
	//Create LFile object
	var logFile = LFile{memLog, serviceName, serviceInfo, false, fluentPort, fluentHost}

	if len(bufSlice) < MaxNumberOfBuffers {
		//If there is room in the slice, append new LFile and buffer to slice
		bufSlice = append(bufSlice, logFile)
		entrySlice = append(entrySlice, entry)
	} else {
		//If there isn't room in the slice, make new slice without first element and append new LFile
		bufSlice = append(bufSlice[1:], logFile)
		entrySlice = append(entrySlice[1:], entry)
	}

	return logFile, entry
}

//initFluent func initializes the fluentd forwarder
func initFluent(port int, host string) *fluent.Fluent {
	logger, err := fluent.New(fluent.Config{FluentPort: port, FluentHost: host, MarshalAsJSON: true})
	if err != nil {
		fmt.Println(err)
	}
	return logger
}

//Error pushes the error onto the buffer and flushes the buffer to file
func Error(logger *logrus.Entry, msg string, err error, logFile *LFile, m map[string]interface{}) {
	fields := logrus.Fields{}
	for key, value := range m {
		fields[key] = value
	}
	logger.WithFields(fields).Error(msg, err)

	tempBool := &logFile.errorHappened
	*tempBool = true
}

/*
	Fatal func pushes the error onto the buffer and flushes the buffer to file
	Afterwards the Fatal function from logrus is called
*/
func Fatal(logger *logrus.Entry, msg string, err error, logFile LFile, m map[string]interface{}) {
	fields := logrus.Fields{}
	for key, value := range m {
		fields[key] = value
	}
	logger.WithFields(fields).Error(msg, err)

	logFile.errorHappened = true
	//Flush to file
	logFile.Flush()
	logrus.Fatal(msg, err)
}

/*
	Panic func pushes the error onto the buffer and flushes the buffer to file
	Afterwards the Panic function from logrus is called
*/
func Panic(logger *logrus.Entry, msg string, err error, logFile LFile, m map[string]interface{}) {
	fields := logrus.Fields{}
	for key, value := range m {
		fields[key] = value
	}
	logger.WithFields(fields).Error(msg, err)
	logFile.errorHappened = true
	//Flush to file
	logFile.Flush()
	logrus.Panic(msg, err)
}

// GetLogBufferAndLogger function
func GetLogBufferAndLogger(serviceName string, serviceInfo string) (LFile, *logrus.Entry) {
	for i, f := range bufSlice {
		if f.serviceName == serviceName && f.serviceInfo == serviceInfo {
			return f, entrySlice[i]
		}
	}
	return LFile{}, nil
}

func checkBufSlice(serviceName string, serviceInfo string) bool {
	for _, f := range bufSlice {
		if f.serviceName == serviceName && f.serviceInfo == serviceInfo {
			return true
		}
	}
	return false
}

//SetMaxAmountOfBuffers func -> Default = 200
func SetMaxAmountOfBuffers(amount int) {
	MaxNumberOfBuffers = amount
}
