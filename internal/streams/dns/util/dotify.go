package util

import (
	"regexp"
)

var DotRegex = regexp.MustCompile("\\.")

// Dotify will include dots every 57 characters
func Dotify(buf []byte) (res []byte) {
	for len(buf) > 57 {
		res = append(res, buf[0:57]...)
		res = append(res, '.')
		buf = buf[57:]
	}
	res = append(res, buf...)
	return
}

// Undotify will remove the dots from the given string
func Undotify(buf string) string {
	return DotRegex.ReplaceAllString(buf, "")
}
