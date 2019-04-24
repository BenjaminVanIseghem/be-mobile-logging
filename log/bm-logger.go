package log

import (
	"bytes"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	//MaxNumberOfFiles var
	MaxNumberOfFiles = 20
	fileArr          = []string{}
	filePath         = "/Users/benjaminvaniseghem/Documents/etc/promtail/"
)

//LFile is an exported struct with a the buffer to which logs are written and extra info for making a write file
type LFile struct {
	buffer        *bytes.Buffer
	serviceName   string
	extraPathInfo string
	errorHappened bool
}

//Flush flushes the buffer to the file which will be scraped to Loki
/*
	If the maximum amount of files is reached, overwrite the oldest file using os.Rename(old, new).
	This limits the file creation overhead to a certain level.
	os.Rename(old, new) is optimized for this use case
*/
func Flush(logFile LFile) {
	if logFile.errorHappened {
		start := time.Now()

		path := filePath + logFile.serviceName + logFile.extraPathInfo + ".log"

		pathInArray := checkPathInArray(path)

		if !pathInArray {
			if len(fileArr) <= MaxNumberOfFiles {
				//Create log file to be scraped to Loki
				w, err := os.Create(path)
				if err != nil {
					panic(err)
				}
				//Write buffer into file
				n, err := logFile.buffer.WriteTo(w)
				if err != nil {
					panic(err)
				}
				logrus.Printf("Copied %v bytes\n", n)
				//Close file
				w.Close()
				//Append filepath to array
				fileArr = append(fileArr, path)
			} else {
				//Take oldest filepath and rename this file to new path name
				err := os.Rename(fileArr[0], path)
				if err != nil {
					logrus.Error("Error renaming file", err)
				}
				//Open renamed log file, this automatically truncates the existing file
				w, err := os.Create(fileArr[0])
				if err != nil {
					panic(err)
				}
				//Write buffer into file
				n, err := logFile.buffer.WriteTo(w)
				if err != nil {
					panic(err)
				}
				logrus.Printf("Copied %v bytes\n", n)
				//Close file
				w.Close()

				//Use slices to add this file to the back of the array
				fileArr = append(fileArr[1:], path)
			}
		} else {
			//Open file and flush buffer into this file
			w, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
			if err != nil {
				panic(err)
			}
			//Write buffer into file
			n, err := logFile.buffer.WriteTo(w)
			if err != nil {
				panic(err)
			}
			logrus.Printf("Copied %v bytes\n", n)
			//Close file
			w.Close()
		}

		//Reset buffer
		logFile.buffer.Reset()

		//Calculate flush time
		logrus.WithFields(
			logrus.Fields{
				"serviceName": logFile.serviceName,
				"extraInfo":   logFile.extraPathInfo,
			}).Info("Flushing took: ", time.Since(start))
	} else {
		logrus.WithFields(
			logrus.Fields{
				"serviceName": logFile.serviceName,
				"extraInfo":   logFile.extraPathInfo,
			}).Info("Buffer cleared without flushing to file")
	}

}

//CreateLogBuffer creates an in-memory buffer to temporarily store logs
func CreateLogBuffer(serviceName string, extraPathInfo string) (LFile, *logrus.Entry) {
	memLog := &bytes.Buffer{}
	logger := logrus.New()
	multiWriter := io.MultiWriter(os.Stdout, memLog)
	logger.SetOutput(multiWriter)

	//Create logrus.Entry
	entry := logrus.NewEntry(logger)

	var logFile = LFile{memLog, serviceName, extraPathInfo, false}

	return logFile, entry
}

//Error pushes the error onto the buffer and flushes the buffer to file
func Error(logger *logrus.Entry, msg string, err error, logFile LFile) {
	logger.Error(msg, err)
	logFile.errorHappened = true
}

/*
	Fatal func pushes the error onto the buffer and flushes the buffer to file
	Afterwards the Fatal function from logrus is called
*/
func Fatal(logger *logrus.Entry, msg string, err error, logFile LFile) {
	logger.Error(msg, err)
	logFile.errorHappened = true
	//Flush to file
	Flush(logFile)
	logrus.Fatal(msg, err)
}

/*
	Panic func pushes the error onto the buffer and flushes the buffer to file
	Afterwards the Panic function from logrus is called
*/
func Panic(logger *logrus.Entry, msg string, err error, logFile LFile) {
	logger.Error(msg, err)
	logFile.errorHappened = true
	//Flush to file
	Flush(logFile)
	logrus.Panic(msg, err)
}

//Check if path is in array
func checkPathInArray(path string) bool {
	for _, p := range fileArr {
		if p == path {
			return true
		}
	}
	return false
}
