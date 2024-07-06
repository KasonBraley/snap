// Package snap provides a simple implementation of Snapshot testing in Go.
//
// The [Snap] function provides the ability for diffing with other strings, and can update it's
// own source code to match the expected value.
//
// Usage:
//
//	func TestAddition(t *testing.T) {
//	  checkAddition := func(x int, y int, want *snap.Snapshot) {
//	      got := x + y
//	      want.Diff(strconv.Itoa(got))
//	  }
//
//	  checkAddition(2, 2, snap.Snap(t, "8")) // should be 4
//	}
//
// Running that test will fail, printing the diff between the actual result (`4`) and what is specified
// in the source code:
//
//	    snap_test.go:34: snap: Snapshot differs: (-want +got):
//	          string(
//	        -       "8",
//	        +       "4",
//	          )
//	    snap_test.go:34: snap: Rerun with SNAP_UPDATE=1 environmental variable to update the snapshot.
//	--- FAIL: TestAddition (0.00s)
//
// Re-running the test with SNAP_UPDATE=1 environmental variable will update the
// source code in-place to say "4". Alternatively, you can use [Snapshot.Update] to auto-update
// just a single test.
//
// Snapshots can use the `<snap:ignore>` marker to ignore part of input. This is helpful when dealing
// with values that change between test runs, like timestamps:
//
//	func TestSnapTime(t *testing.T) {
//		timestampStr := fmt.Sprintf("Unix time is %d ms", time.Now().UnixMilli())
//
//		snap.Snap(t, "Unix time is <snap:ignore> ms").Diff(timestampStr)
//	}
//
// Main idea and influence came from these articles:
//
//   - https://tigerbeetle.com/blog/2024-05-14-snapshot-testing-for-the-masses
//   - https://ianthehenry.com/posts/my-kind-of-repl/
//   - https://speakerdeck.com/mitchellh/advanced-testing-with-go?slide=19
//   - https://blog.janestreet.com/using-ascii-waveforms-to-test-hardware-designs/
package snap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type sourceLocation struct {
	file string
	line int
}

type Snapshot struct {
	location            sourceLocation
	text                string
	updateThis          bool
	t                   *testing.T
	foundCallerLocation bool
}

// Creates a new Snapshot.
//
// Set SNAP_UPDATE=1 environment variable or call the [Snapshot.Update] method to automagically update
// the test value.
func Snap(t *testing.T, text string) *Snapshot {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		t.Errorf("snap: unable to retrieve caller location")
	}

	return &Snapshot{
		location:            sourceLocation{file: file, line: line},
		text:                text,
		t:                   t,
		foundCallerLocation: ok,
	}
}

// Update allows updating just this particular snapshot.
func (s *Snapshot) Update() *Snapshot {
	return &Snapshot{
		location:   sourceLocation{file: s.location.file, line: s.location.line},
		text:       s.text,
		updateThis: true,
	}
}

