package format

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
)

type Format struct {
	format string
	writer io.Writer
}

// TableField defines the mapping between a table header and a data field.
// This allows for custom headers and ordering of columns in the table output.
// The Field should match the name of the field in the struct.
type TableField struct {
	Header string
	Field  string
}

// TabularData is a wrapper for data that should be printed as a table.
// It allows for specifying the fields to be included in the table, along with their headers and order.
type TabularData struct {
	Data   any
	Fields []TableField
}

func (f *Format) Set(format string) {
	f.format = format
}

func (f *Format) SetOutput(writer io.Writer) {
	f.writer = writer
}

func (f *Format) Get() string {
	return f.format
}

func (f *Format) Print(obj any) {
	switch f.format {
	case "json":
		f.printJSON(obj)
	case "text":
		f.printText(obj)
	default:
		// Do nothing
	}
}

func (f *Format) printJSON(obj any) {
	// Marshal the object into a JSON string.
	// We use MarshalIndent for pretty-printing.
	b, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling to JSON: %v\n", err)
		os.Exit(1)
	}
	// Print the JSON string to the writer.
	_, _ = fmt.Fprintln(f.writer, string(b))
}

func (f *Format) printText(obj any) {
	if tabularData, ok := obj.(TabularData); ok {
		f.printTabularData(tabularData)
	} else {
		// Fallback to reflection-based table for slices
		f.printReflectedTable(obj)
	}
}

func (f *Format) printTabularData(tabularData TabularData) {
	val := reflect.ValueOf(tabularData.Data)
	if val.Kind() != reflect.Slice {
		_, _ = fmt.Fprintln(f.writer, tabularData.Data)
		return
	}

	if val.Len() == 0 {
		return
	}

	w := tabwriter.NewWriter(f.writer, 0, 0, 3, ' ', 0)

	headers := make([]string, len(tabularData.Fields))
	for i, field := range tabularData.Fields {
		headers[i] = field.Header
	}
	_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))

	elemType := val.Type().Elem()
	fieldIndices := make([][]int, len(tabularData.Fields))
	for i, field := range tabularData.Fields {
		if sf, ok := elemType.FieldByName(field.Field); ok {
			fieldIndices[i] = sf.Index
		}
	}

	jsonRawMessageType := reflect.TypeOf(json.RawMessage{})

	for i := 0; i < val.Len(); i++ {
		row := make([]string, len(tabularData.Fields))
		item := val.Index(i)
		for j := range tabularData.Fields {
			var fieldVal reflect.Value
			if fieldIndices[j] != nil {
				fieldVal = item.FieldByIndex(fieldIndices[j])
			}

			if fieldVal.IsValid() {
				if fieldVal.Type() == jsonRawMessageType {
					var recommendationContent struct {
						VulnerabilityID string `json:"VulnerabilityID"`
						PrimaryURL      string `json:"PrimaryURL"`
					}
					rawJSON := fieldVal.Interface().(json.RawMessage)
					// Unquote the raw JSON string if it's a string literal
					unquotedJSON, err := strconv.Unquote(string(rawJSON))
					if err != nil {
						// If unquoting fails, assume it's already a direct JSON object
						unquotedJSON = string(rawJSON)
					}

					if err := json.Unmarshal([]byte(unquotedJSON), &recommendationContent); err == nil {
						if recommendationContent.PrimaryURL != "" {
							row[j] = fmt.Sprintf("%s (%s)", recommendationContent.VulnerabilityID, recommendationContent.PrimaryURL)
						} else {
							row[j] = recommendationContent.VulnerabilityID
						}
					} else {
						row[j] = "Error parsing CVE" // Fallback in case of parsing error
					}
				} else {
					row[j] = fmt.Sprintf("%v", fieldVal.Interface())
				}
			} else {
				row[j] = ""
			}
		}
		_, _ = fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	_ = w.Flush()
}

func (f *Format) printReflectedTable(obj any) {
	val := reflect.ValueOf(obj)

	switch val.Kind() {
	case reflect.Slice:
		f.printSlice(val)
	case reflect.Struct:
		f.printStruct(val)
	default:
		_, _ = fmt.Fprintln(f.writer, obj)
	}
}

func (f *Format) printSlice(val reflect.Value) {
	if val.Len() == 0 {
		return
	}

	elemType := val.Type().Elem()
	if elemType.Kind() != reflect.Struct {
		_, _ = fmt.Fprintln(f.writer, val.Interface())
		return
	}

	w := tabwriter.NewWriter(f.writer, 0, 0, 3, ' ', 0)

	headers := make([]string, elemType.NumField())
	for i := range headers {
		headers[i] = elemType.Field(i).Name
	}
	_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))

	for i := 0; i < val.Len(); i++ {
		row := make([]string, elemType.NumField())
		for j := range row {
			row[j] = fmt.Sprintf("%v", val.Index(i).Field(j).Interface())
		}
		_, _ = fmt.Fprintln(w, strings.Join(row, "\t"))
	}

	_ = w.Flush()
}

