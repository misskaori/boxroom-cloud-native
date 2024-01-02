package util_log

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	utilfunc "github.io/misskaori/boxroom-crd/kubernetes/util/util-func"
	"os"
	"path/filepath"
)

var Logger = new(NewLog).GetLogger()

type Log interface {
	Format(entry *logrus.Entry) ([]byte, error)
	GetLog() *logrus.Logger
}

type NewLog struct {
	Kind string
	Name string
}

func (f *NewLog) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}
	timestamp := entry.Time.Format("2006-01-02 15:04:05")
	var newLog string
	fName := filepath.Base(entry.Caller.File)
	if entry.Level.String() == "info" || entry.Level.String() == "debug" {
		newLog = fmt.Sprintf("[%s] [%s] %s\n",
			timestamp, entry.Level, entry.Message)
	} else {
		newLog = fmt.Sprintf("\u001B[31m[%s] [%s] [%s:%d %s] %s\u001B[0m\n",
			timestamp, entry.Level, fName, entry.Caller.Line, entry.Caller.Function, entry.Message)
	}
	b.WriteString(newLog)
	return b.Bytes(), nil
}

func (f *NewLog) GetFileLogger(file string) (*logrus.Logger, *os.File, error) {
	fileOperator := utilfunc.NewWorkDirFileOperator()

	log := logrus.New()
	log.SetReportCaller(true)
	log.SetFormatter(f)

	fileWriter, err := fileOperator.CreateFile(file)
	if err != nil {
		log.Error(err)
		err = fileWriter.Close()
		if err != nil {
			log.Error(err)
		}
		return nil, nil, err
	}

	log.SetOutput(fileWriter)

	return log, fileWriter, nil
}

func (f *NewLog) GetLogger() *logrus.Logger {
	log := logrus.New()
	log.SetReportCaller(true)
	log.SetFormatter(f)

	return log
}