// Diff compares the snapshot with a given string.
// It calls [testing.T.Error] when the snapshot is not equal to the value or when an error is encountered
// elsewhere.
func (s *Snapshot) Diff(got string) {
	s.t.Helper()
	if equalExcludingIgnored(got, s.text) {
		return
	}

	if diff := cmp.Diff(s.text, got); diff != "" {
		s.t.Errorf("snap: Snapshot differs: (-want +got):\n%s", diff)
	}

	if !s.shouldUpdate() {
		s.t.Log("snap: Rerun with SNAP_UPDATE=1 environmental variable to update the snapshot.")
		return
	}

	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, s.location.file, nil, parser.ParseComments)
	if err != nil {
		s.t.Errorf("snap: %v", err)
		return
	}

	// Traverse the AST and find snap.Snap function calls.
	ast.Inspect(f, func(n ast.Node) bool {
		// Check for function call expressions.
		if callExpr, ok := n.(*ast.CallExpr); ok {
			// Check if the function being called is from a package (e.g., snap.Snap).
			if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := selExpr.X.(*ast.Ident); ok {
					if ident.Name == "snap" && selExpr.Sel.Name == "Snap" {
						if s.location.line != fset.Position(callExpr.Pos()).Line {
							return true
						}

						// Check if the __second__ argument is a string literal, the first argument
						// is for *testing.T.
						if len(callExpr.Args) > 0 {
							if strLit, ok := callExpr.Args[1].(*ast.BasicLit); ok && strLit.Kind == token.STRING {
								// TODO: handle overwriting of <snap:ignore>.
								// Check for raw string literal.
								if len(strLit.Value) >= 2 && strLit.Value[0] == '`' && strLit.Value[len(strLit.Value)-1] == '`' {
									strLit.Value = "`" + got + "`"
								} else {
									strLit.Value = `"` + got + `"`
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	// Format the modified AST to a buffer first to avoid writing garbage(or nothing at all) back
	// to the source file. Only if this succeeds, we then flush the buffer to the source file.
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		s.t.Errorf("snap: Failed to format modified AST, aborting: %s", err)
		return
	}

	outFile, err := os.OpenFile(s.location.file, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		s.t.Errorf("snap: Failed to open source file %q for writing to: %s", s.location.file, err)
		return
	}
	defer outFile.Close()

	// Write the modified(and formatted) AST in the buffer back to the original source file.
	if _, err := io.Copy(outFile, &buf); err != nil {
		s.t.Errorf("snap: Failed to write modified AST to source file %q: %s", s.location.file, err)
		return
	}

	s.t.Logf("snap: Updated %s\n", s.location.file)
}

// DiffJSON compares the snapshot with the json serialization of a value.
// It calls [testing.T.Error] when the snapshot is not equal to the value or when an error is encountered
// elsewhere.
func (s *Snapshot) DiffJSON(value any, indent string) {
	s.t.Helper()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", indent)
	if err := enc.Encode(&value); err != nil {
		s.t.Errorf("snap: %v", err)
		return
	}
	s.Diff(strings.TrimSuffix(buf.String(), "\n")) // Trim the trailing newline that *json.Encoder.Encode adds.
}

func (s *Snapshot) shouldUpdate() bool {
	if !s.foundCallerLocation {
		// If for some reason runtime.Caller failed in [Snap], don't try to update the snapshot.
		return false
	}

	if s.updateThis {
		return true
	}
	_, hasEnv := os.LookupEnv("SNAP_UPDATE")
	return hasEnv
}

func equalExcludingIgnored(got string, snapshot string) bool {
	var gotRest = got
	var snapshotRest = snapshot
	const ignoreFmt = "<snap:ignore>"

	// Don't allow ignoring suffixes and prefixes, as that makes it easy to miss trailing or leading
	// data.
	if strings.HasPrefix(snapshot, ignoreFmt) || strings.HasSuffix(snapshot, ignoreFmt) {
		panic(fmt.Sprintf("%q is not allowed as a prefix or suffix", ignoreFmt))
	}

	for {
		// First, check the snapshot for the ignore marker.
		// Cut the part before the first ignore, it should be equal between two strings...
		snapshotCutPrefix, snapshotCutSuffix, foundIgnoreInSnapshot := strings.Cut(snapshotRest, ignoreFmt)
		if !foundIgnoreInSnapshot {
			break
		}

		// Now check that `got` has the data up to the ignore marker that was cut off(the prefix).
		gotPrefix, gotSuffix, found := strings.Cut(gotRest, snapshotCutPrefix)
		if !found {
			break
		}

		// There should be nothing in this prefix if the values are indeed equal.
		if len(gotPrefix) != 0 {
			return false
		}

		gotRest = gotSuffix
		snapshotRest = snapshotCutSuffix

		// ...then find the next part that should match, and cut up to that.
		// This allows handling of multiple <snap:ignore>'s on a single line.
		nextMatchPrefix, _, nextMatchFound := strings.Cut(snapshotRest, ignoreFmt)
		if !nextMatchFound {
			nextMatchPrefix = snapshotRest
		}

		if len(nextMatchPrefix) == 0 {
			panic("nextMatchPrefix should be greater than 0")
		}

		_, snapshotRestSuffix, snapshotRestFound := strings.Cut(snapshotRest, nextMatchPrefix)
		if snapshotRestFound {
			snapshotRest = snapshotRestSuffix
		}

		gotCutNextPrefix, gotCutNextSuffix, gotCutNextFound := strings.Cut(gotRest, nextMatchPrefix)
		if !gotCutNextFound {
			return false
		}

		ignored := gotCutNextPrefix
		// If <snap:ignore> matched an empty string, or several lines, report it as an error.
		if len(ignored) == 0 || strings.Contains(ignored, "\n") {
			return false
		}

		gotRest = gotCutNextSuffix
	}

	return gotRest == snapshotRest
}
