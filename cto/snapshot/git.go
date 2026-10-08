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

// originSlug extracts owner/name from a GitHub remote URL in https, ssh or
// scp form.
var originSlug = regexp.MustCompile(`github\.com[:/]([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+?)(?:\.git)?/?$`)

// gitDetails are the git facts of a project's main checkout.
type gitDetails struct {
	Slug          string
	Branch        string
	DefaultBranch string
	Dirty         int
	Ahead         int
	Behind        int
	Commits       []Commit
}

// gitRoot returns the main checkout of the repository that contains cwd, so
// that linked worktrees fold into one project. A directory that is gone or is
// not in a repository is a fact, not a read failure: isRepo is false and err
// is nil.
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

// gitDetails reads the branch, change count, default branch, origin and the
// last five commits of root. Only the status call must succeed: a repository
// without an origin, an origin HEAD or a commit simply lacks that fact.
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
	ref := "HEAD"
	if d.DefaultBranch != "" {
		ref = d.DefaultBranch
	}
	if out, _, err := r.output(ctx, root, r.cmds.Git, "log", "-n", "5", "--format=%H%x1f%cI%x1f%s", ref); err == nil {
		for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
			if parts := strings.Split(line, "\x1f"); len(parts) == 3 {
				d.Commits = append(d.Commits, Commit{Hash: parts[0], Time: parts[1], Subject: parts[2]})
			}
		}
	}
	return d, nil
}

// onDefault reports whether sha is an ancestor of the default branch:
// "on_default", "not_on_default", or "unchecked" when the repository has no
// default branch or does not know the commit.
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
