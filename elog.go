package elog

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

const (
	Ldate = 1 << iota
	Ltime
	Lmicroseconds
	LUTC
	Llongfile
	Lshortfile
	Lmsgprefix
	Lmsgcolor
	Llevel
	LstdFlags = Ldate | Ltime | Lshortfile
)

type logLevel int

const (
	FatalLevel logLevel = iota // 致命的错误: 表明程序遇到了致命的错误，必须马上终止运行。
	PanicLevel                 // 致命的错误: 表明程序遇到了致命的错误，可以不马上终止运行，依赖 recover()。
	ErrorLevel                 // 状态错误: 该错误发生后程序仍然可以运行，但是极有可能运行在某种非正常的状态下，导致无法完成全部既定的功能。
	WarnLevel                  // 警告信息: 程序处理中遇到非法数据或者某种可能的错误。该错误是一过性的、可恢复的，不会影响程序继续运行，程序仍处在正常状态。
	InfoLevel                  // 报告程序进度和状态信息: 一般这种信息都是一过性的，不会大量反复输出。例如：连接商用库成功后，可以打印一条连库成功的信息，便于跟踪程序进展信息。
	DebugLevel                 // 终端查看、在线调试: 默认情况下会打印到终端输出，但是不会归档到日志文件。因此，一般用于开发者在程序当前启动窗口上，查看日志流水信息。
	TraceLevel                 // 在线调试: 默认情况下，既不打印到终端也不输出到文件。此时，对程序运行效率几乎不产生影响。常用语 for 循环中调试
	Discard
)

const (
	_FatalLabel = "\x1b[0;30;45m FATAL \x1b[0m "
	_PanicLabel = "\x1b[1;37;45m PANIC \x1b[0m "
	_ErrorLabel = "\x1b[1;37;41m ERROR \x1b[0m "
	_WarnLabel  = "\x1b[0;30;43m WARN  \x1b[0m "
	_InfoLabel  = "\x1b[0;30;46m INFO  \x1b[0m "
	_DebugLabel = "\x1b[0;30;44m DEBUG \x1b[0m "
	_TraceLabel = "\x1b[0;30;42m TRACE \x1b[0m "
)

const (
	red     = "\x1b[1;31;40m"
	green   = "\x1b[1;32;40m"
	yellow  = "\x1b[1;33;40m"
	blue    = "\x1b[1;34;40m"
	magenta = "\x1b[1;35;40m"
	cyan    = "\x1b[1;36;40m"
	while   = "\x1b[1;37;40m"
	color_  = "\x1b[0m"
)

type Logger interface {
	Fatal(...interface{})
	Panic(...interface{})
	Error(...interface{})
	Warn(...interface{})
	Info(...interface{})
	Debug(...interface{})
	Trace(...interface{})

	Fatalf(string, ...interface{})
	Panicf(string, ...interface{})
	Errorf(string, ...interface{})
	Warnf(string, ...interface{})
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
	Tracef(string, ...interface{})
}

var levelMap = map[logLevel]struct {
	levelLabel string
	levelColor string
}{
	FatalLevel: {_FatalLabel, magenta},
	PanicLevel: {_PanicLabel, magenta},
	ErrorLevel: {_ErrorLabel, red},
	WarnLevel:  {_WarnLabel, yellow},
	InfoLevel:  {_InfoLabel, cyan},
	DebugLevel: {_DebugLabel, blue},
	TraceLevel: {_TraceLabel, green},
}

type Log struct {
	mu     sync.Mutex
	output io.Writer // 日志输出方式
	level  logLevel  // 日志等级
	name   string    // 日志对象名称
	flag   int       // 日志对象属性
	prefix string    // 日志前缀
	buf    []byte
}

var _ Logger = &Log{}

// core method
func (l *Log) Out(calldepth int, level logLevel, msg string) error {
	now := time.Now()
	var file string
	var line int
	l.mu.Lock()
	defer l.mu.Unlock()

	// 如果设置了 Lshortfile 或 Llongfile 这两个 flag 则通过 runtime.Caller 获取文件路径和行号
	if l.flag&(Lshortfile|Llongfile) != 0 {
		// 获取 Caller 信息时先释放锁，因为上锁成本很高
		l.mu.Unlock()
		var ok bool
		_, file, line, ok = runtime.Caller(calldepth)
		if !ok {
			file = "??? UNKNOWN FILE ???"
			line = 0
		}
		l.mu.Lock()
	}
	// 清空 buffer
	l.buf = l.buf[:0]
	l.formatHeader(&l.buf, now, level, file, line) // 将格式化头部填充到 buffer 中
	if l.flag&Lmsgcolor != 0 {
		setColor(&l.buf, level)
	}
	l.buf = append(l.buf, msg...)                 // 将打印内容填充到 buffer 中
	if len(msg) == 0 || msg[len(msg)-1] != '\n' { // 如果打印内容为空或者内容末尾没有换行符，则追加换行符
		l.buf = append(l.buf, '\n')
	}
	if l.flag&Lmsgcolor != 0 {
		unsetColor(&l.buf)
	}
	_, err := l.output.Write(l.buf)
	return err
}

