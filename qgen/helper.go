package qgen

import (
	"fmt"
	"strconv"
	"time"
)

func ConvertToEscapeString(obj interface{}, def string) (res string) {
	switch v := obj.(type) {
	case int, int64, int32, float64, float32, bool:
		res = fmt.Sprintf("%v", v)
	case time.Time:
		res = (obj.(time.Time)).Format("2006-01-02 15:04:05")
	case string:
		res = strconv.Quote(fmt.Sprintf("%v", v))
	default:
		res = def
	}

	return
}

const (
	VIEW_DEFAULT = `\<view::[\w_]*\s?\/\>`
	VIEW_CURLY   = `\<view::\s*\{[\n\s\w\.\>\<\"\:\;\_\(\)\+\=\,]*\}[\s\n]?\/\>`
	TABLE        = `\<tb\:[\w_]+\s*\/\>`
	JOIN         = `\<join\:[\w_]+[\s\n]*\{[\_\:\@\.\w\s\n\=\'\"\;\,]*\}[\s\n]*\/\>`
	DOB_QUOTE    = `\"[\_\:\@\.\w\s\n\=]*\"`
	COND_PLAIN   = `\<cond\:[\w\.]*\[[\w\.\_]*\]\s?\/\>`
	COND_MODIF   = `\<cond\:[\w\.]+\[[\w\.\_]+\]\s?\>[\s\n\(\_\:\w\.\+\)]*<\/cond:[\w]+\>`
	TABLE_VAR    = `__::[\w\_]+.[\w\_]+__`
	SET_DEFAULT  = `\<set::[\w_]*\s?\/\>`
)
