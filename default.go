package elog

func Default() *Log { return std }

var std *Log = New(InfoLevel, OName("Global"), OPrefix("[eLog]"), OFlag(LstdFlags))

var (
	// Getter & Setter
	Output    = std.Output
	Level     = std.Level
	Name      = std.Name
	Prefix    = std.Prefix
	Order     = std.Order
	Flag      = std.Flag
	SetOutput = std.SetOutput
	SetLevel  = std.SetLevel
	SetName   = std.SetName
	SetPrefix = std.SetPrefix
	SetOrder  = std.SetOrder
	SetFlag   = std.SetFlag
	AddFlag   = std.AddFlag
	SubFlag   = std.SubFlag

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
