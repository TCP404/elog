package elog

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

const (
	RegDate         = `[0-9][0-9][0-9][0-9]/[0-9][0-9]/[0-9][0-9]\s*`
	RegTime         = `[0-9][0-9]:[0-9][0-9]:[0-9][0-9]\s*`
	RegMicroseconds = `\.[0-9][0-9][0-9][0-9][0-9][0-9]\s*`
	RegLevel        = `\x1b\[\d;[0-9][0-9];[0-9][0-9]m(\s+)(\w+)(\s+)\x1b\[0m\s*`
	RegPrefix       = TEST_PREFIX + " "
	RegLine         = `(\d+)\s*`
	RegLongfile     = `.*/[A-Za-z0-9_\-]+\.go:` + RegLine
	RegShortfile    = `[A-Za-z0-9_\-]+\.go:` + RegLine
)

type in struct {
	name   string
	level  logLevel
	flag   int
	prefix string
	order  []logOrder
}
type tester struct {
	in
	pattern string
}

const (
	TEST_PREFIX = "PREFIX"
)

var tests = []tester{
	{
		in{name: "t1", level: ErrorLevel, flag: 0},
		""},
	{
		in{name: "t2", level: DebugLevel, flag: 0, prefix: TEST_PREFIX},
		""},
	{
		in{name: "t3", level: InfoLevel, flag: Lmsgprefix, prefix: TEST_PREFIX},
		RegPrefix},
	{
		in{name: "t4", level: InfoLevel, flag: Llevel | LlevelLabelColor},
		RegLevel},
	{
		in{name: "t5", level: InfoLevel, flag: Ldate},
		RegDate},
	{
		in{name: "t6", level: TraceLevel, flag: Ltime},
		RegTime},
	{
		in{name: "t7", level: WarnLevel, flag: Ltime | Lmsgprefix, prefix: TEST_PREFIX},
		RegTime + RegPrefix},
	{
		in{name: "t8", level: InfoLevel, flag: Ltime | Lmicroseconds, prefix: TEST_PREFIX},
		RegTime + RegMicroseconds},
	{ // microsec implies time
		in{name: "t9", level: InfoLevel, flag: Lmicroseconds},
		RegTime + RegMicroseconds + " "},
	{
		in{name: "t10", level: ErrorLevel, flag: Llongfile},
		RegLongfile},
	{
		in{name: "t11", level: ErrorLevel, flag: Lshortfile},
		RegShortfile},
	{ // shortfile overrides longfile
		in{name: "t11", level: ErrorLevel, flag: Llongfile | Lshortfile},
		RegShortfile},

	{in{name: "t12", level: ErrorLevel, flag: Ldate | Ltime | Lmicroseconds | Llevel | LlevelLabelColor | Llongfile, prefix: TEST_PREFIX}, RegDate + RegTime + RegMicroseconds + RegLevel + RegLongfile},
	{in{name: "t13", level: ErrorLevel, flag: Ldate | Ltime | Lmicroseconds | Llevel | LlevelLabelColor | Lshortfile, prefix: TEST_PREFIX}, RegDate + RegTime + RegMicroseconds + RegLevel + RegShortfile},
	{in{name: "t14", level: ErrorLevel, flag: Ldate | Ltime | Lmicroseconds | Llevel | LlevelLabelColor | Llongfile | Lmsgprefix, prefix: TEST_PREFIX}, RegDate + RegTime + RegMicroseconds + RegLevel + RegLongfile + RegPrefix},
	{in{name: "t15", level: ErrorLevel, flag: Ldate | Ltime | Lmicroseconds | Llevel | LlevelLabelColor | Lshortfile | Lmsgprefix, prefix: TEST_PREFIX}, RegDate + RegTime + RegMicroseconds + RegLevel + RegShortfile + RegPrefix},

	{ // test order
		in{name: "t16", level: ErrorLevel, flag: Lmsgprefix | Ldate | Lshortfile, prefix: TEST_PREFIX, order: []logOrder{OrderLevel, OrderPrefix, OrderDate, OrderPath}},
		RegPrefix + RegDate + RegShortfile},

	{
		in{name: "t17", level: ErrorLevel, flag: Lmsgprefix | Ldate | Lshortfile | Llevel | LlevelLabelColor, prefix: TEST_PREFIX, order: []logOrder{OrderLevel, OrderPrefix, OrderDate, OrderPath}},
		RegLevel + RegPrefix + RegDate + RegShortfile},
}

