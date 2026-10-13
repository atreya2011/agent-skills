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
)

type contextPage struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

var wikiRef = regexp.MustCompile(`(?i)\bwiki:\s*([^\n]+)`)

func wikiNotes(annotations []string) []string {
	var notes []string
	for _, annotation := range gateAnnotations(annotations) {
		for _, match := range wikiRef.FindAllStringSubmatch(annotation, -1) {
			notes = append(notes, match[1])
		}
	}
	return notes
}

func wikiPages(vault, note string, absoluteOnly bool) ([]string, error) {
	aliases := map[string][]string{}
	err := filepath.WalkDir(vault, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || strings.HasPrefix(entry.Name(), ".")) && path != vault {
			return filepath.SkipDir
		}
		if entry.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(vault, path)
		if err != nil {
			return err
		}
		names := []string{path}
		if !absoluteOnly {
			names = append(names, rel, entry.Name(), strings.TrimSuffix(entry.Name(), ".md"))
		}
		for _, name := range names {
			if !slices.Contains(aliases[name], path) {
				aliases[name] = append(aliases[name], path)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if home, err := os.UserHomeDir(); err == nil {
		note = strings.ReplaceAll(note, "~/", home+string(filepath.Separator))
	}
	var pages []string
	for name, matches := range aliases {
		pattern := regexp.MustCompile(`(^|[^\pL\p{Nd}_./-])` + regexp.QuoteMeta(name) + `($|[^\pL\p{Nd}_./-])`)
		if !pattern.MatchString(note) {
			continue
		}
		if len(matches) != 1 {
			return nil, fmt.Errorf("wiki page %s: found %d matching pages in %s", name, len(matches), vault)
		}
		pages = append(pages, matches[0])
	}
	slices.Sort(pages)
	return slices.Compact(pages), nil
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
	notes := wikiNotes(t.Annotations)
	vault := vaultFor(cfg, p, t)
	if vault == "" {
		for _, note := range notes {
			for _, root := range cfg.Vaults {
				matches, err := wikiPages(root, note, true)
				if err != nil {
					return state, err
				}
				if len(matches) != 0 {
					if vault != "" && vault != root {
						return state, fmt.Errorf("wiki pages span different vaults")
					}
					vault = root
				}
			}
		}
	}
	for _, note := range notes {
		if vault == "" {
			return state, fmt.Errorf("wiki note: no vault route for this todo")
		}
		names, err := wikiPages(vault, note, false)
		if err != nil {
			return state, err
		}
		if len(names) == 0 {
			return state, fmt.Errorf("wiki note %q: found 0 matching pages in %s", note, vault)
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
