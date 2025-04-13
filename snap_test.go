package snap

import (
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestSnapDiff(t *testing.T) {
	checkAddition := func(x int, y int, want *Snapshot) {
		got := x + y
		want.Diff(strconv.Itoa(got))
	}

	checkAddition(2, 2, Snap(t, "4"))
}

func TestSnapInlineIgnore(t *testing.T) {
	check := func(want *Snapshot) {
		want.Diff(fmt.Sprintf("the current Unix ms time is %d ms", time.Now().UnixMilli()))
	}

	check(Snap(t, "the current Unix ms time is <snap:ignore> ms"))
}

func TestSnapJSON(t *testing.T) {
	checkJSON := func(want *Snapshot) {
		type person struct {
			Name         string `json:"name"`
			Age          uint   `json:"age"`
			ignoredField string
		}

		p := person{
			Name:         "Doug",
			Age:          20,
			ignoredField: "bar",
		}

		want.DiffJSON(&p, "  ")
	}

	checkJSON(
		Snap(t, `{
  "name": "Doug",
  "age": 20
}`))
}

func TestSnapJSONWithIgnore(t *testing.T) {
	checkJSON := func(want *Snapshot) {
		type person struct {
			Name string    `json:"name"`
			Age  uint      `json:"age"`
			Time time.Time `json:"timestamp"`
		}

		p := person{
			Name: "Doug",
			Age:  20,
			Time: time.Now(),
		}

		want.DiffJSON(&p, "  ")
	}

	checkJSON(
		Snap(t, `{
  "name": "Doug",
  "age": 20,
  "timestamp": "<snap:ignore>"
}`))
}

func TestEqualExcludingIgnored(t *testing.T) {
	casesOk := []struct {
		got, snapshot string
	}{
		{got: "123", snapshot: "123"},
		{got: "1234", snapshot: "1<snap:ignore>4"},
		{got: "12345678", snapshot: "12<snap:ignore>56<snap:ignore>8"},
		{got: `{
  "id": "1",
  "timestamp": "timestamp_value"
}`, snapshot: `{
  "id": "1",
  "timestamp": "<snap:ignore>"
}`},
	}

	for _, tc := range casesOk {
		t.Run("", func(t *testing.T) {
			result := equalExcludingIgnored(tc.got, tc.snapshot)
			if !result {
				t.Errorf("expected true, got false for got: %q, snapshot: %q", tc.got, tc.snapshot)
			}
		})
	}

	casesErr := []struct {
		got, snapshot string
	}{
		{got: "123", snapshot: "132"},
		{got: "1234", snapshot: "1<snap:ignore>5"},
		{got: "12345678", snapshot: "12<snap:ignore>43<snap:ignore>78"},
		{got: "12345678", snapshot: "12<snap:ignore>34<snap:ignore>87"},
		{got: "123", snapshot: "12<snap:ignore>3"},
		{got: "1\n2\n3", snapshot: "1<snap:ignore>3"},
	}

	for _, tc := range casesErr {
		t.Run("", func(t *testing.T) {
			result := equalExcludingIgnored(tc.got, tc.snapshot)
			if result {
				t.Errorf("expected false, got true for got: %q, snapshot: %q", tc.got, tc.snapshot)
			}
		})
	}
}

func TestPreserveIgnoreMarkers(t *testing.T) {
	tests := []struct {
		name     string
		original string
		newJSON  string
		expected string
	}{
		{
			name:     "Doesn't work on same line",
			original: `{"name":"John","age":"<snap:ignore>"}`,
			newJSON:  `{"name":"Jane","age":25}`,
			expected: `{"name":<snap:ignore>`,
		},
		{
			name: "Preserve ignore markers in multi-line JSON",
			original: `{
  "name": "John",
  "age": "<snap:ignore>",
  "address": {
    "street": "<snap:ignore>",
    "city": "New York"
  }
}`,
			newJSON: `{
  "name": "Jane",
  "age": 25,
  "address": {
    "street": "123 Main St",
    "city": "Boston"
  }
}`,
			expected: `{
  "name": "Jane",
  "age": "<snap:ignore>",
  "address": {
    "street": "<snap:ignore>",
    "city": "Boston"
  }
}`,
		},
		{
			name: "Preserve ignore marker at end of object",
			original: `{
  "name": "John",
  "timestamp": "<snap:ignore>"
}`,
			newJSON: `{
  "name": "Jane",
  "timestamp": "2023-05-15T12:34:56Z"
}`,
			expected: `{
  "name": "Jane",
  "timestamp": "<snap:ignore>"
}`,
		},
		{
			name: "Preserve ignore marker in middle of object",
			original: `{
  "name": "John",
  "timestamp": "<snap:ignore>",
  "age": 30
}`,
			newJSON: `{
  "name": "Jane",
  "timestamp": "2023-05-15T12:34:56Z",
  "age": 25
}`,
			expected: `{
  "name": "Jane",
  "timestamp": "<snap:ignore>",
  "age": 25
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := preserveIgnoreMarkers(tt.newJSON, tt.original)
			if result != tt.expected {
				t.Errorf("preserveIgnoreMarkers() =\n%v\nwant\n%v", result, tt.expected)
			}
		})
	}
}
