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

	var pc, file_name, line_number, ok = runtime.Caller(l.calldepth)
	var func_name = ""
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
	func_name = runtime.FuncForPC(pc).Name()

	buf = append(buf, prefix...)
	buf = append(buf, tfmt...)
	buf = append(buf, " "...)
	buf = append(buf, file_name...)
	buf = append(buf, ":"...)
	buf = append(buf, strconv.Itoa(line_number)...)
	buf = append(buf, " ["...)
	buf = append(buf, func_name...)
	buf = append(buf, "]"...)
	buf = append(buf, " ▸ "...)
	buf = append(buf, logstr...)

	var _, err = l.Write(&buf, now, color)
	if err != nil {
		return err
	}

	return nil
}

/*
 return, FALSE (index + 1, error), SUCCESS (0, nil)
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
				var bak_file_path = fmt.Sprintf("%s.bak.%d", name, time.Unix())
				if !check_path_exists(bak_file_path) {
					var _ = os.Rename(name, bak_file_path)
					var file, err_open_file = os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
					if err_open_file != nil {
						return i + 1, err_open_file
					}
					var _ = f.Close()
					l.outputs[i] = file
					f = file
				}
			}

			final_buf = append(final_buf, *b...)
		}

		var _, err = f.Write(final_buf)
		if err != nil {
			return i + 1, err
		}
	}

	return 0, nil
}

func check_path_exists(path string) bool {
	var _, err = os.Stat(path)
	if err == nil {
		return true
	}

	if os.IsExist(err) {
		return true
	}

	return false
}

func CreateFile(path string) (*os.File, error) {
	var file, err = os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	return file, err
}

func (l *Logger) Debug(v ...interface{}) {
	if l.level <= LevelDebug {
		var prefix = "[DEBUG] "
		var s = fmt.Sprintln(v...)
		var err = l.Output(prefix, s, TerminalColorGreen)
		if err != nil {
			panic(err.Error())
		}
	}
}

func (l *Logger) Info(v ...interface{}) {
	if l.level <= LevelInfo {
		var prefix = "[Info] "
		var s = fmt.Sprintln(v...)
		var err = l.Output(prefix, s, TerminalColorWhite)
		if err != nil {
			panic(err.Error())
		}
	}
}

func (l *Logger) Warn(v ...interface{}) {
	if l.level <= LevelWarn {
		var prefix = "[Warn] "
		var s = fmt.Sprintln(v...)
		var err = l.Output(prefix, s, TerminalColorYellow)
		if err != nil {
			panic(err.Error())
		}
	}
}

func (l *Logger) Error(v ...interface{}) {
	if l.level <= LevelError {
		var prefix = "[Error] "
		var s = fmt.Sprintln(v...)
		var err = l.Output(prefix, s, TerminalColorRed)
		if err != nil {
			panic(err.Error())
		}

		panic(s)
	}
}
