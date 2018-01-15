package log

import (
	"log"
	"os"
	"testing"
)

/*
export GOPATH=/root/devel/golang/go-libs/
go test go-log -v -test.run Test_Output
*/
func Test_Output(t *testing.T) {
	var fs = []*os.File{
		os.Stdout,
	}

	var logger = NewLogger(fs, LevelDebug, "2006-01-02 15:04:05.000")

	logger.Debug("12", "asd")
	logger.Info("12", "asd")
	logger.Warn("12", "asd")
	//logger.Error("12", "asd")
}

/*
 go test go-log -v -test.bench Benchmark_Mylog


  200000	     11370 ns/op
PASS
ok  	go-log	2.383s
*/
func Benchmark_Mylog(b *testing.B) {
	var f, _ = os.OpenFile("/tmp/testme.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)

	var fs = []*os.File{
		f,
	}

	var logger = NewLogger(fs, LevelDebug, "2006-01-02 15:04:05.000")

	for i := 0; i < 5000000; i++ {
		logger.Debug("benchmark test测试")
	}
}

/*

  300000	      5071 ns/op
PASS
ok  	go-log	1.575s

*/
func Benchmark_Syslog(b *testing.B) {
	var f, _ = os.OpenFile("/tmp/testsys.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	var l = log.New(f, "", log.LstdFlags|log.Lshortfile)

	for i := 0; i < b.N; i++ {
		l.Println("benchmark test测试")
	}
}
