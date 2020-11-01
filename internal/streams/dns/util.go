package dns

import (
	"regexp"
	"time"
)

var DotRegex = regexp.MustCompile("\\.")

// Dotify will include dots every 57 characters
func Dotify(buf string) (res string) {
	for len(buf) > 57 {
		res = buf[0:57] + "."
		buf = buf[57:]
	}
	res = res + buf

	return
}

// Undotify will remove the dots from the given string
func Undotify(buf string) string {
	return DotRegex.ReplaceAllString(buf, "")
}

// secs will return the supplied parameter as duration in secs
func secs(i int) time.Duration {
	return time.Second * time.Duration(i)
}
