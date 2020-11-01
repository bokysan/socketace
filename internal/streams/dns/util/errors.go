package util

import "github.com/pkg/errors"

var (
	ErrTooLong  = errors.New("token too long")
	ErrCaseSwap = errors.New("case swap, no need for any further test")
)
