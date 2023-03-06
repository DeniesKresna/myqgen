package qgen

import (
	"fmt"
)

func ConvertToString(obj interface{}, def string) (res string) {
	switch v := obj.(type) {
	case int, int64, int32, float64, float32, bool:
		return fmt.Sprintf("%v", v)
	case string:
		return fmt.Sprintf("'%v'", v)
	default:
		return def
	}
}

const (
	VIEW_DEFAULT = `\<view::[\w_]*\s?\/\>`
	VIEW_CURLY   = `\<view::\s*\{[\n\s\w\.\>\<\"\:\;\_\(\)\+\=\,]*\}[\s\n]?\/\>`
	TABLE        = `\<tb\:[\w_]+\s*\/\>`
	JOIN         = `\<join\:[\w_]+[\s\n]*\{[\_\:\@\.\w\s\n\=\'\"\;\,]*\}[\s\n]*\/\>`
	DOB_QUOTE    = `\"[\_\:\@\.\w\s\n\=]*\"`
	COND_PLAIN   = `\<cond\:[\w\.]*\[[\w\.\_]*\]\s?\/\>`
	COND_MODIF   = `\<cond\:[\w\.]+\[[\w\.\_]+\]\s?\>[\s\n\(\_\:\w\.\+\)]*<\/cond:[\w]+\>`
)