func setColor(buf *[]byte, level logLevel) {
	*buf = append(*buf, levelMap[level].levelColor...)
}

func unsetColor(buf *[]byte) {
	*buf = append(*buf, color_...)
}

func (l *Log) formatHeader(buf *[]byte, t time.Time, level logLevel, file string, line int) {
	// Ldate Ltime Lmicroseconds [LEVEL_LABEL] Lshortfile/Llongfile:[LINE] | Lmsgprefix [MESSAGE]

	// 处理日期和时间
	if l.flag&(Ldate|Ltime|Lmicroseconds) != 0 {
		if l.flag&LUTC != 0 {
			t = t.UTC()
		}

		if l.flag&Ldate != 0 {
			year, month, day := t.Date()
			itoa(buf, year, 4)
			*buf = append(*buf, '/')
			itoa(buf, int(month), 2)
			*buf = append(*buf, '/')
			itoa(buf, day, 2)
			*buf = append(*buf, ' ')
		}

		if l.flag&(Ltime|Lmicroseconds) != 0 {
			hour, min, sec := t.Clock()
			itoa(buf, hour, 2)
			*buf = append(*buf, ':')
			itoa(buf, min, 2)
			*buf = append(*buf, ':')
			itoa(buf, sec, 2)

			if l.flag&Lmicroseconds != 0 {
				*buf = append(*buf, '.')
				itoa(buf, t.Nanosecond()/1e3, 6)
			}
			*buf = append(*buf, ' ')
		}
	}

	// 处理等级前缀
	*buf = append(*buf, levelMap[level].levelLabel...)

	// 处理文件路径
	if l.flag&(Lshortfile|Llongfile) != 0 {
		// 如果设置了简洁文件路径，则将文件路径从后往前遍历，找到第一个 '/'，然后取后面的部分
		if l.flag&Lshortfile != 0 {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' {
					short = file[i+1:]
					break
				}
			}
			file = short
		}
		// 如果设置了全文件路径，则直接将填入 buffer
		*buf = append(*buf, file...)
		// 追加行号
		*buf = append(*buf, ':')
		itoa(buf, line, -1)
		// 追加间隔符号，间隔符号后就是打印内容了
		*buf = append(*buf, " |"...)
	}

	// 处理消息前缀 msgPrefix
	if l.flag&Lmsgprefix != 0 {
		*buf = append(*buf, l.prefix...)
	}
}

// 格式化数字，用于格式化日期和时间。当 num 小于 10 时添加一个 0 前缀，当 num 大于 10 时
// 则逐一切取 num 的数值，追加到 buffer 中。wid 为负数时不填充 0 前缀。
func itoa(buf *[]byte, num int, wid int) {
	var b [20]byte
	bIdx := len(b) - 1
	for num >= 10 || wid > 1 {
		// 取模
		q := num / 10
		b[bIdx] = byte('0' + num - q*10)
		// 更新
		num = q
		wid--
		bIdx--
	}
	// num < 10
	b[bIdx] = byte('0' + num)
	*buf = append(*buf, b[bIdx:]...)
}

// Create Logger
type LogOption func(logger *Log)

func OFlag(flag int) LogOption {
	return func(logger *Log) {
		logger.flag = flag
	}
}

func OPrefix(prefix string) LogOption {
	return func(logger *Log) {
		logger.prefix = prefix
	}
}

func OName(name string) LogOption {
	return func(logger *Log) {
		logger.name = name
	}
}

func New(w io.Writer, level logLevel, options ...LogOption) *Log {
	l := new(Log)
	l.output = w
	l.level = level

	for _, opt := range options {
		opt(l)
	}
	return l
}

func Extend(parent *Log, options ...LogOption) *Log {
	son := new(Log)
	if parent == nil {
		parent = std
	}
	son.name = "SonBy" + parent.name
	son.output = parent.output
	son.level = parent.level
	son.flag = parent.flag
	son.prefix = parent.prefix
	for _, opt := range options {
		opt(son)
	}
	return son
}

