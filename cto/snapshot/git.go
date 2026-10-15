package main

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var originSlug = regexp.MustCompile(`github\.com[:/]([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+?)(?:\.git)?/?$`)

type gitDetails struct {
	Slug          string
	Branch        string
	DefaultBranch string
	Dirty         int
	Ahead         int
	Behind        int
}

func (r runner) gitRoot(ctx context.Context, cwd string) (root string, isRepo bool, err error) {
	if _, statErr := os.Stat(cwd); statErr != nil {
		return "", false, nil
	}
	out, _, err := r.output(ctx, cwd, r.cmds.Git, "rev-parse", "--path-format=absolute", "--show-toplevel", "--git-common-dir")
	if err != nil {
		if strings.Contains(err.Error(), "not a git repository") {
			return "", false, nil
		}
		return "", false, err
	}
	top, common, _ := strings.Cut(strings.TrimSpace(string(out)), "\n")
	if filepath.Base(common) == ".git" {
		return filepath.Dir(common), true, nil
	}
	return top, true, nil
}

func (r runner) gitDetails(ctx context.Context, root string) (gitDetails, error) {
	var d gitDetails
	out, _, err := r.output(ctx, root, r.cmds.Git, "status", "--porcelain=v2", "--branch")
	if err != nil {
		return d, err
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			d.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.ab "):
			fields := strings.Fields(line)
			if len(fields) == 4 {
				d.Ahead, _ = strconv.Atoi(strings.TrimPrefix(fields[2], "+"))
				d.Behind, _ = strconv.Atoi(strings.TrimPrefix(fields[3], "-"))
			}
		case !strings.HasPrefix(line, "#"):
			d.Dirty++
		}
	}
	if out, _, err := r.output(ctx, root, r.cmds.Git, "remote", "get-url", "origin"); err == nil {
		if m := originSlug.FindStringSubmatch(strings.TrimSpace(string(out))); m != nil {
			d.Slug = m[1]
		}
	}
	if out, _, err := r.output(ctx, root, r.cmds.Git, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		d.DefaultBranch = strings.TrimSpace(string(out))
	}
	return d, nil
}

func (r runner) onDefault(ctx context.Context, root, defaultBranch, sha string) string {
	if defaultBranch == "" {
		return "unchecked"
	}
	_, code, _ := r.output(ctx, root, r.cmds.Git, "merge-base", "--is-ancestor", sha, defaultBranch)
	switch code {
	case 0:
		return "on_default"
	case 1:
		return "not_on_default"
	}
	return "unchecked"
}
