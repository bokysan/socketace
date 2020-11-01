package flags

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"github.com/jessevdk/go-flags"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"reflect"
	"unsafe"
)

// YamlParser is an argument parser for flags package but takes a YAML file instead of a standard INI.
type YamlParser struct {
	ParseAsDefaults bool // override default flags
	parser          *flags.Parser
}

// NewYamlParser creates a new yaml parser for a given flags.Parser.
func NewYamlParser(p *flags.Parser) *YamlParser {
	return &YamlParser{
		parser: p,
	}
}

// ParseFile parses flags from an yaml formatted file. The returned errors
// can be of the type flags.Error or flags.IniError.
func (y *YamlParser) ParseFile(filename string) error {
	body, err := os.Open(filename)

	if err != nil {
		return errors.WithStack(err)
	}

	defer func() {
		if err := body.Close(); err != nil {
			log.Errorf("Could not close %s: %v", filename, err)
		}
	}()

	// Parse the file. QueryDns into the decoder the location of the file and the support for recursive
	// directories. This allows you to reference files in subdirs
	return y.parse(body, yaml.ReferenceDirs(path.Dir(filename)), yaml.RecursiveDir(true))
}

// parse is an internal function which takes an input stream (a reader) and parses YAML segments
// one after another, using the provided decode options. This allows you to have multiple individual
// YAML segments within one physical file / input stream, all separated by triple dashes (`---`).
func (y *YamlParser) parse(config io.Reader, opts ...yaml.DecodeOption) error {

	// Create a new decoder
	decoder := yaml.NewDecoder(config, opts...)

	i := 0
	for true {
		i++

		obj := make(map[string]interface{})
		err := decoder.Decode(&obj)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return errors.Wrapf(err, "Could not decode element at position %v", i)
		}

		if err = y.parseSegment(obj); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

// parseSegment will get the "segment" from our input stream and try to match it key = name to our parameter
// groups. E.g. -- top level yaml line "generic:" will be matched to a group named "generic". This is the only
// way the parser works. If you don't have any groups, the parser will fail.
func (y *YamlParser) parseSegment(obj map[string]interface{}) error {
	for name, val := range obj {

		// Find the "group" / command this key belongs to
		command := y.parser.Find(name)
		if command == nil {
			return errors.WithStack(&flags.Error{
				Type:    flags.ErrUnknownGroup,
				Message: fmt.Sprintf("could not find option command '%s'", name),
			})
		}

		// We need to complicate things a bit here, as the flags library does not allow direct access to
		// the underlying data structure. It's not really a nice way to do it, but currently there's no
		// other way to implement this.
		group := reflect.ValueOf(command.Group)
		dereferencedGroup := reflect.Indirect(group)
		dataField := dereferencedGroup.FieldByName("data")
		dataField = reflect.NewAt(dataField.Type(), unsafe.Pointer(dataField.UnsafeAddr())).Elem()
		dataFieldPtr := dataField.Elem() // ptr / *Host

		if conv, err := yaml.Marshal(val); err != nil {
			return errors.WithStack(err)
		} else if err := yaml.Unmarshal(conv, dataFieldPtr.Interface()); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
