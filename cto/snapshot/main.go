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
	maxReplyBytes = 16 << 10

	runTimeout = 100 * time.Second
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr, time.Now()))
}

func defaultLocalFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "cto.md"
	}
	return filepath.Join(home, ".agents", "local", "cto.md")
}

func run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, now time.Time) int {
	evaluate := len(args) > 0 && args[0] == "evaluate"
	reply := len(args) > 0 && args[0] == "reply"
	if reply || evaluate {
		args = args[1:]
	}
	fs := flag.NewFlagSet("cto-snapshot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	casesFile := fs.String("cases", "", "JSON evaluation cases containing uuid and gate")
	localFile := fs.String("local-file", defaultLocalFile(), "path of the CTO local file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
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
	if evaluate {
		doc, err = evaluateTodos(ctx, cfg, gate, now, *casesFile)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 2
		}
	} else if reply {
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

func judgeReply(ctx context.Context, gate *Gate, stdin io.Reader) Judgment {
	data, err := io.ReadAll(stdin)
	data = bytes.TrimSpace(data)
	if len(data) > maxReplyBytes {
		data = data[len(data)-maxReplyBytes:]
	}
	text := strings.ToValidUTF8(string(data), "")
	if err != nil || text == "" {
		return Judgment{Subject: "reply", Gate: gateReply.Name, Verdict: verdictEscalate, DecidedBy: decidedByError,
			Reason: "there is no reply text to judge"}
	}
	return gate.Judge(ctx, "reply", "", gateReply, map[string]string{"reply": text})
}
