package format

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormat_Print(t *testing.T) {
	type testStruct struct {
		ID   int
		Name string
	}

	testData := []testStruct{
		{ID: 1, Name: "one"},
		{ID: 2, Name: "two"},
	}

	t.Run("json format", func(t *testing.T) {
		var buf bytes.Buffer
		f := GetFormat()
		f.SetOutput(&buf)
		f.Set("json")

		f.Print(testData)

		expected, _ := json.MarshalIndent(testData, "", "  ")
		assert.JSONEq(t, string(expected), buf.String())
	})

	t.Run("text format with TabularData", func(t *testing.T) {
		var buf bytes.Buffer
		f := GetFormat()
		f.SetOutput(&buf)
		f.Set("text")

		tabularData := TabularData{
			Data: testData,
			Fields: []TableField{
				{Header: "Identifier", Field: "ID"},
				{Header: "Description", Field: "Name"},
			},
		}
		f.Print(tabularData)

		expected := "Identifier   Description\n1            one\n2            two\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("text format with reflected table", func(t *testing.T) {
		var buf bytes.Buffer
		f := GetFormat()
		f.SetOutput(&buf)
		f.Set("text")

		f.Print(testData)

		expected := "ID   Name\n1    one\n2    two\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("text format with struct", func(t *testing.T) {
		var buf bytes.Buffer
		f := GetFormat()
		f.SetOutput(&buf)
		f.Set("text")

		f.Print(testStruct{ID: 1, Name: "one"})

		expected := "ID     1\nName   one\n"
		assert.Equal(t, expected, buf.String())
	})

	t.Run("text format with evidences", func(t *testing.T) {
		var buf bytes.Buffer
		f := GetFormat()
		f.SetOutput(&buf)
		f.Set("text")

		type evidenceStruct struct {
			Evidences json.RawMessage
		}

		type tableData struct {
			Headers []string   `json:"headers"`
			Rows    [][]string `json:"rows"`
		}

		table := tableData{
			Headers: []string{"h1", "h2"},
			Rows:    [][]string{{"r1c1", "r1c2"}},
		}

		tableJSON, _ := json.Marshal(table)

		evidences := []struct {
			Data           json.RawMessage `json:"data"`
			Type           string          `json:"type"`
			AdditionalInfo struct {
				Title string `json:"title"`
			} `json:"additional_info"`
		}{
			{
				Data: json.RawMessage(tableJSON),
				Type: "table",
				AdditionalInfo: struct {
					Title string `json:"title"`
				}{Title: "Test Table"},
			},
		}

		evidencesJSON, _ := json.Marshal(evidences)

		f.Print(evidenceStruct{Evidences: json.RawMessage(evidencesJSON)})

		// Just a basic check to ensure it doesn't crash and prints something
		assert.Contains(t, buf.String(), "Evidences:")
		assert.Contains(t, buf.String(), "Test Table")
		assert.Contains(t, buf.String(), "h1\th2")
		assert.Contains(t, buf.String(), "r1c1\tr1c2")
	})
}
