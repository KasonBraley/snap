package snap

import "testing"

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
