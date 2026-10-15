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

var shimNames = []string{"herdr", "gh", "task"}

func TestMain(m *testing.M) {
	if name := filepath.Base(os.Args[0]); slices.Contains(shimNames, name) {
		os.Exit(replay(name, os.Args[1:]))
	}
	_ = os.Setenv("GORACE", "atexit_sleep_ms=0")
	os.Exit(m.Run())
}

func fixtureKey(args []string) string {
	for i, a := range args {
		if a == "--json" {
			args = args[:i]
			break
		}
	}
	return strings.NewReplacer("/", "-", ":", "-").Replace(strings.Join(args, "_"))
}

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
