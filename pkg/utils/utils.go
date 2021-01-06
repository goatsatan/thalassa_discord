package utils

import (
	"github.com/volatiletech/null/v8"
)

func InterfaceToNullString(x interface{}) null.String {
	stringParse, ok := x.(string)
	if !ok {
		return null.String{}
	}
	return null.StringFrom(stringParse)
}
