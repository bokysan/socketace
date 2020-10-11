package main

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/args"
	"github.com/bokysan/socketace/v2/internal/commands/client"
	"github.com/bokysan/socketace/v2/internal/commands/server"
	"github.com/bokysan/socketace/v2/internal/commands/version"
	scFlags "github.com/bokysan/socketace/v2/internal/flags"
	"github.com/bokysan/socketace/v2/internal/util"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"os"
	"path"
)

const (
	// ErrConfigFileDoesNotExist is raised when configuration file cannot be found
	ErrConfigFileDoesNotExist = flags.ErrInvalidTag + 1
)

// SocketAce is the main executable
type SocketAce struct {
	parser *flags.Parser
}

// NewSocketAce will create a new instance of SocketAce and initialize the parser
func NewSocketAce() *SocketAce {
	executableFilename := os.Args[0]
	executablePath := path.Base(executableFilename)

	sc := &SocketAce{
		parser: flags.NewNamedParser(executablePath, flags.HelpFlag|flags.PrintErrors),
	}

	sc.setupGeneral()
	sc.setupVersion()
	sc.setupServer()
	sc.setupClient()

	return sc
}

// setupGeneral will configure general options
func (sc *SocketAce) setupGeneral() {
	if _, err := sc.parser.AddGroup("General", "General options", &args.General); err != nil {
		err = errors.WithStack(err)
		util.MustErrorNilOrExit(err)
	}
}

// setupVersion adds the `version` command
func (sc *SocketAce) setupVersion() {
	cmd := &version.Command{}
	_, err := sc.parser.AddCommand(
		"version",
		"Print the version",
		"Print the application version and exit",
		cmd,
	)
	util.MustErrorNilOrExit(err)
}

// setupServer adds the `server` command
func (sc *SocketAce) setupServer() {
	cmd := server.NewCommand()
	_, err := sc.parser.AddCommand(
		"server",
		"Run the server",
		"Run a server listening to websocket requests",
		cmd,
	)
	util.MustErrorNilOrExit(err)
}


// setupClient adds the `client` command
func (sc *SocketAce) setupClient() {
	cmd := client.NewCommand()
	_, err := sc.parser.AddCommand(
		"client",
		"Run the client",
		"Run a client connecting forwarding requests to websockets",
		cmd,
	)
	util.MustErrorNilOrExit(err)
}

// main starts socketace and reads the configuration file
func main() {

	socketAce := NewSocketAce()
	args.General.ConfigurationFile = func(file string) error {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			message := fmt.Sprintf("Configuration file %s does not exist.", file)
			util.MustErrorNilOrExit(&flags.Error{
				Type:    ErrConfigFileDoesNotExist,
				Message: message,
			})
		}

		yamlParser := scFlags.NewYamlParser(socketAce.parser)

		args.General.ConfigurationFilePath = file
		return yamlParser.ParseFile(file)
	}

	_, err := socketAce.parser.Parse()
	util.MustErrorNilOrExit(err)

}

