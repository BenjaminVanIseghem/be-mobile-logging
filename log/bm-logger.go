package log

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	counter  = 0
	filePath = "/private/etc/promtail/"
	fileName = "send"
)

//Flush flushes the buffer to the file which will be scraped to Loki
func Flush(buf *bytes.Buffer) {
	file := filePath + fileName + strconv.Itoa(counter) + ".log"

	//Create log file to be scraped to Loki
	w, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	start := time.Now()
	n, err := buf.WriteTo(w)
	if err != nil {
		panic(err)
	}
	logrus.Printf("Copied %v bytes\n", n)

	//Reset buffer
	buf.Reset()
	//Close file
	w.Close()

	counter++

	logrus.Info("Flushing took: ", time.Since(start))
}

//CreateLogBuffer creates an in-memory buffer to temporarily store logs
func CreateLogBuffer() (*bytes.Buffer, *logrus.Entry) {
	memLog := &bytes.Buffer{}
	logger := logrus.New()
	multiWriter := io.MultiWriter(os.Stdout, memLog)
	logger.SetOutput(multiWriter)

	//Create logrus.Entry
	entry := logrus.NewEntry(logger)

	return memLog, entry
}

//Error pushes the error onto the buffer and flushes the buffer to file
func Error(logger *logrus.Entry, msg string, err error, buf *bytes.Buffer, fileName string) {
	logger.Error(msg, err)
	//Flush to file
	Flush(buf)
}

/*
	Fatal func pushes the error onto the buffer and flushes the buffer to file
	Afterwards the Fatal function from logrus is called
*/
func Fatal(logger *logrus.Entry, msg string, err error, buf *bytes.Buffer, fileName string) {
	logger.Error(msg, err)
	//Flush to file
	Flush(buf)
	logrus.Fatal(msg)
}

/*
	Panic func pushes the error onto the buffer and flushes the buffer to file
	Afterwards the Panic function from logrus is called
*/
func Panic(logger *logrus.Entry, msg string, err error, buf *bytes.Buffer, fileName string) {
	logger.Error(msg, err)
	//Flush to file
	Flush(buf)
	logrus.Panic(msg, err)
}
