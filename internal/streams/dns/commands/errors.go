package commands

import "errors"

// Declare a few of standard error responses
var (
	BadVersion    = errors.New("BADVER")
	BadLen        = errors.New("BADLEN")
	BadIp         = errors.New("BADIP")
	BadCommand    = errors.New("BADCOMMAND")
	BadCodec      = errors.New("BADCODEC")
	BadFrag       = errors.New("BADFRAG")
	BadUser       = errors.New("BADUSER")
	BadConn       = errors.New("BADCONN")
	BadServerFull = errors.New("VFUL")
	NoData        = errors.New("VOK")
	VersionOk     = errors.New("VACK")
	VersionNotOk  = errors.New("VNAK")
	LazyModeOk    = errors.New("LACK")
	ErrTimeout    = errors.New("TIMEOUT")
)

var BadErrors = []error{
	BadVersion, BadLen, BadIp, BadCommand, BadCodec, BadFrag,
	BadUser, BadConn, BadServerFull,
	NoData, VersionOk, VersionNotOk, LazyModeOk, ErrTimeout,
}
