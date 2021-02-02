package fmt

import (
	sysfmt "fmt"
	"reflect"
	"strings"

	"path/filepath"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/qjpcpu/qjson"
)

var (
	Green      = color.New(color.FgGreen, color.Bold).SprintFunc()
	Cyan       = color.New(color.FgCyan, color.Bold).SprintFunc()
	Magenta    = color.New(color.FgMagenta, color.Bold).SprintFunc()
	Yellow     = color.New(color.FgYellow, color.Bold).SprintFunc()
	Red        = color.New(color.FgRed, color.Bold).SprintFunc()
	Blue       = color.New(color.FgBlue, color.Bold).SprintFunc()
	colorFuncs = []func(a ...interface{}) string{
		Green,
		Cyan,
		Magenta,
		Yellow,
		Red,
		Blue,
		color.New(color.FgWhite, color.BgBlack, color.Bold).SprintFunc(),
		color.New(color.FgBlack, color.BgWhite, color.Bold).SprintFunc(),
	}
)

type Printer func(format string, args ...interface{})

func (p Printer) PrependTime() Printer {
	return func(format string, args ...interface{}) {
		p(timeStr(time.Now())+" "+format, args...)
	}
}

func (p Printer) PrependFile() Printer {
	return func(format string, args ...interface{}) {
		for i := 1; i < 100; i++ {
			_, file, line, ok := runtime.Caller(i)
			if !ok {
				file = "???"
				line = 0
			} else if !strings.Contains(file, `github.com/qjpcpu/common.v2/fmt`) {
				file = filepath.Base(file)
				args = append([]interface{}{file, line}, args...)
				break
			}
		}
		p("%s:%d "+format, args...)
	}
}

var (
	// Print with color
	Print                = Printer(rawPrint)
	Printf               = Printer(rawPrint)
	PrintWithFile        = Printer(rawPrint).PrependFile()
	PrintWithFileAndTime = Printer(rawPrint).PrependTime().PrependFile()
	// PrintJSON complex value to json with color
	PrintJSON = Printer(rawPrintJSON)
	// PrintWithTime print with time
	PrintWithTime = Printer(rawPrint).PrependTime()
	// PrintJSONWithTime print with time
	PrintJSONWithTime = Printer(rawPrintJSON).PrependTime()
)

func rawPrint(format string, args ...interface{}) {
	if len(args) == 0 {
		sysfmt.Println(format)
		return
	}
	sysfmt.Printf(rewriteFormat(format, nil), colorArgs(rewriteArgsToString(format, args, false), nil)...)
}

func Println(args ...interface{}) {
	f := strings.TrimSuffix(strings.Repeat("%v ", len(args)), " ")
	rawPrintJSON(f, args...)
}

// PrintObject with color
func PrintObject(v interface{}) {
	sysfmt.Println(string(qjson.PrettyMarshalWithIndent(v)))
}

func rawPrintJSON(format string, args ...interface{}) {
	if len(args) == 0 {
		sysfmt.Println(format)
		return
	}
	withoutColor := make(map[int]bool)
	for i := range args {
		withoutColor[i] = isComplexValue(args[i])
	}
	sysfmt.Printf(rewriteFormat(format, nil), colorArgs(rewriteArgsToString(format, args, true), withoutColor)...)
}

func rewriteArgsToString(format string, args []interface{}, complextToJSON bool) []interface{} {
	rewriteFormat(format, func(idx int, sysfmtToken string) {
		if idx >= len(args) {
			return
		}
		if complextToJSON && isComplexValue(args[idx]) {
			args[idx] = string(qjson.PrettyMarshal(args[idx]))
		} else {
			args[idx] = sysfmt.Sprintf(sysfmtToken, args[idx])
		}
	})
	return args
}

func rewriteFormat(format string, cb func(int, string)) string {
	if cb == nil {
		cb = func(int, string) {}
	}
	var idx int

	var newsysfmt []rune
	runes := []rune(format)
	for i := 0; i < len(runes); {
		/* skip double % */
		if runes[i] == '%' && i < len(runes)-1 && runes[i+1] == '%' {
			newsysfmt = append(newsysfmt, runes[i], runes[i+1])
			i += 2
			continue
		}
		/* find format token like %[^a-zA-Z] */
		if runes[i] == '%' {
			j := i + 1
			for ; j < len(runes); j++ {
				if (runes[j] >= 'A' && runes[j] <= 'Z') || (runes[j] >= 'a' && runes[j] <= 'z') {
					break
				}
			}
			cb(idx, string(runes[i:j+1]))
			idx++
			newsysfmt = append(newsysfmt, '%', 's')
			i = j + 1
			continue
		}
		newsysfmt = append(newsysfmt, runes[i])
		i++
	}
	/* always end with newline */
	if newsysfmt[len(newsysfmt)-1] != '\n' {
		newsysfmt = append(newsysfmt, '\n')
	}
	return string(newsysfmt)
}

func colorArgs(args []interface{}, withoutColor map[int]bool) []interface{} {
	ret := make([]interface{}, len(args))
	for i, v := range args {
		if withoutColor != nil && withoutColor[i] {
			ret[i] = args[i]
			continue
		}
		ret[i] = colorFuncs[i%len(colorFuncs)](v)
	}
	return ret
}

func isComplexValue(v interface{}) bool {
	typ := reflect.TypeOf(v)
	if typ == nil {
		return false
	}
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Map, reflect.Struct, reflect.Slice:
		return true
	default:
		return false
	}
}

func timeStr(tm time.Time) string {
	return tm.Format("15:04:05")
}
