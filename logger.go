package elog

const defaultCallDepth = 2

type logLevel int

// Fatal > Panic > Error > Warn > Info > Debug > Trace > Discard
const (
	Discard    logLevel = iota
	TraceLevel          // 在线调试: 默认情况下，既不打印到终端也不输出到文件。此时，对程序运行效率几乎不产生影响。常用语 for 循环中调试
	DebugLevel          // 终端查看、在线调试: 默认情况下会打印到终端输出，但是不会归档到日志文件。因此，一般用于开发者在程序当前启动窗口上，查看日志流水信息。
	InfoLevel           // 报告程序进度和状态信息: 一般这种信息都是一过性的，不会大量反复输出。例如：连接商用库成功后，可以打印一条连库成功的信息，便于跟踪程序进展信息。
	WarnLevel           // 警告信息: 程序处理中遇到非法数据或者某种可能的错误。该错误是一过性的、可恢复的，不会影响程序继续运行，程序仍处在正常状态。
	ErrorLevel          // 状态错误: 该错误发生后程序仍然可以运行，但是极有可能运行在某种非正常的状态下，导致无法完成全部既定的功能。
	PanicLevel          // 致命的错误: 表明程序遇到了致命的错误，可以不马上终止运行，依赖 recover()。
	FatalLevel          // 致命的错误: 表明程序遇到了致命的错误，必须马上终止运行。
)

const (
	_FatalLabel = "FATAL"
	_PanicLabel = "PANIC"
	_ErrorLabel = "ERROR"
	_WarnLabel  = "WARN "
	_InfoLabel  = "INFO "
	_DebugLabel = "DEBUG"
	_TraceLabel = "TRACE"
)

const (
	_red     = "\x1b[1;31;40m "
	_green   = "\x1b[1;32;40m "
	_yellow  = "\x1b[1;33;40m "
	_blue    = "\x1b[1;34;40m "
	_magenta = "\x1b[1;35;40m "
	_cyan    = "\x1b[1;36;40m "
	_while   = "\x1b[1;37;40m "

	Fatal_ = "\x1b[0;30;45m "
	Panic_ = "\x1b[1;37;45m "
	Error_ = "\x1b[1;37;41m "
	Warn_  = "\x1b[0;30;43m "
	Info_  = "\x1b[0;30;46m "
	Debug_ = "\x1b[0;37;44m "
	Trace_ = "\x1b[0;30;42m "

	color_ = " \x1b[0m "
)

var levelMap = map[logLevel]struct {
	levelLabel      string
	levelLabelColor string
	levelColor      string
}{
	FatalLevel: {_FatalLabel, Fatal_, _magenta},
	PanicLevel: {_PanicLabel, Panic_, _magenta},
	ErrorLevel: {_ErrorLabel, Error_, _red},
	WarnLevel:  {_WarnLabel, Warn_, _yellow},
	InfoLevel:  {_InfoLabel, Info_, _cyan},
	DebugLevel: {_DebugLabel, Debug_, _blue},
	TraceLevel: {_TraceLabel, Trace_, _green},
}

// Content Order (date、time、level、prefix、filepath、msg)
type logOrder string

const (
	OrderDate   logOrder = "Date"
	OrderTime   logOrder = "Time"
	OrderLevel  logOrder = "Level"
	OrderPrefix logOrder = "Prefix"
	OrderPath   logOrder = "Path"
	OrderMsg    logOrder = "Message"
)

// Flag set include setting of date, time, path, prefix, level, msg
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
	LlevelLabelColor
	LstdFlags = Ldate | Ltime | Lshortfile | Llevel
)

type Logger interface {
	Fatal(...any)
	Panic(...any)
	Error(...any)
	Warn(...any)
	Info(...any)
	Debug(...any)
	Trace(...any)

	Fatalf(string, ...any)
	Panicf(string, ...any)
	Errorf(string, ...any)
	Warnf(string, ...any)
	Infof(string, ...any)
	Debugf(string, ...any)
	Tracef(string, ...any)
}
