package args

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

type YamlParser struct {
	ParseAsDefaults bool // override default flags
	parser          *flags.Parser
}

// NewYamlParser creates a new yaml parser for a given Parser.
func NewYamlParser(p *flags.Parser) *YamlParser {
	return &YamlParser{
		parser: p,
	}
}

// ParseFile parses flags from an ini formatted file. See Parse for more
// information on the ini file format. The returned errors can be of the type
// flags.Error or flags.IniError.
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

	return y.parse(body, yaml.ReferenceDirs(path.Dir(filename)), yaml.RecursiveDir(true))
}

func (y *YamlParser) parse(config io.Reader, opts ...yaml.DecodeOption) error {
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

func (y *YamlParser) parseSegment(obj map[string]interface{}) error {
	for name, val := range obj {
		command := y.parser.Find(name)
		if command == nil {
			return errors.WithStack(&flags.Error{
				Type:    flags.ErrUnknownGroup,
				Message: fmt.Sprintf("could not find option command '%s'", name),
			})
		}

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