func testPrint(t *testing.T, name string, level logLevel, flag int, prefix string, order []logOrder, pattern string, useFormat bool) {
	var buf bytes.Buffer
	l := New(level, OOutput(&buf), OFlag(flag), OPrefix(prefix), OOrder(order...))
	if useFormat {
		switch level {
		case ErrorLevel:
			l.Error("hello", 18, "word")
		case WarnLevel:
			l.Warn("hello", 18, "word")
		case InfoLevel:
			l.Info("hello", 18, "word")
		case DebugLevel:
			l.Debug("hello", 18, "word")
		case TraceLevel:
			l.Trace("hello", 18, "word")
		}
	} else {
		switch level {
		case ErrorLevel:
			l.Errorf("hello %d word", 18)
		case WarnLevel:
			l.Warnf("hello %d word", 18)
		case InfoLevel:
			l.Infof("hello %d word", 18)
		case DebugLevel:
			l.Debugf("hello %d word", 18)
		case TraceLevel:
			l.Tracef("hello %d word", 18)
		}
	}

	got := buf.String()
	got = got[0 : len(got)-1]
	pattern = `^` + pattern + `hello 18 word\s*$`
	matched, err := regexp.MatchString(pattern, got)
	if err != nil {
		t.Errorf(`%s: pattern did not compile: %q`, name, err)
	}
	if !matched {
		t.Errorf(`%s: log output want: %s[ %q ]%s , got %s[ %q ]%s`, name, _blue, pattern, color_, _green, got, color_)
	}
	SetOutput(os.Stderr)
}

func TestDefault(t *testing.T) {
	if got := Default(); got != std {
		t.Errorf("Default [%p] should be std [%p]", got, std)
	}
}

func TestAll(t *testing.T) {
	for _, tc := range tests {
		testPrint(t, tc.name, tc.level, tc.flag, tc.prefix, tc.order, tc.pattern, true)
		testPrint(t, tc.name, tc.level, tc.flag, tc.prefix, tc.order, tc.pattern, false)
	}
}

func TestExtend(t *testing.T) {
	var b bytes.Buffer
	parent := New(InfoLevel, OOutput(&b), OFlag(Llevel|Ldate), OPrefix("Test: "), OOrder(OrderDate, OrderLevel))
	child := parent.Extend()
	if !reflect.DeepEqual(parent, child) {
		t.Errorf("logger child has some different with logger parent.\n child:  %q,\n parent: %q", child, parent)
	}
	child.SetOrder(OrderMsg, OrderLevel)
	if reflect.DeepEqual(child, parent) {
		t.Error("the change of logger child affected logger parent")
	}
	grandchild := child.Extend(OPrefix("Boii: "))
	if grandchild.Prefix() != "Boii: " {
		t.Errorf("logger grandchild's prefix expected: `Boii: `, but got: %v", grandchild.Prefix())
	}
}

func TestMethodChaining(t *testing.T) {
	var b bytes.Buffer
	parent := New(InfoLevel).SetFlag(Llevel).SetName("chaining").SetOutput(&b)
	if parent.Flag() != Llevel || parent.Name() != "chaining" {
		t.Errorf("the method chaining may have some problem when logger parent creating. parent: %q", parent)
	}
	child := parent.Extend().AddFlag(Ldate)
	if child.Flag() != Llevel|Ldate {
		t.Errorf("the method chaining may have some problem when logger child extending. child:  %q", child)
	}
}

