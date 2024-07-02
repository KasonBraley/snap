package snap_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/KasonBraley/snap"
)

func TestSnapDiff(t *testing.T) {
	checkAddition := func(x int, y int, want *snap.Snapshot) {
		got := x + y
		want.Diff(strconv.Itoa(got))
	}

	checkAddition(2, 2, snap.Snap(t, "4"))
}

func TestSnapInlineIgnore(t *testing.T) {
	check := func(want *snap.Snapshot) {
		want.Diff(fmt.Sprintf("the current Unix ms time is %d ms", time.Now().UnixMilli()))
	}

	check(snap.Snap(t, "the current Unix ms time is <snap:ignore> ms"))
}

func TestSnapJSON(t *testing.T) {
	checkJSON := func(want *snap.Snapshot) {
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
		snap.Snap(t, `{
  "name": "Doug",
  "age": 20
}`))
}

func TestSnapJSONWithIgnore(t *testing.T) {
	checkJSON := func(want *snap.Snapshot) {
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
		snap.Snap(t, `{
  "name": "Doug",
  "age": 20,
  "timestamp": "<snap:ignore>"
}`))
}
