package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/open-edge-platform/cli/pkg/filter"
	"github.com/open-edge-platform/cli/pkg/format"
	"github.com/open-edge-platform/cli/pkg/order"
	"gopkg.in/yaml.v2"
)

type OutputType uint8

const (
	OUTPUT_TABLE OutputType = iota
	OUTPUT_JSON
	OUTPUT_YAML
)

type CommandResult struct {
	Format    format.Format
	Filter    string
	OrderBy   string
	OutputAs  OutputType
	NameLimit int
	Data      interface{}
}

func Fatalf(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
	os.Exit(1)
}

func toOutputType(in string) OutputType {
	switch in {
	case "table":
		fallthrough
	default:
		return OUTPUT_TABLE
	case "json":
		return OUTPUT_JSON
	case "yaml":
		return OUTPUT_YAML
	}
}

func GenerateOutput(result *CommandResult) {
	if result != nil && result.Data != nil {
		data := result.Data
		if result.Filter != "" {
			f, err := filter.Parse(result.Filter)
			if err != nil {
				Fatalf("Unable to parse specified output filter '%s': %s", result.Filter, err.Error())
			}
			data, err = f.Process(data)
			if err != nil {
				Fatalf("Unexpected error while filtering command results: %s", err.Error())
			}
		}
		if result.OrderBy != "" {
			s, err := order.Parse(result.OrderBy)
			if err != nil {
				Fatalf("Unable to parse specified sort specification '%s': %s", result.OrderBy, err.Error())
			}
			data, err = s.Process(data)
			if err != nil {
				Fatalf("Unexpected error while sorting command result: %s", err.Error())
			}
		}
		if result.OutputAs == OUTPUT_TABLE {
			tableFormat := format.Format(result.Format)
			if err := tableFormat.Execute(os.Stdout, true, result.NameLimit, data); err != nil {
				Fatalf("Unexpected error while attempting to format results as table : %s", err.Error())
			}
		} else if result.OutputAs == OUTPUT_JSON {
			// first try to convert it as an array of protobufs
			//asJson, err := ConvertJsonProtobufArray(data)
			//if err != nil {
			// if that fails, then just do a standard json conversion
			asJsonB, err := json.Marshal(&data)
			if err != nil {
				Fatalf("Unexpected error while processing command results to JSON: %s", err.Error())
			}
			asJson := string(asJsonB)
			//}
			fmt.Printf("%s", asJson)
		} else if result.OutputAs == OUTPUT_YAML {
			asYaml, err := yaml.Marshal(&data)
			if err != nil {
				Fatalf("Unexpected error while processing command results to YAML: %s", err.Error())
			}
			fmt.Printf("%s", asYaml)
		}
	}
}
