package main_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/KasonBraley/snap"
)

const (
	binName = "temp-testbinary"
)

func TestMain(t *testing.T) {
	check := func(t *testing.T, flag string, want *snap.Snapshot) {
		t.Helper()
		cmd, cleanup, err := launchTestProgram(flag)
		if err != nil {
			t.Error(err)
			return
		}
		t.Cleanup(cleanup)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			t.Error(err)
			return
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			t.Error(err)
			return
		}

		if err := cmd.Start(); err != nil {
			t.Error(err)
			return
		}

		stdoutData, _ := io.ReadAll(stdout)
		stderrData, _ := io.ReadAll(stderr)

		waitErr := cmd.Wait()

		var buf strings.Builder

		buf.WriteString("\n")
		if waitErr != nil {
			if exitErr, isExitError := waitErr.(*exec.ExitError); isExitError {
				buf.WriteString(fmt.Sprintf("status: %d\n", exitErr.ExitCode()))
			}
		}

		if len(stdoutData) > 0 {
			buf.WriteString(fmt.Sprintf("stdout:\n%s", string(stdoutData)))
		}

		if len(stderrData) > 0 {
			buf.WriteString(fmt.Sprintf("stderr:\n%s", string(stderrData)))
		}

		want.Diff(buf.String())
	}

	t.Run("echo", func(t *testing.T) {
		check(t, "-echo=foo", snap.Snap(t, `
stdout:
foo
`))
	})

	t.Run("help", func(t *testing.T) {
		check(t, "-help", snap.Snap(t, `
stdout:
 example-cli-program [flags]
`))
	})

	t.Run("help shorthand", func(t *testing.T) {
		check(t, "-h", snap.Snap(t, `
stdout:
 example-cli-program [flags]
`))
	})

	t.Run("bad flag", func(t *testing.T) {
		check(t, "-badflag", snap.Snap(t, `
status: 2
stderr:
flag provided but not defined: -badflag
Usage of ./temp-testbinary:
  -echo string
    	
  -h	
  -help
    	
`))
	})

}

func launchTestProgram(flag string) (cmd *exec.Cmd, cleanup func(), err error) {
	binName, err := buildBinary()
	if err != nil {
		return nil, nil, err
	}

	cmd, kill, err := newCmd(binName, flag)

	cleanup = func() {
		if kill != nil {
			kill()
		}
		os.Remove(binName)
	}

	if err != nil {
		cleanup()
		return nil, nil, err
	}

	return cmd, cleanup, nil
}

func buildBinary() (string, error) {
	build := exec.Command("go", "build", "-o", binName)

	if err := build.Run(); err != nil {
		return "", fmt.Errorf("cannot build binary %s: %s", binName, err)
	}
	return binName, nil
}

func newCmd(binName string, flag string) (cmd *exec.Cmd, kill func(), err error) {
	cmd = exec.Command("./"+binName, flag)

	kill = func() {
		_ = cmd.Process.Kill()
	}

	return
}
