package elog

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

const (
	RegDate         = `[0-9][0-9][0-9][0-9]/[0-9][0-9]/[0-9][0-9]`
	RegTime         = `[0-9][0-9]:[0-9][0-9]:[0-9][0-9]`
	RegMicroseconds = `\.[0-9][0-9][0-9][0-9][0-9][0-9]`
	RegLabel        = `\x1b\[\d;[0-9][0-9];[0-9][0-9]m(\s+)(\w+)(\s+)\x1b\[0m\s*`
	RegLine         = `(60|62) |`
	RegLongfile     = `.*/[A-Za-z0-9_\-]+\.go:` + RegLine
	RegShortfile    = `[A-Za-z0-9_\-]+\.go:` + RegLine
)

type in struct {
	name   string
	level  logLevel
	flag   int
	prefix string
}
type tester struct {
	in
	pattern string
}

const (
	TEST_PREFIX = "PREFIX "
)

var tests = []tester{
	{
		in{"t1", ErrorLevel, 0, ""},
		RegLabel},
	{
		in{"t2", DebugLevel, 0, TEST_PREFIX},
		RegLabel},
	{
		in{"t3", InfoLevel, Ldate, ""},
		RegDate + " " + RegLabel + " "},
	{
		in{"t4", TraceLevel, Ltime, ""},
		RegTime + " " + RegLabel + " "},
	{
		in{"t5", WarnLevel, Ltime | Lmsgprefix, TEST_PREFIX},
		RegTime + " " + RegLabel + TEST_PREFIX},
	{
		in{"t6", InfoLevel, Ltime | Lmicroseconds, TEST_PREFIX},
		RegTime + RegMicroseconds + " " + RegLabel},
	{ // microsec implies time
		in{"t7", InfoLevel, Lmicroseconds, ""},
		RegTime + RegMicroseconds + " " + RegLabel},
	{
		in{"t8", ErrorLevel, Llongfile, ""},
		RegLabel + RegLongfile},
	{
		in{"t9", ErrorLevel, Lshortfile, ""},
		RegLabel + RegShortfile},
	{ // shortfile overrides longfile
		in{"t10", ErrorLevel, Llongfile | Lshortfile, ""},
		RegLabel + RegShortfile},

	{in{"t11", ErrorLevel, Ldate | Ltime | Lmicroseconds | Llongfile, TEST_PREFIX}, RegDate + " " + RegTime + RegMicroseconds + " " + RegLabel + RegLongfile},
	{in{"t12", ErrorLevel, Ldate | Ltime | Lmicroseconds | Lshortfile, TEST_PREFIX}, RegDate + " " + RegTime + RegMicroseconds + " " + RegLabel + RegShortfile},
	{in{"t13", ErrorLevel, Ldate | Ltime | Lmicroseconds | Llongfile | Lmsgprefix, TEST_PREFIX}, RegDate + " " + RegTime + RegMicroseconds + " " + RegLabel + RegLongfile + TEST_PREFIX},
	{in{"t14", ErrorLevel, Ldate | Ltime | Lmicroseconds | Lshortfile | Lmsgprefix, TEST_PREFIX}, RegDate + " " + RegTime + RegMicroseconds + " " + RegLabel + RegLongfile + TEST_PREFIX},
}

func testPrint(t *testing.T, name string, level logLevel, flag int, prefix string, pattern string, useFormat bool) {
	buf := new(bytes.Buffer)
	SetOutput(buf)
	SetLevel(level)
	SetFlag(flag)
	SetPrefix(prefix)
	if useFormat {
		switch level {
		case ErrorLevel:
			Error("hello", 18, "word")
		case WarnLevel:
			Warn("hello", 18, "word")
		case InfoLevel:
			Info("hello", 18, "word")
		case DebugLevel:
			Debug("hello", 18, "word")
		case TraceLevel:
			Trace("hello", 18, "word")
		}
	} else {
		switch level {
		case ErrorLevel:
			Errorf("hello %d word", 18)
		case WarnLevel:
			Warnf("hello %d word", 18)
		case InfoLevel:
			Infof("hello %d word", 18)
		case DebugLevel:
			Debugf("hello %d word", 18)
		case TraceLevel:
			Tracef("hello %d word", 18)
		}
	}

	got := buf.String()
	got = got[0 : len(got)-1]
	pattern = "^" + pattern + "hello 18 word$"
	matched, err := regexp.MatchString(pattern, got)
	if err != nil {
		t.Errorf("%s: pattern did not compile: %q", name, err)
	}
	if !matched {
		t.Errorf("%s: log output want: %s[ %q ]%s , got %s[ %q ]%s", name, blue, pattern, color_, green, got, color_)
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
		testPrint(t, tc.name, tc.level, tc.flag, tc.prefix, tc.pattern, true)
		testPrint(t, tc.name, tc.level, tc.flag, tc.prefix, tc.pattern, false)
	}
}

func TestOut(t *testing.T) {
	const testString = "test"
	var b bytes.Buffer
	l := New(&b, InfoLevel)
	l.Warn(testString)
	if expect := _WarnLabel + testString + "\n"; b.String() != expect {
		t.Errorf("log output should match %q is %q", expect, b.String())
	}
}

func TestOutRace(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, InfoLevel)
	for i := 0; i < 100; i++ {
		go func() {
			l.SetFlag(0)
		}()
	}
}

func TestFlagAndPrefixSetting(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, InfoLevel, OFlag(LstdFlags), OPrefix("Test: "))

	f := l.Flag()
	if f != LstdFlags {
		t.Errorf("Flags 1: expected %x got %x", LstdFlags, f)
	}
	l.SetFlag(f | Lmicroseconds)
	f = l.Flag()
	if f != LstdFlags|Lmicroseconds {
		t.Errorf("Flags 2: expected %x got %x", LstdFlags|Lmicroseconds, f)
	}

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
	pattern := "^Boii: " + RegDate + " " + RegTime + RegMicroseconds + " " + RegLabel + RegShortfile + "test string\n"
	matched, err := regexp.Match(pattern, b.Bytes())
	if err != nil {
		t.Fatalf("pattern %q did not compile: %s", pattern, err)
	}
	if !matched {
		t.Errorf(`message did not match pattern. Message: "test string" , Pattern: %q`, pattern)
	}
}

func TestUTCFlag(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, InfoLevel, OPrefix("Boii: "), OFlag(Ldate|Ltime|LUTC))

	now := time.Now().UTC()
	l.Info("Hello")
	want := fmt.Sprintf("%d/%.2d/%.2d %.2d:%.2d:%.2d "+_InfoLabel+"Hello\n",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	got := b.String()
	if got == want {
		return
	}

	// 可能会有细微时差，所以加一秒再试一次
	now = now.Add(time.Second)
	want = fmt.Sprintf("%d/%.2d/%.2d %.2d:%.2d:%.2d "+_InfoLabel+"Hello\n",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second())
	got = b.String()
	if got == want {
		return
	}

	t.Errorf("\n got:  %q \n want: %q", got, want)
}

func TestEmptyPrintCreatesLine(t *testing.T) {
	var b bytes.Buffer
	l := New(&b, InfoLevel, OPrefix("Boii:"), OFlag(Ldate|Ltime|Lmsgprefix))
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
	l := New(&buf, InfoLevel, OFlag(LstdFlags))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		l.Info(testString)
	}
}

func BenchmarkPrintNoFlag(b *testing.B) {
	const testString = "Hello"
	var buf bytes.Buffer
	l := New(&buf, InfoLevel)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		l.Info(testString)
	}
}