func (f *Format) printStruct(val reflect.Value) {
	w := tabwriter.NewWriter(f.writer, 0, 0, 3, ' ', 0)

	typ := val.Type()
	var evidenceField reflect.Value
	var evidenceFieldName string

	for i := 0; i < val.NumField(); i++ {
		fieldName := typ.Field(i).Name
		fieldValue := val.Field(i)

		if fieldName == "Evidences" {
			evidenceField = fieldValue
			evidenceFieldName = fieldName
			continue
		}

		if fieldValue.Type() == reflect.TypeOf(json.RawMessage{}) {
			var data map[string]any
			if err := json.Unmarshal(fieldValue.Interface().(json.RawMessage), &data); err == nil {
				_, _ = fmt.Fprintf(w, "%s:\n", fieldName)
				for k, v := range data {
					_, _ = fmt.Fprintf(w, "\t%s\t%v\n", k, v)
				}
			} else {
				_, _ = fmt.Fprintf(w, "%s\t%s\n", fieldName, string(fieldValue.Interface().(json.RawMessage)))
			}
		} else if fieldValue.Kind() == reflect.Slice && fieldValue.Type().Elem().Kind() == reflect.Struct {
			_, _ = fmt.Fprintf(w, "\n%s:\n", fieldName)
			f.printSlice(fieldValue)
		} else if fieldValue.Kind() == reflect.Struct {
			_, _ = fmt.Fprintf(w, "\n%s:\n", fieldName)
			nestedTyp := fieldValue.Type()
			for j := 0; j < fieldValue.NumField(); j++ {
				nestedFieldName := nestedTyp.Field(j).Name
				nestedFieldValue := fieldValue.Field(j)
				_, _ = fmt.Fprintf(w, "\t%s\t%v\n", nestedFieldName, nestedFieldValue.Interface())
			}
		} else {
			_, _ = fmt.Fprintf(w, "%s\t%v\n", fieldName, fieldValue.Interface())
		}
	}

	if evidenceField.IsValid() {
		_, _ = fmt.Fprintf(f.writer, "\n%s:\n", evidenceFieldName)
		f.printEvidences(evidenceFieldName, evidenceField.Interface().(json.RawMessage))
	}

	_ = w.Flush()
}

func (f *Format) printEvidences(fieldName string, evidences json.RawMessage) {
	var evidenceData []struct {
		Data           json.RawMessage `json:"data"`
		Type           string          `json:"type"`
		AdditionalInfo struct {
			Title string `json:"title"`
		} `json:"additional_info"`
	}

	if err := json.Unmarshal(evidences, &evidenceData); err != nil {
		// if unmarshal fails, print as string
		_, _ = fmt.Fprintf(f.writer, "%s\t%s\n", fieldName, string(evidences))
		return
	}

	bold := lipgloss.NewStyle().Bold(true)

	for _, ev := range evidenceData {
		_, _ = fmt.Fprintf(f.writer, "\n%s:\n", bold.Render(ev.AdditionalInfo.Title))
		switch ev.Type {
		case "table":
			var tableData struct {
				Headers []string `json:"headers"`
				Rows    [][]any  `json:"rows"`
			}
			if err := json.Unmarshal(ev.Data, &tableData); err == nil {
				// print table
				_, _ = fmt.Fprintln(f.writer, strings.Join(tableData.Headers, "\t"))
				for _, row := range tableData.Rows {
					var rowStr []string
					for _, cell := range row {
						rowStr = append(rowStr, fmt.Sprintf("%v", cell))
					}
					_, _ = fmt.Fprintln(f.writer, strings.Join(rowStr, "\t"))
				}
			}
		case "json":
			var jsonData map[string]any
			if err := json.Unmarshal(ev.Data, &jsonData); err == nil {
				for k, v := range jsonData {
					_, _ = fmt.Fprintf(f.writer, "\t%s\t%v\n", k, v)
				}
			}
		case "gz":
			_, _ = fmt.Fprintf(f.writer, "\t[gzipped data]\n")
		default:
			_, _ = fmt.Fprintf(f.writer, "\t%s\n", string(ev.Data))
		}
	}
}

var formatHandler = &Format{writer: os.Stdout}

func GetFormat() *Format {
	return formatHandler
}
