package mime

import "regexp"

var commaSeparator = regexp.MustCompile("\\s*,\\s*")

// SplitField will take a comma-separated list and return the values (without potential blanks inbetween)
func SplitField(s string) []string {
	return commaSeparator.Split(s, -1)
}
