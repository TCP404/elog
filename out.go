package elog

import (
	"time"
)

func (l *Log) outputDate(flag *int, t time.Time) {
	// 处理日期和时间
	tmpFlag := *flag
	if tmpFlag&Ldate != 0 {
		year, month, day := t.Date()
		itoa(&l.buf, year, 4)
		l.buf = append(l.buf, '/')
		itoa(&l.buf, int(month), 2)
		l.buf = append(l.buf, '/')
		itoa(&l.buf, day, 2)
		addSpace(&l.buf)
		*flag = subFlag(*flag, Ldate)
	}
}

func (l *Log) outputTime(flag *int, t time.Time) {
	tmpFlag := *flag
	if tmpFlag&(Ltime|Lmicroseconds) != 0 {
		hour, min, sec := t.Clock()
		itoa(&l.buf, hour, 2)
		l.buf = append(l.buf, ':')
		itoa(&l.buf, min, 2)
		l.buf = append(l.buf, ':')
		itoa(&l.buf, sec, 2)

		if tmpFlag&Lmicroseconds != 0 {
			l.buf = append(l.buf, '.')
			itoa(&l.buf, t.Nanosecond()/1e3, 6)

		}
		addSpace(&l.buf)
		*flag = subFlag(*flag, Ltime|Lmicroseconds)
	}
}

func (l *Log) outputPath(flag *int, file string, line int) {
	// 处理文件路径
	tmpFlag := *flag
	if tmpFlag&(Lshortfile|Llongfile) != 0 {
		// 如果设置了简洁文件路径，则将文件路径从后往前遍历，找到第一个 '/'，然后取后面的部分
		if tmpFlag&Lshortfile != 0 {
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
		l.buf = append(l.buf, file...)
		// 追加行号
		l.buf = append(l.buf, ':')
		itoa(&l.buf, line, -1)
		// 追加间隔符号，间隔符号后就是打印内容了
		addSpace(&l.buf)
		*flag = subFlag(*flag, Lshortfile|Llongfile)
	}
}

func (l *Log) outputLevel(flag *int, level logLevel) {
	// 处理等级前缀
	tmpFlag := *flag
	if tmpFlag&Llevel != 0 {
		l.buf = append(l.buf, levelMap[level].levelLabel...)
		addSpace(&l.buf)
		*flag = subFlag(*flag, Llevel)
	}
}

func (l *Log) outputPrefix(flag *int) {
	// 处理消息前缀 msgPrefix
	tmpFlag := *flag
	if tmpFlag&Lmsgprefix != 0 {
		l.buf = append(l.buf, l.prefix...)
		addSpace(&l.buf)
		*flag = subFlag(*flag, Lmsgprefix)
	}
}

func (l *Log) outputMsg(written *bool, msg string) {
	if *written {
		return
	}

	if l.flag&Lmsgcolor != 0 {
		setColor(&l.buf, l.level)
	}
	l.buf = append(l.buf, msg...)                 // 将打印内容填充到 buffer 中
	if len(msg) == 0 || msg[len(msg)-1] != '\n' { // 如果打印内容为空或者内容末尾没有换行符，则追加换行符
		l.buf = append(l.buf, '\n')
	}
	if l.flag&Lmsgcolor != 0 {
		unsetColor(&l.buf)
	}
	*written = true
}

func addSpace(buf *[]byte) {
	b := *buf
	if b[len(b)-1] != ' ' {
		*buf = append(*buf, ' ')
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

func setColor(buf *[]byte, level logLevel) {
	*buf = append(*buf, levelMap[level].levelColor...)
}

func unsetColor(buf *[]byte) {
	*buf = append(*buf, color_...)
}
