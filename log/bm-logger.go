package log

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/sirupsen/logrus"
)

//S3Handler struct
type S3Handler struct {
	Session *session.Session
	Bucket  string
}

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
	//S3Region is the region of the bucket
	S3Region = ""
	//S3Bucket is the name of the bucket
	S3Bucket = ""
	handler  S3Handler
)

//InitAWSSession func
func InitAWSSession(region string, bucket string) {
	S3Region = region
	S3Bucket = bucket

	sess, err := session.NewSession(&aws.Config{Region: aws.String(S3Region)})
	if err != nil {
		// Handle error
	}

	h := S3Handler{
		Session: sess,
		Bucket:  S3Bucket,
	}
	handler = h
}

//Flush flushes the buffer to the file which will be scraped to Loki
/*
	If the maximum amount of files is reached, overwrite the oldest file using os.Rename(old, new).
	This limits the file creation overhead to a certain level.
	os.Rename(old, new) is optimized for this use case
*/
func (logFile LFile) Flush() {
	if logFile.errorHappened {
		start := time.Now()

		path := logFile.path + logFile.serviceName + logFile.extraPathInfo + ".log"

		//Upload file to S3 through handler
		err := handler.UploadFile(path, logFile.buffer)
		if err != nil {
			// Handle error
			fmt.Println(err, "Error upload")
		}
		//Get amount of log lines
		n := logFile.buffer.Len()

		logrus.Printf("Copied %v logs\n", n)

		//Append filepath to array
		fileArr = append(fileArr, path)

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
	logFile.Flush()
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
	logFile.Flush()
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

// //Check if path is in array
// func checkPathInArray(path string) bool {
// 	for _, p := range fileArr {
// 		if p == path {
// 			return true
// 		}
// 	}
// 	return false
// }

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

//SetMaxAmountOfFiles func -> Default = 20
func SetMaxAmountOfFiles(amount int) {
	MaxNumberOfFiles = amount
}

//SetMaxAmountOfBuffers func -> Default = 200
func SetMaxAmountOfBuffers(amount int) {
	MaxNumberOfBuffers = amount
}

//UploadFile function
func (h S3Handler) UploadFile(key string, body *bytes.Buffer) error {
	buffer := body.Bytes()

	_, err := s3.New(h.Session).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(h.Bucket),
		Key:                  aws.String(key),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(int64(len(buffer))),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})

	return err
}

//ReadFile function
func (h S3Handler) ReadFile(key string) (string, error) {
	results, err := s3.New(h.Session).GetObject(&s3.GetObjectInput{
		Bucket: aws.String(h.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}
	defer results.Body.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, results.Body); err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}
