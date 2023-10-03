package elog

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

type Log struct {
	mu     sync.RWMutex
	output io.Writer // 日志输出方式
	level  logLevel  // 日志最低等级，低于这个等级的日志不会被打印
	name   string    // 日志对象名称
	flag   int       // 日志对象属性
	prefix string    // 日志前缀
	buf    []byte
	// 日志输出顺序，如果没有设置输出顺序，输出内容项以 flag 为准，输出顺序为默认顺序
	// 如果设置了输出顺序，输出内容项先以 order 为准，输出顺序以 order 为准，再以 flag 为准，输出顺序为剩余的默认顺序
	order []logOrder
}

var _ Logger = &Log{}

// Out is a core method
func (l *Log) Out(calldepth int, level logLevel, msg string) error {
	now := time.Now()
	var file string
	var line int
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.flag&LUTC != 0 {
		now = now.UTC()
	}
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

	var (
		unwriteFlag int  = l.flag
		msgWritten  bool // msg 有可能 order 里有，
	)
	if len(l.order) > 0 {
		for _, order := range l.order {
			switch order {
			case OrderDate:
				l.outputDate(&unwriteFlag, now)
			case OrderTime:
				l.outputTime(&unwriteFlag, now)
			case OrderLevel:
				l.outputLevel(&unwriteFlag, level)
			case OrderPrefix:
				l.outputPrefix(&unwriteFlag)
			case OrderPath:
				l.outputPath(&unwriteFlag, file, line)
			case OrderMsg:
				l.outputMsg(&msgWritten, level, msg)
			}
		}
	}
	// Default order: Date Time Microseconds Level shortfile/longfile:Line Msgprefix MESSAGE
	// 将格式化头部填充到 buffer 中
	l.outputDate(&unwriteFlag, now)
	l.outputTime(&unwriteFlag, now)
	l.outputLevel(&unwriteFlag, level)
	l.outputPath(&unwriteFlag, file, line)
	l.outputPrefix(&unwriteFlag)
	l.outputMsg(&msgWritten, level, msg)

	setNewLine(&l.buf)
	_, err := l.output.Write(l.buf)
	return err
}

// Create Logger Option
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

func OOrder(order ...logOrder) LogOption {
	return func(logger *Log) {
		logger.order = order
	}
}

func OOutput(w1 io.Writer, w ...io.Writer) LogOption {
	return func(logger *Log) {
		if w1 == nil {
			w1 = os.Stderr
		}
		w = append(w, w1)
		if logger.output != nil {
			w = append(w, logger.output)
		}
		logger.output = io.MultiWriter(w...)
	}
}

func New(level logLevel, options ...LogOption) *Log {
	l := new(Log)
	l.level = level
	for _, opt := range options {
		opt(l)
	}
	if l.output == nil {
		l.output = os.Stderr
	}
	return l
}

func Extend(options ...LogOption) *Log {
	return std.Extend(options...)
}

func (parent *Log) Extend(options ...LogOption) *Log {
	son := new(Log)
	if parent == nil {
		parent = std
	}
	son.output = parent.output
	son.level = parent.level
	son.flag = parent.flag
	son.prefix = parent.prefix
	son.order = make([]logOrder, len(parent.order))
	copy(son.order, parent.order)
	for _, opt := range options {
		opt(son)
	}
	return son
}

// Getter & Setter
func (l *Log) Output() io.Writer {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.output
}
func (l *Log) Level() logLevel {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.level
}
func (l *Log) Name() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.name
}
func (l *Log) Prefix() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.prefix
}
func (l *Log) Order() []logOrder {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.order
}
func (l *Log) Flag() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.flag
}
func (l *Log) SetOutput(w1 io.Writer, w ...io.Writer) *Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	if w1 == nil {
		w1 = os.Stderr
	}
	l.output = io.MultiWriter(append(w, w1)...)
	return l
}
func (l *Log) SetLevel(level logLevel) *Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
	return l
}
func (l *Log) SetName(name string) *Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.name = name
	return l
}
func (l *Log) SetPrefix(prefix string) *Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prefix = prefix
	return l
}
func (l *Log) SetFlag(flag int) *Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flag = flag
	return l
}
func (l *Log) SetOrder(orders ...logOrder) *Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.order = l.order[:0]
	l.order = append(l.order, orders...)
	return l
}

// Manipulate Flag
func (l *Log) AddFlag(flag int) *Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flag = l.flag | flag
	return l
}
func (l *Log) SubFlag(flag int) *Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.flag = l.flag &^ flag
	return l
}

// 这个方法仅限于 Out() 方法用，因为在 Out 方法中已经上锁了，所以这里不能再上锁
func subFlag(flag1 int, flag2 int) int {
	return flag1 &^ flag2
}

// Method Set
func (l *Log) Fatal(v ...any) {
	if l.level <= FatalLevel {
		l.Out(defaultCallDepth, FatalLevel, fmt.Sprintln(v...))
		os.Exit(1)
	}
}
func (l *Log) Panic(v ...any) {
	if l.level <= PanicLevel {
		s := fmt.Sprintln(v...)
		l.Out(defaultCallDepth, PanicLevel, s)
		panic(s)
	}
}
func (l *Log) Error(v ...any) {
	if l.level <= ErrorLevel {
		l.Out(defaultCallDepth, ErrorLevel, fmt.Sprintln(v...))
	}
}
func (l *Log) Warn(v ...any) {
	if l.level <= WarnLevel {
		l.Out(defaultCallDepth, WarnLevel, fmt.Sprintln(v...))
	}
}
func (l *Log) Info(v ...any) {
	if l.level <= InfoLevel {
		l.Out(defaultCallDepth, InfoLevel, fmt.Sprintln(v...))
	}
}
func (l *Log) Debug(v ...any) {
	if l.level <= DebugLevel {
		l.Out(defaultCallDepth, DebugLevel, fmt.Sprintln(v...))
	}
}
func (l *Log) Trace(v ...any) {
	if l.level <= TraceLevel {
		l.Out(defaultCallDepth, TraceLevel, fmt.Sprintln(v...))
	}
}

func (l *Log) Fatalf(format string, v ...any) {
	if l.level <= FatalLevel {
		l.Out(defaultCallDepth, FatalLevel, fmt.Sprintf(format, v...))
		os.Exit(1)
	}
}
func (l *Log) Panicf(format string, v ...any) {
	if l.level <= PanicLevel {
		s := fmt.Sprintf(format, v...)
		l.Out(defaultCallDepth, PanicLevel, s)
		panic(s)
	}
}
func (l *Log) Errorf(format string, v ...any) {
	if l.level <= ErrorLevel {
		l.Out(defaultCallDepth, ErrorLevel, fmt.Sprintf(format, v...))
	}
}
func (l *Log) Warnf(format string, v ...any) {
	if l.level <= WarnLevel {
		l.Out(defaultCallDepth, WarnLevel, fmt.Sprintf(format, v...))
	}
}
func (l *Log) Infof(format string, v ...any) {
	if l.level <= InfoLevel {
		l.Out(defaultCallDepth, InfoLevel, fmt.Sprintf(format, v...))
	}
}
func (l *Log) Debugf(format string, v ...any) {
	if l.level <= DebugLevel {
		l.Out(defaultCallDepth, DebugLevel, fmt.Sprintf(format, v...))
	}
}
func (l *Log) Tracef(format string, v ...any) {
	if l.level <= TraceLevel {
		l.Out(defaultCallDepth, TraceLevel, fmt.Sprintf(format, v...))
	}
}
