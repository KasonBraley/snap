# Snap

[![Go Reference](https://pkg.go.dev/badge/github.com/KasonBraley/snap.svg)](https://pkg.go.dev/github.com/KasonBraley/snap)

Minimalistic snapshot testing for Go.

Similar to the concept of golden files, but instead of a separate file that contains the snapshot,
the snapshot is directly in the source code tests.

Highlights:

- Simple, minimal API.
- Provides automatic updating of the shapshot in code. Can trigger via environment variable `SNAP_UPDATE=1`
  to update all snapshots at once, or can update one test at a time using the `Update` method.
- Leverages the powerful [go-cmp](https://github.com/google/go-cmp) package for displaying [rich diffs](#usage)
  when the snapshot differs from what is expected.
- Ability to ignore part of the input text by using a special `<snap:ignore>` marker.

Limitations:

- When updating a snapshot that uses the `<snap:ignore>` marker, the marker is overwritten. This can be
  worked around by undoing that specific line back to the ignore marker(I do this easily with Git hunks),
  but it is indeed a little annoying to deal with.
- Updating the snapshot does not currently work if the `snap.Snap` function is assigned to a different variable.
  Such as `check := snap.Snap`.

Inspired by:

- https://tigerbeetle.com/blog/2024-05-14-snapshot-testing-for-the-masses
- https://ianthehenry.com/posts/my-kind-of-repl/
- https://speakerdeck.com/mitchellh/advanced-testing-with-go?slide=19
- https://blog.janestreet.com/using-ascii-waveforms-to-test-hardware-designs/

### Usage

```go
func TestExample(t *testing.T) {
    checkAddition := func(x int, y int, want *snap.Snapshot) {
        got := x + y
        want.Diff(strconv.Itoa(got))
    }

    checkAddition(2, 2, snap.Snap(t, "8")) // should be 4
}
```

Running that test will fail, and prints the diff between the actual result (`4`) from the `checkAddtion`
function, and what is specified in the snapshot:

```bash
=== RUN   TestExample
    snap_test.go:149: snap: Snapshot differs: (-want +got):
          string(
        -       "8",
        +       "4",
          )
    snap_test.go:149: snap: Rerun with SNAP_UPDATE=1 environmental variable to update the snapshot.
--- FAIL: TestExample (0.00s)
```

To update that snapshot automatically without manually editing the code, rerun the test with `SNAP_UPDATE=1`
and it will change `snap.Snap("8")` to `snap.Snap("4")` for you.

This is a small example, this testing strategy really speeds things up when you have large outputs
that need changing, such as large JSON blobs or any substantial amount of text.

#### Ignoring data

Sometimes you have data in tests that change on each run. Such as timestamps, or random value.
These values can be ignored using the special marker `<snap:ignore>`.

This example shows how to ignore a JSON field. The `timestamp` field in the `person` struct will be ignored when diffing the
expected and got data.

```go
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
```

#### Import alias

Snapshot updating still works if you decide to import this package under a different alias, such as:

```go
import (
    "strconv"
    "testing"
    foo "github.com/KasonBraley/snap"
)

func TestExample(t *testing.T) {
    checkAddition := func(x int, y int, want *snap.Snapshot) {
        got := x + y
        want.Diff(strconv.Itoa(got))
    }

    checkAddition(2, 2, foo.Snap(t, "8")) // "foo" instead of "snap" still works when using SNAP_UPDATE=1
}
```

### Examples

The [./examples](./examples) directory showcases some more elaborate use cases for this package, such
as testing a CLI application.

The [tests](./snap_test.go) for this package might also serve as a good reference.