// Getter & Setter
func (l *Log) Output() io.Writer {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.output
}
func (l *Log) Level() logLevel {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}
func (l *Log) Name() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.name
}
func (l *Log) Flag() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.flag
}
func (l *Log) Prefix() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.prefix
}
func (l *Log) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = w
}
func (l *Log) SetLevel(level logLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}
func (l *Log) SetName(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.name = name
}
func (l *Log) SetFlag(flag int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flag = flag
}
func (l *Log) SetPrefix(prefix string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
}
func (l *Log) UnSetFlag(flag int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flag = l.flag &^ flag
}

// Method Set
func (l *Log) Fatal(v ...interface{}) {
	if l.level >= FatalLevel {
		l.Out(2, FatalLevel, fmt.Sprintln(v...))
		os.Exit(1)
	}
}
func (l *Log) Panic(v ...interface{}) {
	if l.level >= PanicLevel {
		s := fmt.Sprintln(v...)
		l.Out(2, PanicLevel, s)
		panic(s)
	}
}
func (l *Log) Error(v ...interface{}) {
	if l.level >= ErrorLevel {
		l.Out(2, ErrorLevel, fmt.Sprintln(v...))
	}
}
func (l *Log) Warn(v ...interface{}) {
	if l.level >= WarnLevel {
		l.Out(2, WarnLevel, fmt.Sprintln(v...))
	}
}
func (l *Log) Info(v ...interface{}) {
	if l.level >= InfoLevel {
		l.Out(2, InfoLevel, fmt.Sprintln(v...))
	}
}
func (l *Log) Debug(v ...interface{}) {
	if l.level >= DebugLevel {
		l.Out(2, DebugLevel, fmt.Sprintln(v...))
	}
}
func (l *Log) Trace(v ...interface{}) {
	if l.level >= TraceLevel {
		l.Out(2, TraceLevel, fmt.Sprintln(v...))
	}
}

func (l *Log) Fatalf(format string, v ...interface{}) {
	if l.level >= FatalLevel {
		l.Out(2, FatalLevel, fmt.Sprintf(format, v...))
		os.Exit(1)
	}
}
func (l *Log) Panicf(format string, v ...interface{}) {
	if l.level >= PanicLevel {
		s := fmt.Sprintf(format, v...)
		l.Out(2, PanicLevel, s)
		panic(s)
	}
}
func (l *Log) Errorf(format string, v ...interface{}) {
	if l.level >= ErrorLevel {
		l.Out(2, ErrorLevel, fmt.Sprintf(format, v...))
	}
}
func (l *Log) Warnf(format string, v ...interface{}) {
	if l.level >= WarnLevel {
		l.Out(2, WarnLevel, fmt.Sprintf(format, v...))
	}
}
func (l *Log) Infof(format string, v ...interface{}) {
	if l.level >= InfoLevel {
		l.Out(2, InfoLevel, fmt.Sprintf(format, v...))
	}
}
func (l *Log) Debugf(format string, v ...interface{}) {
	if l.level >= DebugLevel {
		l.Out(2, DebugLevel, fmt.Sprintf(format, v...))
	}
}
func (l *Log) Tracef(format string, v ...interface{}) {
	if l.level >= TraceLevel {
		l.Out(2, TraceLevel, fmt.Sprintf(format, v...))
	}
}

// # ============================== 默认全局对象 ======================================
func Default() *Log { return std }

var std *Log = New(os.Stderr, InfoLevel, OName("Global"), OPrefix("[eLog]"), OFlag(LstdFlags))

var (
	// Getter & Setter
	Output    = std.Output
	Level     = std.Level
	Name      = std.Name
	Prefix    = std.Prefix
	Flag      = std.Flag
	SetOutput = std.SetOutput
	SetLevel  = std.SetLevel
	SetName   = std.SetName
	SetPrefix = std.SetPrefix
	SetFlag   = std.SetFlag

	// Method Set
	Fatal = std.Fatal
	Panic = std.Panic
	Error = std.Error
	Warn  = std.Warn
	Info  = std.Info
	Debug = std.Debug
	Trace = std.Trace

	Fatalf = std.Fatalf
	Panicf = std.Panicf
	Errorf = std.Errorf
	Warnf  = std.Warnf
	Infof  = std.Infof
	Debugf = std.Debugf
	Tracef = std.Tracef
)