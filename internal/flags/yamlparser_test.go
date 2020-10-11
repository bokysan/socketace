package flags

import (
	"github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/require"
	"testing"
)

var General struct {
	Experimental bool   `long:"experimental" description:"Enable experimental features"`
	File         string `long:"file"`
}

func Test_EmptyParse(t *testing.T) {
	file := "testdata/empty.yml"

	parser := flags.NewNamedParser("yaml-test", flags.HelpFlag|flags.PrintErrors)
	yamlParser := NewYamlParser(parser)
	err := yamlParser.ParseFile(file)

	require.NoErrorf(t, err, "Parsing not successful: %v", file)
}

func Test_GeneralParse(t *testing.T) {
	file := "testdata/general.yml"

	parser := flags.NewNamedParser("yaml-test", flags.HelpFlag|flags.PrintErrors)
	yamlParser := NewYamlParser(parser)

	data := &General
	_, err := parser.AddCommand("general", "General", "General options", data)
	require.NoErrorf(t, err, "Could not add general group")

	err = yamlParser.ParseFile(file)
	require.NoErrorf(t, err, "Parsing not successful: %v", file)

	require.Equal(t, true, data.Experimental, "Invalid reading of boolean value")
	require.Equal(t, "something.txt", data.File, "Invalid reading of string value")

}

func Test_InvalidGeneralParse(t *testing.T) {
	file := "testdata/invalid_general.yml"

	parser := flags.NewNamedParser("yaml-test", flags.HelpFlag|flags.PrintErrors)
	yamlParser := NewYamlParser(parser)

	_, err := parser.AddCommand("general", "General", "General options", &General)
	require.NoErrorf(t, err, "Could not add general group")

	err = yamlParser.ParseFile(file)
	require.NoErrorf(t, err, "Parsing not successful: %v", file)
}

func Test_InvalidNoCommand(t *testing.T) {
	file := "testdata/invalid_no_command.yml"

	parser := flags.NewNamedParser("yaml-test", flags.HelpFlag|flags.PrintErrors)
	yamlParser := NewYamlParser(parser)

	_, err := parser.AddCommand("general", "General", "General options", &General)
	require.NoErrorf(t, err, "Could not add general group")

	err = yamlParser.ParseFile(file)
	require.Errorf(t, err, "Parsing not successful, expected error but did not get one: %v", file)
}
