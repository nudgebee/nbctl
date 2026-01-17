package format

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
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

var (
	jsonRawMessageType = reflect.TypeOf(json.RawMessage{})
	stringerType       = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	errorType          = reflect.TypeOf((*error)(nil)).Elem()
	tabBytes           = []byte("\t")
	newlineBytes       = []byte("\n")
)

func (f *Format) Set(format string) {
	f.format = format
}

func (f *Format) SetOutput(writer io.Writer) {
	f.writer = writer
}

func (f *Format) GetOutput() io.Writer {
	return f.writer
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
	if td, ok := obj.(TabularData); ok {
		obj = td.Data
	}
	// Use json.Encoder to stream directly to the writer, avoiding
	// large intermediate byte slice allocations from MarshalIndent.
	enc := json.NewEncoder(f.writer)
	enc.SetIndent("", "  ")
	if err := enc.Encode(obj); err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling to JSON: %v\n", err)
		os.Exit(1)
	}
}

func (f *Format) printText(obj any) {
	if tabularData, ok := obj.(TabularData); ok {
		f.printTabularData(tabularData)
	} else {
		// Fallback to reflection-based table for slices
		f.printReflectedTable(obj)
	}
}

// writeValue optimized writer for common types to avoid fmt.Fprintf reflection overhead
func (f *Format) writeValue(w io.Writer, v reflect.Value) {
	// If the value implements fmt.Stringer or error, let fmt handle it to preserve custom formatting.
	// This check does involve some reflection overhead, but it's necessary for correctness.
	// We check if the type implements the interface.
	if v.Type().Implements(stringerType) || v.Type().Implements(errorType) {
		_, _ = fmt.Fprintf(w, "%v", v.Interface())
		return
	}

	switch v.Kind() {
	case reflect.String:
		_, _ = io.WriteString(w, v.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var buf [24]byte
		b := strconv.AppendInt(buf[:0], v.Int(), 10)
		_, _ = w.Write(b)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var buf [24]byte
		b := strconv.AppendUint(buf[:0], v.Uint(), 10)
		_, _ = w.Write(b)
	case reflect.Bool:
		if v.Bool() {
			_, _ = io.WriteString(w, "true")
		} else {
			_, _ = io.WriteString(w, "false")
		}
	case reflect.Float32:
		var buf [64]byte
		// Use 32 for bitSize to avoid precision artifacts
		b := strconv.AppendFloat(buf[:0], v.Float(), 'g', -1, 32)
		_, _ = w.Write(b)
	case reflect.Float64:
		var buf [64]byte
		b := strconv.AppendFloat(buf[:0], v.Float(), 'g', -1, 64)
		_, _ = w.Write(b)
	default:
		// Fallback for complex types
		_, _ = fmt.Fprintf(w, "%v", v.Interface())
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

	for i, field := range tabularData.Fields {
		if i > 0 {
			_, _ = w.Write(tabBytes)
		}
		_, _ = io.WriteString(w, field.Header)
	}
	_, _ = w.Write(newlineBytes)

	elemType := val.Type().Elem()
	fieldIndices := make([][]int, len(tabularData.Fields))
	isJsonRawMessage := make([]bool, len(tabularData.Fields))

	for i, field := range tabularData.Fields {
		if sf, ok := elemType.FieldByName(field.Field); ok {
			fieldIndices[i] = sf.Index
			if sf.Type == jsonRawMessageType {
				isJsonRawMessage[i] = true
			}
		}
	}

	for i := 0; i < val.Len(); i++ {
		item := val.Index(i)
		for j := range tabularData.Fields {
			if j > 0 {
				_, _ = w.Write(tabBytes)
			}
			var fieldVal reflect.Value
			if fieldIndices[j] != nil {
				fieldVal = item.FieldByIndex(fieldIndices[j])
			}

			if fieldVal.IsValid() {
				if isJsonRawMessage[j] {
					var recommendationContent struct {
						VulnerabilityID string `json:"VulnerabilityID"`
						PrimaryURL      string `json:"PrimaryURL"`
					}
					rawJSON := fieldVal.Interface().(json.RawMessage)

					var dataToUnmarshal []byte
					// Check if it's a quoted string literal (starts with quote)
					if len(rawJSON) > 0 && rawJSON[0] == '"' {
						// Use json.Unmarshal to unquote the string. This avoids the string(rawJSON) allocation
						// required by strconv.Unquote and correctly handles JSON escape sequences (like \/)
						// that strconv.Unquote rejects.
						var s string
						if err := json.Unmarshal(rawJSON, &s); err == nil {
							dataToUnmarshal = []byte(s)
						} else {
							// If unmarshalling fails, fallback to original
							dataToUnmarshal = rawJSON
						}
					} else {
						// It's already a direct JSON object/array
						dataToUnmarshal = rawJSON
					}

					if err := json.Unmarshal(dataToUnmarshal, &recommendationContent); err == nil {
						if recommendationContent.PrimaryURL != "" {
							_, _ = fmt.Fprintf(w, "%s (%s)", recommendationContent.VulnerabilityID, recommendationContent.PrimaryURL)
						} else {
							_, _ = fmt.Fprint(w, recommendationContent.VulnerabilityID)
						}
					} else {
						_, _ = io.WriteString(w, "Error parsing CVE") // Fallback in case of parsing error
					}
				} else {
					f.writeValue(w, fieldVal)
				}
			} else {
				_, _ = io.WriteString(w, "")
			}
		}
		_, _ = w.Write(newlineBytes)
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

	for i := 0; i < elemType.NumField(); i++ {
		if i > 0 {
			_, _ = w.Write(tabBytes)
		}
		_, _ = io.WriteString(w, elemType.Field(i).Name)
	}
	_, _ = w.Write(newlineBytes)

	numFields := elemType.NumField()
	for i := 0; i < val.Len(); i++ {
		// Hoist item retrieval to avoid repeated bounds checks and Value allocations
		item := val.Index(i)
		for j := 0; j < numFields; j++ {
			if j > 0 {
				_, _ = w.Write(tabBytes)
			}
			f.writeValue(w, item.Field(j))
		}
		_, _ = w.Write(newlineBytes)
	}

	_ = w.Flush()
}

func (f *Format) printStruct(val reflect.Value) {
	w := tabwriter.NewWriter(f.writer, 0, 0, 3, ' ', 0)

	typ := val.Type()
	var evidenceField reflect.Value
	var evidenceFieldName string

	for i := 0; i < val.NumField(); i++ {
		structField := typ.Field(i)
		fieldName := structField.Name
		fieldValue := val.Field(i)
		fieldType := structField.Type

		if fieldName == "Evidences" {
			evidenceField = fieldValue
			evidenceFieldName = fieldName
			continue
		}

		if fieldType == jsonRawMessageType {
			var data map[string]any
			if err := json.Unmarshal(fieldValue.Interface().(json.RawMessage), &data); err == nil {
				_, _ = fmt.Fprintf(w, "%s:\n", fieldName)
				for k, v := range data {
					_, _ = fmt.Fprintf(w, "\t%s\t%v\n", k, v)
				}
			} else {
				_, _ = fmt.Fprintf(w, "%s\t%s\n", fieldName, string(fieldValue.Interface().(json.RawMessage)))
			}
		} else if fieldType.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Struct {
			_, _ = fmt.Fprintf(w, "\n%s:\n", fieldName)
			f.printSlice(fieldValue)
		} else if fieldType.Kind() == reflect.Struct {
			_, _ = fmt.Fprintf(w, "\n%s:\n", fieldName)
			nestedTyp := fieldType
			for j := 0; j < fieldValue.NumField(); j++ {
				nestedFieldName := nestedTyp.Field(j).Name
				nestedFieldValue := fieldValue.Field(j)
				_, _ = fmt.Fprintf(w, "\t%s\t", nestedFieldName)
				f.writeValue(w, nestedFieldValue)
				_, _ = fmt.Fprintln(w)
			}
		} else {
			_, _ = fmt.Fprintf(w, "%s\t", fieldName)
			f.writeValue(w, fieldValue)
			_, _ = fmt.Fprintln(w)
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
				for i, header := range tableData.Headers {
					if i > 0 {
						_, _ = fmt.Fprint(f.writer, "\t")
					}
					_, _ = fmt.Fprint(f.writer, header)
				}
				_, _ = fmt.Fprintln(f.writer)

				for _, row := range tableData.Rows {
					for i, cell := range row {
						if i > 0 {
							_, _ = fmt.Fprint(f.writer, "\t")
						}
						_, _ = fmt.Fprint(f.writer, cell)
					}
					_, _ = fmt.Fprintln(f.writer)
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
