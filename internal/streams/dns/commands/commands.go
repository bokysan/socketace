package commands

import (
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

type Command struct {
	Code        byte
	NeedsUserId bool // Defines if the command needs user ID in the query sring or not
	NewRequest  func() Request
	NewResponse func() Response
}

type LazyMode byte

const (
	LazyModeOn  LazyMode = 'l'
	LazyModeOff LazyMode = 'i'
)

var (
	// Command 0123456789abcdef are reserved for user IDs
	CmdLogin = Command{
		Code: 'l',
	}
	CmdTestMultiQuery = Command{
		Code: 'm',
	}
)

var Commands = []Command{
	CmdVersion,
	CmdLogin,
	CmdSetOptions,
	CmdTestDownstreamFragmentSize,
	CmdTestDownstreamEncoder,
	CmdTestUpstreamEncoder,
	CmdTestMultiQuery,
	CmdPacket,
	CmdError,
}

// String will retun the command code as string, e.g. 'z', 's', 'v'...
func (c Command) String() string {
	return string(c.Code)
}

// String will retun the command code as string, e.g. 'z', 's', 'v'...
func (c Command) Byte() byte {
	return c.Code
}

// ValidateType will check if the supplied string starts with the given command type and return an error if its not.
func (c Command) ValidateType(data []byte) error {
	if !c.IsOfType(data) {
		return errors.Errorf("Invalid command type. Expected %v, got, %v", c, data[0])
	}
	return nil
}

// IsOfType will check if the supplied string starts with the given command type
func (c Command) IsOfType(data []byte) bool {
	if data == nil || len(data) < 0 {
		return false
	}
	if data[0] == c.Code {
		return true
	}
	if strings.ToLower(string(data[0:1]))[0] == c.Code {
		return true
	}
	return false
}

// EncodeRequestHeader will prepare a common request header used by all commands
func EncodeRequestHeader(c Command, userId uint16) []byte {

	var hostname []byte

	hostname = append(hostname, c.Code)
	hostname = append(hostname, []byte(randomChars())...)

	if c.NeedsUserId {
		u := EncodeUserId(userId)
		hostname = append(hostname, []byte(u)...)
	}

	return hostname
}

func EncodeUserId(userId uint16) string {
	const MaxUserId = 36 * 36
	userId = userId % MaxUserId // Make sure it's not over 1296
	u := strconv.FormatInt(int64(userId), 36)
	for len(u) < 2 {
		u = "0" + u
	}
	return u
}

func DecodeRequestHeader(c Command, req []byte) (remaining []byte, userId uint16, err error) {
	err = c.ValidateType(req)
	if err != nil {
		return req, 0, err
	}

	req = req[4:] // Remove command type + cache

	if c.NeedsUserId {
		u, err := strconv.ParseUint(string(req[0:2]), 36, 16)
		if err != nil {
			return req, 0, err
		} else {
			userId = uint16(u)
		}

		// Remove user ID
		req = req[2:]
	}

	return req, userId, nil
}
