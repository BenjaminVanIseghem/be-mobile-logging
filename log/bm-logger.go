package log

import (
	"bytes"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

//LFile is an exported struct with a the buffer to which logs are written and extra info for making a write file
type LFile struct {
	buffer        *bytes.Buffer
	path          string
	serviceName   string
	extraPathInfo string
	errorHappened bool
}

var (
	//MaxNumberOfFiles var
	MaxNumberOfFiles = 20
	//MaxNumberOfBuffers var
	MaxNumberOfBuffers = 200
	fileArr            = []string{}
	bufSlice           = []LFile{}
	entrySlice         = []*logrus.Entry{}
)

//Flush flushes the buffer to the file which will be scraped to Loki
/*
	If the maximum amount of files is reached, overwrite the oldest file using os.Rename(old, new).
	This limits the file creation overhead to a certain level.
	os.Rename(old, new) is optimized for this use case
*/
func Flush(logFile LFile) {
	if logFile.errorHappened {
		start := time.Now()

		path := logFile.path + logFile.serviceName + logFile.extraPathInfo + ".log"

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
func CreateLogBuffer(path string, serviceName string, extraPathInfo string) (LFile, *logrus.Entry) {
	//Check if there is already an LFile with these credentials
	if checkBufSlice(serviceName, extraPathInfo) {
		//if LFile already exists, return it
		logrus.Warn("Buffer already exists, returning existing buffer")
		var logFile, entry = getLogFileAndEntry(serviceName, extraPathInfo)
		if entry == nil {
			logrus.Warn("Nil buffer")
		}
		return logFile, entry
	}
	//If it's a new LFile, return it and append it in the slice
	memLog := &bytes.Buffer{}
	logger := logrus.New()
	multiWriter := io.MultiWriter(os.Stdout, memLog)
	logger.SetOutput(multiWriter)

	//Create logrus.Entry
	entry := logrus.NewEntry(logger)
	//Create LFile object
	var logFile = LFile{memLog, path, serviceName, extraPathInfo, false}

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

//Error pushes the error onto the buffer and flushes the buffer to file
func Error(logger *logrus.Entry, msg string, err error, logFile *LFile) {
	logger.Error(msg, err)

	tempBool := &logFile.errorHappened
	*tempBool = true
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

//GetLogBuffer function
func GetLogBuffer(serviceName string, extraInfo string) LFile {
	for _, f := range bufSlice {
		if f.serviceName == serviceName && f.extraPathInfo == extraInfo {
			return f
		}
	}
	return LFile{}
}

//GetLogger function
func GetLogger(serviceName string, extraInfo string) *logrus.Entry {
	for i, f := range bufSlice {
		if f.serviceName == serviceName && f.extraPathInfo == extraInfo {
			return entrySlice[i]
		}
	}
	return nil
}

// GetLogBufferAndLogger function
func GetLogBufferAndLogger(serviceName string, extraInfo string) (LFile, *logrus.Entry) {
	for i, f := range bufSlice {
		if f.serviceName == serviceName && f.extraPathInfo == extraInfo {
			return f, entrySlice[i]
		}
	}
	return LFile{}, nil
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

func checkBufSlice(serviceName string, extraInfo string) bool {
	for _, f := range bufSlice {
		if f.serviceName == serviceName && f.extraPathInfo == extraInfo {
			return true
		}
	}
	return false
}

func getLogFileAndEntry(serviceName string, extraInfo string) (LFile, *logrus.Entry) {
	for i, f := range bufSlice {
		if f.serviceName == serviceName && f.extraPathInfo == extraInfo {
			return f, entrySlice[i]
		}
	}
	return LFile{}, nil
}
