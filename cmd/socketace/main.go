package main

import (
	"fmt"
	"github.com/bokysan/socketace/v2/internal/args"
	"github.com/bokysan/socketace/v2/internal/client"
	"github.com/bokysan/socketace/v2/internal/server"
	"github.com/bokysan/socketace/v2/internal/util"
	"github.com/bokysan/socketace/v2/internal/version"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	"os"
	"path"
)

const (
	ErrConfigFileDoesNotExist = flags.ErrInvalidTag + 1
)

func main() {

	parser := newParser()

	setupGeneral(parser)
	setupServer(parser)
	setupClient(parser)
	setupVersion(parser)

	args.General.ConfigurationFile = func(file string) error {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			message := fmt.Sprintf("Configuration file %s does not exist.", file)
			util.MustErrorNilOrExit(&flags.Error{
				Type:    ErrConfigFileDoesNotExist,
				Message: message,
			})
		}

		yamlParser := args.NewYamlParser(parser)
		return yamlParser.ParseFile(file)
	}

	_, err := parser.Parse()
	util.MustErrorNilOrExit(err)

}

func newParser() *flags.Parser {
	executableFilename := os.Args[0]
	executablePath := path.Base(executableFilename)
	parser := flags.NewNamedParser(executablePath, flags.HelpFlag|flags.PrintErrors)
	return parser
}

func setupGeneral(parser *flags.Parser) {
	if _, err := parser.AddGroup("General", "General options", &args.General); err != nil {
		err = errors.WithStack(err)
		util.MustErrorNilOrExit(err)
	}
}

func setupServer(parser *flags.Parser) {
	cmd := server.NewService()
	_, err := parser.AddCommand(
		"server",
		"Run the server",
		"Run a server listening to websocket requests",
		cmd,
	)
	util.MustErrorNilOrExit(err)
}

func setupClient(parser *flags.Parser) {
	cmd := client.NewService()
	_, err := parser.AddCommand(
		"client",
		"Run the client",
		"Run a client connecting forwarding requests to websockets",
		cmd,
	)
	util.MustErrorNilOrExit(err)
}

func setupVersion(parser *flags.Parser) {
	cmd := &version.Command{}
	_, err := parser.AddCommand(
		"version",
		"Print the version",
		"Print the application version and exit",
		cmd,
	)
	util.MustErrorNilOrExit(err)
}
