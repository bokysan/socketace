package commands

import (
	"github.com/pkg/errors"
	"strings"
)

type Command struct {
	Code        byte
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
	CmdPing = Command{
		Code: 'p',
	}
	CmdTestFragmentSize = Command{
		Code: 'r',
	}
	CmdSetDownstreamFragmentSize = Command{
		Code: 'n',
	}
	CmdSetDownstreamEncoder = Command{
		Code: 'o',
	}
	CmdTestUpstreamEncoder = Command{
		Code: 'z',
	}
	CmdSetUpstreamEncoder = Command{
		Code: 's',
	}
	CmdTestMultiQuery = Command{
		Code: 'm',
	}
)

var Commands = []Command{
	CmdVersion,
	CmdLogin,
	CmdPing,
	CmdTestFragmentSize,
	CmdSetDownstreamFragmentSize,
	CmdTestDownstreamEncoder,
	CmdSetDownstreamEncoder,
	CmdTestUpstreamEncoder,
	CmdSetUpstreamEncoder,
	CmdTestMultiQuery,
}

// RequiresUser returs true if the command requires the user ID
func (c *Command) RequiresUser() bool {
	return c.Code == CmdPing.Code ||
		c.Code == CmdSetDownstreamFragmentSize.Code ||
		c.Code == CmdSetDownstreamEncoder.Code ||
		c.Code == CmdTestFragmentSize.Code ||
		c.Code == CmdTestUpstreamEncoder.Code

}

// ExpectsEmptyReply will return true if the command expects empty reply (no data
func (c Command) ExpectsEmptyReply() bool {
	return c.Code == CmdVersion.Code || c.Code == CmdTestDownstreamEncoder.Code || c.Code == CmdTestMultiQuery.Code
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
func (c Command) ValidateType(data string) error {
	if !c.IsOfType(data) {
		return errors.Errorf("Invalid command type. Expected %v, got, %v", c, data[0])
	}
	return nil
}

// IsOfType will check if the supplied string starts with the given command type
func (c Command) IsOfType(data string) bool {
	if len(data) < 0 {
		return false
	}
	if data[0] == uint8(c.Code) {
		return true
	}
	if strings.ToLower(data[0:1])[0] == uint8(c.Code) {
		return true
	}
	return false
}
