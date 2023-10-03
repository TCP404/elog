package elog

import (
	"bytes"
	"fmt"
)

func ExamplePrint() {
	var b bytes.Buffer
	l := New(TraceLevel, OOutput(&b), OFlag(Lshortfile))

	l.Trace("This is the TRACE level. It is often used to print loop variables.")
	l.Debug("This is the DEBUG level. It is usually used for sequential debugging.")
	l.Info("This is the INFO level. It is often used to print some infomation such as database connected.")
	l.Warn("This is the WARN level. It is usually used to print some warning infomation.")
	l.Error("This is the ERROR level. It is often used for print some error infomation.")
	fmt.Println(b.String())

	// Output:
	// example_test.go:12 This is the TRACE level. It is often used to print loop variables.
	// example_test.go:13 This is the DEBUG level. It is usually used for sequential debugging.
	// example_test.go:14 This is the INFO level. It is often used to print some infomation such as database connected.
	// example_test.go:15 This is the WARN level. It is usually used to print some warning infomation.
	// example_test.go:16 This is the ERROR level. It is often used for print some error infomation.
}

func ExampleSetOrder() {
	var b bytes.Buffer
	l := New(
		InfoLevel, OOutput(&b), OPrefix("Test: "),
		OFlag(Lshortfile|Lmsgprefix),
		OOrder(OrderPrefix, OrderMsg, OrderPath),
	)
	l.Info("You can set the output order by SetOrder().")
	fmt.Println(b.String())

	// Output:
	// Test: You can set the output order by SetOrder(). example_test.go:34
}

func ExampleSetLevel() {
	var b bytes.Buffer
	l := New(InfoLevel, OOutput(&b), OFlag(Lshortfile))

	l.Info(`Info level message will be printed when you set the "InfoLevel"`)
	l.Debug(`Debug level message will not be printed because you set the "InfoLevel"`)
	l.Warn(`Warn level message will be printed because it is higher than Info level`)
	l.Error(`Error level as well`)
	fmt.Println(b.String())

	// Output:
	// example_test.go:45 Info level message will be printed when you set the "InfoLevel"
	// example_test.go:47 Warn level message will be printed because it is higher than Info level
	// example_test.go:48 Error level as well
}

func ExampleSetOutput() {
	// Single output
	var b1 bytes.Buffer
	var b2 bytes.Buffer
	l := New(InfoLevel, OFlag(Lshortfile), OOutput(&b1))

	l.Info(`This is single output example`)
	fmt.Println(b1.String())

	// Multiple output
	b1.Reset()
	l.SetOutput(&b1, &b2)
	l.Info(`This is multiple output example`)
	fmt.Println(b1.String())
	fmt.Println(b2.String())

	// Output:
	// example_test.go:63 This is single output example
	//
	// example_test.go:69 This is multiple output example
	//
	// example_test.go:69 This is multiple output example
}

func ExampleDefault() {
	var b1 bytes.Buffer
	SetOutput(&b1).SetFlag(Lshortfile)
	Info(`This is the default logger. It is often used for global logging.`)
	SetLevel(DebugLevel)
	Debug(`You can change the level of default logger by SetLevel().`)

	fmt.Println(b1.String())
	// Output:
	// example_test.go:84 This is the default logger. It is often used for global logging.
	// example_test.go:86 You can change the level of default logger by SetLevel().
}
