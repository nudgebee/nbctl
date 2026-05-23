package cmd

import (
	"encoding/json"
	"testing"
)

func TestCountJSONArray(t *testing.T) {
	cases := []struct {
		name string
		in   json.RawMessage
		want int
	}{
		{"empty", nil, 0},
		{"native array", json.RawMessage(`[1,2,3]`), 3},
		{"native empty array", json.RawMessage(`[]`), 0},
		{"double-encoded string", json.RawMessage(`"[{\"a\":1},{\"a\":2}]"`), 2},
		{"double-encoded empty", json.RawMessage(`"[]"`), 0},
		{"malformed", json.RawMessage(`{not json`), 0},
		{"object (not array)", json.RawMessage(`{"a":1}`), 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := countJSONArray(c.in); got != c.want {
				t.Errorf("countJSONArray(%q) = %d, want %d", string(c.in), got, c.want)
			}
		})
	}
}

func TestJoinJSONNames(t *testing.T) {
	cases := []struct {
		name string
		in   json.RawMessage
		key  string
		want string
	}{
		{"empty", nil, "role", ""},
		{"native array of objects", json.RawMessage(`[{"role":"admin"},{"role":"viewer"}]`), "role", "admin,viewer"},
		{"double-encoded array", json.RawMessage(`"[{\"name\":\"group1\"},{\"name\":\"group2\"}]"`), "name", "group1,group2"},
		{"missing key skipped", json.RawMessage(`[{"role":"admin"},{"other":"x"}]`), "role", "admin"},
		{"non-string value skipped", json.RawMessage(`[{"role":42},{"role":"admin"}]`), "role", "admin"},
		{"empty string skipped", json.RawMessage(`[{"role":""},{"role":"admin"}]`), "role", "admin"},
		{"malformed", json.RawMessage(`{not json`), "role", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := joinJSONNames(c.in, c.key); got != c.want {
				t.Errorf("joinJSONNames(%q, %q) = %q, want %q", string(c.in), c.key, got, c.want)
			}
		})
	}
}

func TestPeelJSONString(t *testing.T) {
	cases := []struct {
		name string
		in   json.RawMessage
		want string
	}{
		{"empty", nil, ""},
		{"native object", json.RawMessage(`{"a":1}`), `{"a":1}`},
		{"native array", json.RawMessage(`[1,2,3]`), `[1,2,3]`},
		{"double-encoded object", json.RawMessage(`"{\"a\":1}"`), `{"a":1}`},
		{"double-encoded array", json.RawMessage(`"[1,2,3]"`), `[1,2,3]`},
		{"plain string passed through", json.RawMessage(`"not json"`), `not json`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := string(peelJSONString(c.in)); got != c.want {
				t.Errorf("peelJSONString(%q) = %q, want %q", string(c.in), got, c.want)
			}
		})
	}
}
