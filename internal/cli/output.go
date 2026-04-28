package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

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

func GenerateOutput(writer io.Writer, result *CommandResult) {
	if writer == nil {
		writer = os.Stdout
	}

	if result != nil && result.Data != nil {
		data := result.Data
		if result.Filter != "" {
			f, err := filter.Parse(result.Filter)
			if err != nil {
				Fatalf("Unable to parse specified output filter '%s': %s", result.Filter, err.Error())
			}
			// Normalize filter field names to match struct fields (case-insensitive)
			f = f.Normalize(result.Data)
			data, err = f.Process(data)
			if err != nil {
				// If the error appears to be about a missing field, try to provide
				// a helpful hint derived from the table format header fields.
				errStr := err.Error()
				if strings.Contains(errStr, "Failed to find field") || strings.Contains(errStr, "did not resolve to a valid field") {
					// Attempt to extract header fields from the format
					if headerFields, hErr := format.Format(result.Format).HeaderFields(result.NameLimit); hErr == nil && len(headerFields) > 0 {
						Fatalf("Invalid output-filter: %s. Available fields: %s", errStr, strings.Join(headerFields, ", "))
					}
				}
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
			if err := tableFormat.Execute(writer, true, result.NameLimit, data); err != nil {
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
			if _, err = fmt.Fprintf(writer, "%s", asJson); err != nil {
				Fatalf("Unexpected error while writing JSON output: %s", err.Error())
			}
		} else if result.OutputAs == OUTPUT_YAML {
			asYaml, err := yaml.Marshal(&data)
			if err != nil {
				Fatalf("Unexpected error while processing command results to YAML: %s", err.Error())
			}
			if _, err = fmt.Fprintf(writer, "%s", asYaml); err != nil {
				Fatalf("Unexpected error while writing YAML output: %s", err.Error())
			}
		}
	}
}
