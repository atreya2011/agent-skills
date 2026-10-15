package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const commandTimeout = 30 * time.Second

type runner struct {
	cmds Commands
}

func (r runner) output(parent context.Context, dir, name string, args ...string) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(parent, commandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.WaitDelay = 2 * time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.Bytes(), 0, nil
	}
	code := -1
	if exit, ok := errors.AsType[*exec.ExitError](err); ok {
		code = exit.ExitCode()
	}
	switch {
	case errors.Is(parent.Err(), context.DeadlineExceeded):
		err = errors.New("the run deadline passed")
	case parent.Err() != nil:
		err = errors.New("the run was cancelled")
	case ctx.Err() != nil:
		err = fmt.Errorf("timed out after %s", commandTimeout)
	}
	shown := args[:min(len(args), 3)]
	text := fmt.Sprintf("%s %s: %v", filepath.Base(name), strings.Join(shown, " "), err)
	if msg := firstLine(stderr.String()); msg != "" {
		text += ": " + msg
	}
	return stdout.Bytes(), code, errors.New(text)
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
