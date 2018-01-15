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
	timeinterval int64
	maxsize      int64

	backtype string
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
		outputs:    f,
		level:      level,
		calldepth:  2,
		timeformat: timeformat,
		backtype:   "m",
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
			var hour, min, sec = time.Clock()
			var _, month, day = time.Date()
			var weekday = time.Weekday()

			var bak = false
			switch l.backtype {
			case "size":
				//file size
			case "m":
				//minute
				if sec == 0 {
					bak = true
				}
			case "h":
				//hour
				if min == 0 && sec == 0 {
					bak = true
				}
			case "D":
				//day
				if hour == 0 && min == 0 && sec == 0 {
					bak = true
				}
			case "W":
				//weekday
				if weekday.String() == "Sunday" && hour == 0 && min == 0 && sec == 0 {
					bak = true
				}
			case "M":
				//month
				if day == 1 && hour == 0 && min == 0 && sec == 0 {
					bak = true
				}
			case "Y":
				//year
				if month.String() == "January" && day == 1 && hour == 0 && min == 0 && sec == 0 {
					bak = true
				}
			default:
				//pass
			}

			if bak {
				var bak_name []byte
				bak_name = append(bak_name, name...)
				bak_name = append(bak_name, ".bak."...)
				bak_name = append(bak_name, time.Format("2006-01-02 15:04:05")...)
				var new_file, err = backup(name, string(bak_name))
				if err != nil {
					return i + 1, err
				}

				if new_file != nil {
					var err_c = f.Close()
					if err_c != nil {
						return i + 1, err_c
					}

					l.outputs[i] = new_file
					f = new_file
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

func backup(old_path string, new_path string) (*os.File, error) {
	if !CheckPathExists(new_path) {
		var err_rn = os.Rename(old_path, new_path)
		if err_rn != nil {
			return nil, err_rn
		}

		var new_file, err_nf = CreateFile(old_path)
		if err_nf != nil {
			return nil, err_nf
		}

		return new_file, nil
	}

	return nil, nil
}

func CheckPathExists(path string) bool {
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
