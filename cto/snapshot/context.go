package main

import (
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type contextPage struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

var wikiRef = regexp.MustCompile(`(?i)\bwiki:\s*([^\n]+)`)

var markdownEnd = regexp.MustCompile(`\.md(?:[:][0-9]+)?`)

var wikiSource = regexp.MustCompile(`[ \t]*\|[ \t]*src:`)

func wikiNames(annotations []string) []string {
	var names []string
	for _, a := range gateAnnotations(annotations) {
		for _, match := range wikiRef.FindAllStringSubmatch(a, -1) {
			text := match[1]
			paths := markdownPaths(text)
			if len(paths) > 0 {
				for _, path := range paths {
					// Existing annotations use "daily" as a label before sources/ paths.
					if strings.HasPrefix(path, "daily sources/") {
						path = strings.TrimPrefix(path, "daily ")
					}
					names = append(names, path)
				}
				continue
			}
			for _, stop := range []string{". Repo:", ". Contract:", "|"} {
				text, _, _ = strings.Cut(text, stop)
			}
			// Bare page names form the leading comma-separated list; the rest is prose.
			text, _, _ = strings.Cut(text, " — ")
			text, _, _ = strings.Cut(text, " (")
			for name := range strings.SplitSeq(text, ",") {
				fields := strings.Fields(name)
				if len(fields) == 1 {
					names = append(names, fields[0])
				}
			}
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

func referenceBoundary(rest string) bool {
	if rest == "" {
		return true
	}
	r, _ := utf8.DecodeRuneInString(rest)
	return unicode.IsSpace(r) || strings.ContainsRune(",;|", r)
}

func markdownReference(text string) (path, rest string, ok bool) {
	// A marker is formatting only when its matching closer follows the whole reference.
	for _, pair := range []struct{ open, close string }{{"[[", "]]"}, {"[", "]"}, {"`", "`"}, {"\"", "\""}, {"'", "'"}} {
		if !strings.HasPrefix(text, pair.open) {
			continue
		}
		for _, end := range markdownEnd.FindAllStringIndex(text, -1) {
			if tail, closed := strings.CutPrefix(text[end[1]:], pair.close); closed && referenceBoundary(tail) {
				return text[len(pair.open):end[1]], tail, true
			}
		}
	}
	for _, stop := range []string{". Repo:", ". Contract:"} {
		text, _, _ = strings.Cut(text, stop)
	}
	if source := wikiSource.FindStringIndex(text); source != nil {
		text = text[:source[0]]
	}
	for _, end := range markdownEnd.FindAllStringIndex(text, -1) {
		if referenceBoundary(text[end[1]:]) {
			return text[:end[1]], text[end[1]:], true
		}
	}
	return "", text, false
}

func markdownPaths(text string) []string {
	var paths []string
	text = strings.TrimSpace(text)
	for text != "" {
		path, rest, ok := markdownReference(text)
		if !ok {
			break
		}
		paths = append(paths, path)
		text = strings.TrimSpace(rest)
		if strings.HasPrefix(text, "—") || strings.HasPrefix(text, "|") {
			break
		}
		// Parenthetical annotations after a reference end at a list separator or field end.
		if strings.HasPrefix(text, "(") {
			if end := strings.IndexByte(text, ')'); end >= 0 {
				tail := strings.TrimSpace(text[end+1:])
				if tail == "" || strings.HasPrefix(tail, ";") || strings.HasPrefix(tail, ",") {
					text = tail
				}
			}
		}
		text = strings.TrimLeftFunc(text, func(r rune) bool { return unicode.IsSpace(r) || r == ',' || r == ';' })
	}
	return paths
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func expandHome(path string) string {
	if rest, ok := strings.CutPrefix(path, "~/"); ok {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, rest)
		}
	}
	return path
}

func vaultFor(cfg Config, p *Project, t Todo) string {
	if p.Vault != nil {
		return p.Vault.Path
	}
	if path := cmp.Or(cfg.Vaults[p.ID], cfg.Vaults[t.Domain]); path != "" {
		return path
	}
	var best string
	for route := range cfg.Vaults {
		if filepath.IsAbs(route) && within(route, p.Root) && len(route) > len(best) {
			best = route
		}
	}
	return cfg.Vaults[best]
}

func sensitiveContextPath(path string) bool {
	for _, part := range strings.Split(filepath.Clean(path), string(filepath.Separator)) {
		name := strings.ToLower(part)
		if name == ".env" || strings.HasPrefix(name, ".env.") || strings.HasSuffix(name, ".key") || strings.HasSuffix(name, ".pem") || strings.Contains(name, "credentials") {
			return true
		}
	}
	return false
}

func wikiPage(cfg Config, vault, name string) (contextPage, error) {
	name = expandHome(name)
	if before, line, ok := strings.CutLast(name, ".md:"); ok && line != "" && strings.IndexFunc(line, func(r rune) bool { return r < '0' || r > '9' }) < 0 {
		name = before + ".md"
	}
	if sensitiveContextPath(name) {
		return contextPage{}, fmt.Errorf("wiki page %s is a sensitive path", name)
	}
	if filepath.IsAbs(name) {
		allowed := false
		for _, root := range cfg.Vaults {
			if within(root, name) && (vault == "" || root == vault) {
				allowed = true
			}
		}
		if !allowed {
			return contextPage{}, fmt.Errorf("wiki page %s is outside the project's configured vault", name)
		}
	} else {
		if vault == "" {
			return contextPage{}, fmt.Errorf("wiki page %s: no vault route for this todo", name)
		}
		if !strings.HasSuffix(name, ".md") {
			name += ".md"
		}
		// An annotation may spell the vault label before its relative page path.
		name = strings.TrimPrefix(name, filepath.Base(vault)+"/")
		direct := filepath.Join(vault, name)
		if !within(vault, direct) {
			return contextPage{}, fmt.Errorf("wiki page %s escapes its vault", name)
		}
		if strings.Contains(name, "/") {
			name = direct
		} else {
			var matches []string
			err := filepath.WalkDir(vault, func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() && (entry.Name() == ".git" || strings.HasPrefix(entry.Name(), ".")) && path != vault {
					return filepath.SkipDir
				}
				if !entry.IsDir() && entry.Name() == name {
					matches = append(matches, path)
				}
				return nil
			})
			if err != nil {
				return contextPage{}, fmt.Errorf("wiki page %s: %w", name, err)
			}
			if len(matches) != 1 {
				return contextPage{}, fmt.Errorf("wiki page %s: found %d matching pages in %s", name, len(matches), vault)
			}
			name = matches[0]
		}
	}
	if sensitiveContextPath(name) {
		return contextPage{}, fmt.Errorf("wiki page %s is a sensitive path", name)
	}
	real, err := filepath.EvalSymlinks(name)
	if err != nil {
		return contextPage{}, fmt.Errorf("wiki page %s: %w", name, err)
	}
	if sensitiveContextPath(real) {
		return contextPage{}, fmt.Errorf("wiki page %s resolves to a sensitive path", name)
	}
	allowed := false
	for _, root := range cfg.Vaults {
		realRoot, err := filepath.EvalSymlinks(root)
		if err == nil && (vault == "" || root == vault) && within(realRoot, real) {
			allowed = true
		}
	}
	if !allowed {
		return contextPage{}, fmt.Errorf("wiki page %s resolves outside configured vaults", name)
	}
	text, err := os.ReadFile(real)
	if err != nil {
		return contextPage{}, fmt.Errorf("wiki page %s: %w", name, err)
	}
	return contextPage{Path: real, Text: string(text)}, nil
}

func handoffMatches(text string, t Todo) bool {
	refs := []string{t.UUID}
	if len(t.UUID) >= 8 {
		refs = append(refs, t.UUID[:8])
	}
	for _, link := range t.Links {
		if link.Kind == "pr" {
			refs = append(refs, link.URL)
		}
	}
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		re := regexp.MustCompile(`(^|[^A-Za-z0-9])` + regexp.QuoteMeta(ref) + `($|[^A-Za-z0-9])`)
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

func todoContext(cfg Config, p *Project, t Todo, tabs []tabBrief, today string) (todoState, error) {
	state := todoState{ID: t.UUID, Today: today, Description: t.Description, Domain: t.Domain, Due: t.Due,
		Annotations: gateAnnotations(t.Annotations), Links: t.Links, ProjectTabs: tabs, OwnerPolicy: cfg.OwnerPolicy}
	names := wikiNames(t.Annotations)
	vault := vaultFor(cfg, p, t)
	if vault == "" {
		for _, name := range names {
			name = expandHome(name)
			if filepath.IsAbs(name) {
				for _, root := range cfg.Vaults {
					if within(root, name) {
						if vault != "" && vault != root {
							return state, fmt.Errorf("wiki pages span different vaults")
						}
						vault = root
					}
				}
			}
		}
	}
	for _, name := range names {
		page, err := wikiPage(cfg, vault, name)
		if err != nil {
			return state, err
		}
		if !slices.ContainsFunc(state.WikiPages, func(p contextPage) bool { return p.Path == page.Path }) {
			state.WikiPages = append(state.WikiPages, page)
		}
	}
	if cfg.Context.HandoffDir == "" {
		return state, nil
	}
	err := filepath.WalkDir(expandHome(cfg.Context.HandoffDir), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}
		if sensitiveContextPath(path) {
			return fmt.Errorf("handoff %s is a sensitive path", path)
		}
		real, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		if sensitiveContextPath(real) {
			return fmt.Errorf("handoff %s resolves to a sensitive path", path)
		}
		text, err := os.ReadFile(real)
		if err != nil {
			return err
		}
		if handoffMatches(string(text), t) {
			state.Handoffs = append(state.Handoffs, contextPage{Path: path, Text: string(text)})
		}
		return nil
	})
	if err != nil {
		return state, fmt.Errorf("handoff directory %s: %w", cfg.Context.HandoffDir, err)
	}
	return state, nil
}
