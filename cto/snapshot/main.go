// Command cto-snapshot gathers the facts a CTO sweep needs, lets rules and
// gates judge them, and prints one JSON document. It only reads: it never
// writes to herdr, Taskwarrior, git, GitHub, a transcript or a vault.
//
//	cto-snapshot [--local-file PATH]         print the snapshot
//	cto-snapshot reply [--local-file PATH]   judge an orchestrator reply on standard input
package main

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// maxReplyBytes bounds the orchestrator reply the reply gate reads. The
	// last bytes are kept, because a reply ends with its verdict.
	maxReplyBytes = 16 << 10
	// runTimeout bounds one whole run, so a hung source or gate cannot stall a
	// sweep. What it cuts off is reported as unreadable or escalated.
	runTimeout = 5 * time.Minute
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr, time.Now()))
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
// returns the process exit code: 0 once it prints a document, whatever it could
// not read, and 2 for a bad invocation or a bad local file.
func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, now time.Time) int {
	reply := len(args) > 0 && args[0] == "reply"
	if reply {
		args = args[1:]
	}
	fs := flag.NewFlagSet("cto-snapshot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	localFile := fs.String("local-file", defaultLocalFile(), "path of the CTO local file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	// The key leaves the environment before any source command can start.
	key, keyErr := readAPIKey()
	cfg, err := loadConfig(*localFile)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	logger, logFile, err := openGateLog(cfg.Gate.Log)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	defer func() { _ = logFile.Close() }()
	gate := newGate(cfg.Gate, cfg.Thresholds, key, keyErr, logger, now)

	var doc any
	if reply {
		doc = judgeReply(ctx, gate, stdin)
	} else {
		doc = collect(ctx, cfg, gate, now)
	}
	if err := json.MarshalWrite(stdout, doc, json.Deterministic(true), jsontext.WithIndent("  ")); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	_, _ = fmt.Fprintln(stdout)
	return 0
}

// judgeReply asks the orchestrator_reply gate what an orchestrator's chat reply
// says about its tabs. A reply that is empty after trimming escalates without a
// call; a long reply is cut to its last maxReplyBytes.
func judgeReply(ctx context.Context, gate *Gate, stdin io.Reader) Judgment {
	data, err := io.ReadAll(stdin)
	data = bytes.TrimSpace(data)
	if len(data) > maxReplyBytes {
		data = data[len(data)-maxReplyBytes:]
	}
	text := strings.ToValidUTF8(string(data), "") // drops a character the cut split
	if err != nil || text == "" {
		return Judgment{Subject: "reply", Gate: gateReply.Name, Verdict: verdictEscalate, DecidedBy: decidedByError,
			Reason: "there is no reply text to judge"}
	}
	return gate.Judge(ctx, "reply", "", gateReply, map[string]string{"reply": text})
}
