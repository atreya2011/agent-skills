// Command cto-snapshot gathers the facts a CTO sweep needs and prints them as
// one JSON document.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr, time.Now()))
}

// defaultLocalFile is the CTO skill's local file under the user's home.
func defaultLocalFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "cto.md"
	}
	return filepath.Join(home, ".agents", "local", "cto.md")
}

// run is the whole program with its inputs explicit so tests can call it. It
// returns the process exit code: 2 for a bad invocation or local file.
func run(_ context.Context, args []string, _, stderr io.Writer, _ time.Time) int {
	fs := flag.NewFlagSet("cto-snapshot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	localFile := fs.String("local-file", defaultLocalFile(), "path of the CTO local file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if _, err := loadConfig(*localFile); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}