func TestOut(t *testing.T) {
	const testString = "test"
	var b bytes.Buffer
	l := New(InfoLevel, OOutput(&b))
	l.AddFlag(Llevel)
	l.Warn(testString)
	if expect := levelMap[WarnLevel].levelLabel + testString + "\n"; b.String() != expect {
		t.Errorf("log output should match %q, but got %q", expect, b.String())
	}
}

func TestOutRace(t *testing.T) {
	var b bytes.Buffer
	l := New(InfoLevel, OOutput(&b))
	for i := 0; i < 100; i++ {
		go func() {
			l.SetFlag(0)
		}()
	}
}

func TestFlagSetting(t *testing.T) {
	var b bytes.Buffer
	l := New(InfoLevel, OOutput(&b), OFlag(LstdFlags))

	f := l.Flag()
	if f != LstdFlags {
		t.Errorf("Flags 1: expected %x got %x", LstdFlags, f)
	}

	l.SetFlag(f | Lmicroseconds)
	f = l.Flag()
	if f != LstdFlags|Lmicroseconds {
		t.Errorf("Flags 2: expected %x got %x", LstdFlags|Lmicroseconds, f)
	}

	l.AddFlag(Lmsgcolor)
	f = l.Flag()
	if f != LstdFlags|Lmicroseconds|Lmsgcolor {
		t.Errorf("Flags 3: expected %x got %x", LstdFlags|Lmicroseconds|Lmsgcolor, f)
	}

	l.SubFlag(Lmsgcolor)
	f = l.Flag()
	if f != LstdFlags|Lmicroseconds {
		t.Errorf("Flags 4: expected %x got %x", LstdFlags|Lmicroseconds, f)
	}
}

func TestPrefixSetting(t *testing.T) {
	var b bytes.Buffer
	l := New(InfoLevel, OOutput(&b), OFlag(LstdFlags|Lmsgprefix), OPrefix("Test: "))

	p := l.Prefix()
	if p != "Test: " {
		t.Errorf(`Prefix 1: expected "Test: " got %q`, p)
	}
	l.SetPrefix("Boii: ")
	p = l.Prefix()
	if p != "Boii: " {
		t.Errorf(`Prefix 2: expected "Boii: " got %q`, p)
	}

	l.Warn("test string")
	pattern := "^" + RegDate + RegTime + RegLevel + RegShortfile + "Boii: " + "test string\n$"
	got := b.Bytes()
	matched, err := regexp.Match(pattern, got)
	if err != nil {
		t.Fatalf("pattern %q did not compile: %s", pattern, err)
	}
	if !matched {
		t.Errorf("message did not match pattern. \nMessage: `test string`, \nPattern: %q, \ngot: %q", pattern, got)
	}
}

