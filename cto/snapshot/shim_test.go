package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// shimNames are the programs the tests replace. The test binary is linked
// under each name, and when the sources run it as one of them it replays the
// recorded output instead of running tests.
var shimNames = []string{"herdr", "gh", "task"}

func TestMain(m *testing.M) {
	if name := filepath.Base(os.Args[0]); slices.Contains(shimNames, name) {
		os.Exit(replay(name, os.Args[1:]))
	}
	// A race-enabled shim would sleep one second at exit, once per command. The
	// setting only reaches the shims this process starts.
	_ = os.Setenv("GORACE", "atexit_sleep_ms=0")
	os.Exit(m.Run())
}

// fixtureKey names the recorded output for an argument list: the arguments up
// to --json joined with underscores, with slashes and colons made dashes.
func fixtureKey(args []string) string {
	for i, a := range args {
		if a == "--json" {
			args = args[:i]
			break
		}
	}
	return strings.NewReplacer("/", "-", ":", "-").Replace(strings.Join(args, "_"))
}

// replay prints <SHIM_FIXTURES>/<name>/<key>.out and exits with the code in
// <key>.code, as the recorded program did. A missing recording exits 127. When
// SHIM_ENVLOG is set it appends the environment it was started with.
func replay(name string, args []string) int {
	if path := os.Getenv("SHIM_ENVLOG"); path != "" {
		if f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
			_, _ = fmt.Fprintln(f, strings.Join(os.Environ(), "\n"))
			_ = f.Close()
		}
	}
	key := fixtureKey(args)
	base := filepath.Join(os.Getenv("SHIM_FIXTURES"), name, key)
	out, err := os.ReadFile(base + ".out")
	if err != nil {
		fmt.Fprintf(os.Stderr, "no recording for %s %s\n", name, key)
		return 127
	}
	if msg, err := os.ReadFile(base + ".err"); err == nil {
		_, _ = os.Stderr.Write(msg)
	}
	_, _ = os.Stdout.Write(out)
	if c, err := os.ReadFile(base + ".code"); err == nil {
		code, _ := strconv.Atoi(strings.TrimSpace(string(c)))
		return code
	}
	return 0
}
