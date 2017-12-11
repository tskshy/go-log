package log

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

/*
 开发日志：
 1. 替换 fmt.Sprintf => []byte(三个地方) 节省大约800ns
*/

const (
	/*terminal color format
	"\x1b[0;%dm%s\x1b[0m"
	*/
	TerminalColorBlack = iota + 30
	TerminalColorRed
	TerminalColorGreen
	TerminalColorYellow
	TerminalColorBlue
	TerminalColorMagenta
	TerminalColorCyan
	TerminalColorWhite
)

const (
	/*log level*/
	LevelDebug = 0
	LevelInfo  = 1
	LevelWarn  = 2
	LevelError = 3
)

func init() {
	/**/
}

type logger_init struct {
}

type Logger struct {
	mux          sync.Mutex
	outputs      []*os.File
	level        int
	calldepth    int
	timeformat   string
	timeinterval int64 //unit: seconds
	maxsize      int64

	inittime time.Time
}

func NewLogger(f []*os.File, level int, timeformat string) *Logger {
	if len(f) == 0 {
		f = []*os.File{os.Stdout}
	}

	if level < LevelDebug || level > LevelError {
		level = LevelInfo
	}

	if timeformat == "" {
		timeformat = "2006-01-02 15:04:05.000"
	}

	return &Logger{
		outputs:      f,
		level:        level,
		calldepth:    2,
		timeformat:   timeformat,
		timeinterval: 1, //当值大于0秒时，按间隔计算，否则按照文件大小计算
		inittime:     time.Now(),
	}
}

func (l *Logger) Output(prefix, logstr string, color int) error {
	var now = time.Now()

	l.mux.Lock()
	defer l.mux.Unlock()

	var buf []byte

	/*logstr format*/
	var tfmt = now.Format(l.timeformat)

	var _, file_name, line_number, ok = runtime.Caller(l.calldepth)
	if !ok {
		return errors.New("runtime caller false.")
	} else {
		for i := len(file_name) - 1; i > 0; i-- {
			if file_name[i] == '/' {
				file_name = file_name[i+1:]
				break
			}
		}
	}

	buf = append(buf, prefix...)
	buf = append(buf, tfmt...)
	buf = append(buf, " "...)
	buf = append(buf, file_name...)
	buf = append(buf, ":"...)
	buf = append(buf, strconv.Itoa(line_number)...)
	buf = append(buf, " ▸ "...)
	buf = append(buf, logstr...)

	//var _, err = l.Write(&buf, now, color)
	//write ...

	for i, f := range l.outputs {
		var fd = f.Fd()
		var name = f.Name() //full name
		var stat, stat_err = f.Stat()
		if stat_err != nil {
			return stat_err
		}
		var size = stat.Size()

	}
	return nil
}

/*
 @param(color): terminal color
 @param(s)    : output string
*/
func (l *Logger) Write(b *[]byte, time time.Time, color int) (int, error) {
	for i, f := range l.outputs {
		var fd = f.Fd()
		var name = f.Name()

		var final_buf []byte
		if (fd == 1 && name == os.Stdout.Name()) || (fd == 2 && name == os.Stderr.Name()) {
			if TerminalColorBlack <= color && color <= TerminalColorWhite {
				final_buf = append(final_buf, "\x1b[0;"...)
				final_buf = append(final_buf, strconv.Itoa(color)...)
				final_buf = append(final_buf, "m"...)
				final_buf = append(final_buf, *b...)
				final_buf = append(final_buf, "\x1b[0m"...)
			} else {
				final_buf = append(final_buf, *b...)
			}
		} else {
			if l.timeinterval > 0 && (time.Unix()-l.inittime.Unix() != 0) && (time.Unix()-l.inittime.Unix())%l.timeinterval == 0 {
				var _ = f.Close()
				var _ = os.Rename(name, fmt.Sprintf("%s.bak.%d", name, time.Unix))
				var nf, _ = os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
				l.outputs[i] = nf
				f = nf
			}

			fmt.Println(f.Stat().Size())

			final_buf = append(final_buf, *b...)
		}

		var _, err = f.Write(final_buf)
		if err != nil {
			return i + 1, err
		}
	}

	return len(l.outputs), nil
}

func create_bak_file(old, new string) {

}

func (l *Logger) Debug(v ...interface{}) {
	if l.level <= LevelDebug {
		var prefix = "[DEBUG] "
		var s = fmt.Sprintln(v...)
		var err = l.Output(prefix, s, TerminalColorGreen)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func (l *Logger) Info(v ...interface{}) {
	if l.level <= LevelInfo {
		var prefix = "[Info] "
		var s = fmt.Sprintln(v...)
		var err = l.Output(prefix, s, TerminalColorWhite)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func (l *Logger) Warn(v ...interface{}) {
	if l.level <= LevelWarn {
		var prefix = "[Warn] "
		var s = fmt.Sprintln(v...)
		var err = l.Output(prefix, s, TerminalColorYellow)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func (l *Logger) Error(v ...interface{}) {
	if l.level <= LevelError {
		var prefix = "[Error] "
		var s = fmt.Sprintln(v...)
		var err = l.Output(prefix, s, TerminalColorRed)
		if err != nil {
			fmt.Println(err.Error())
		}

		panic(s)
	}
}