func TestOrderSetting(t *testing.T) {
	var b bytes.Buffer
	l := New(InfoLevel, OOutput(&b))

	o := l.Order()
	if len(o) != 0 {
		t.Errorf("Order1: expected 0 got %q", o)
	}

	l.SetOrder(OrderLevel, OrderTime, OrderDate)
	o = l.Order()
	if len(o) != 3 {
		t.Errorf("Order2: expected 3 got %q", o)
	}

	l.Warn("test string")
	pattern := "test string\n" // 没有设置 flag 时，设置 order 没有意义
	got := b.Bytes()
	b.Reset()
	matched, err := regexp.Match(pattern, got)
	if err != nil {
		t.Fatalf("pattern %q did not compile: %s", pattern, err)
	}
	if !matched {
		t.Errorf("message did not match pattern. \nMessage: `test string`, \nPattern: %q, \ngot: %q", pattern, got)
	}

	l.SetFlag(Llevel | LlevelLabelColor | Ldate | Ltime)
	l.Warn("test string")
	pattern = "^" + RegLevel + RegTime + RegDate + "test string\n$" // 设置了 flag，order 才会生效
	got = b.Bytes()
	b.Reset()
	matched, err = regexp.Match(pattern, got)
	if err != nil {
		t.Fatalf("pattern %q did not compile: %s", pattern, err)
	}
	if !matched {
		t.Errorf("message did not match pattern. \nMessage: `test string`, \nPattern: %q, \ngot: %q", pattern, got)
	}

	l.AddFlag(Lmsgprefix)
	l.SetPrefix(TEST_PREFIX)
	l.Warn("test string")
	pattern = "^" + RegLevel + RegTime + RegDate + RegPrefix + "test string\n$" // 再次额外增加 flag，依然有效
	got = b.Bytes()
	b.Reset()
	l.SubFlag(Lmsgprefix)
	matched, err = regexp.Match(pattern, got)
	if err != nil {
		t.Fatalf("pattern %q did not compile: %s", pattern, err)
	}
	if !matched {
		t.Errorf("message did not match pattern. \nMessage: `test string`, \nPattern: %q, \ngot: %q", pattern, got)
	}

	l.SetOrder(OrderTime, OrderLevel)
	l.SetFlag(Llevel | LlevelLabelColor | Ltime)
	l.Warn("test string")
	pattern = "^" + RegTime + RegLevel + "test string\n$" // SetOrder 后应覆盖之前的 order
	got = b.Bytes()
	b.Reset()
	matched, err = regexp.Match(pattern, got)
	if err != nil {
		t.Fatalf("pattern %q did not compile: %s", pattern, err)
	}
	if !matched {
		t.Errorf("message did not match pattern. \nMessage: `test string`, \nPattern: %q, \ngot: %q", pattern, got)
	}
}

func TestUTCFlag(t *testing.T) {
	var b bytes.Buffer
	l := New(InfoLevel, OOutput(&b), OPrefix("Boii: "), OFlag(Ldate|Ltime|LUTC|Llevel|LlevelLabelColor))

	now := time.Now().UTC()
	l.Info("Hello")

	label := levelMap[InfoLevel].levelLabelColor + levelMap[InfoLevel].levelLabel + color_
	want := fmt.Sprintf("%d/%.2d/%.2d %.2d:%.2d:%.2d "+label+"Hello\n",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	got := b.String()
	if got == want {
		return
	}

	// 可能会有细微时差，所以加一秒再试一次
	now = now.Add(time.Second)
	want = fmt.Sprintf("%d/%.2d/%.2d %.2d:%.2d:%.2d "+label+"Hello\n",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	got = b.String()
	if got == want {
		return
	}

	t.Errorf("\n got:  %q \n want: %q", got, want)
}

func TestEmptyPrintCreatesLine(t *testing.T) {
	var b bytes.Buffer
	l := New(InfoLevel, OOutput(&b), OPrefix("Boii:"), OFlag(Ldate|Ltime|Lmsgprefix))
	l.Info()
	l.Info("non-empty")
	output := b.String()
	if n := strings.Count(output, "Boii:"); n != 2 {
		t.Errorf("expected 2 headers, got %d", n)
	}
	if n := strings.Count(output, "\n"); n != 2 {
		t.Errorf("expected 2 lines, got %d", n)
	}
}

func BenchmarkItoa(b *testing.B) {
	dst := make([]byte, 0, 64)
	for i := 0; i < b.N; i++ {
		dst = dst[0:0]
		itoa(&dst, 2022, 4)   // year
		itoa(&dst, 1, 2)      // month
		itoa(&dst, 25, 2)     // day
		itoa(&dst, 16, 2)     // hour
		itoa(&dst, 44, 2)     // minute
		itoa(&dst, 42, 2)     // second
		itoa(&dst, 123456, 6) // microsecond
	}
}

func BenchmarkPrint(b *testing.B) {
	const testString = "Hello"
	var buf bytes.Buffer
	l := New(InfoLevel, OFlag(LstdFlags))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		l.Info(testString)
	}
}

func BenchmarkPrintNoFlag(b *testing.B) {
	const testString = "Hello"
	var buf bytes.Buffer
	l := New(InfoLevel, OOutput(&buf))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		l.Info(testString)
	}
}
