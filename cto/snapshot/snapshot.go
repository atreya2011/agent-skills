package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// collector gathers the facts of one sweep. A source that fails is recorded in
// bad and the sweep goes on.
type collector struct {
	cfg    Config
	r      runner
	now    time.Time
	window time.Duration

	mu  sync.Mutex
	bad []Unreadable
}

// fail lists a source the snapshot could not read.
func (c *collector) fail(source, target string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bad = append(c.bad, Unreadable{Source: source, Target: target, Error: err.Error()})
}

type rootResult struct {
	root   string
	isRepo bool
}

// collect reads every source, groups the facts by project and judges them. It
// never fails: what it cannot read goes to the unreadable list.
func collect(ctx context.Context, cfg Config, gate *Gate, now time.Time) Snapshot {
	c := &collector{cfg: cfg, r: runner{cmds: cfg.Commands}, now: now,
		window: time.Duration(cfg.Thresholds.WindowHours) * time.Hour}

	lists := map[string]herdrLists{}
	for _, noun := range []string{"workspace", "tab", "agent"} {
		l, err := c.r.herdrList(ctx, noun)
		if err != nil {
			c.fail("herdr "+noun+" list", "", err)
		}
		lists[noun] = l
	}
	workspaces, tabs := buildHerdr(cfg, lists["workspace"].Result.Workspaces, lists["tab"].Result.Tabs, lists["agent"].Result.Agents)
	todos, err := c.r.todos(ctx, now, c.window)
	if err != nil {
		c.fail("task export", "", err)
	}
	ix, bad := buildIndex(cfg.Profiles, now, c.window)
	c.bad = append(c.bad, bad...)

	boundTab := c.bindTabs(tabs, ix)
	sessions := c.readSessions(ix, boundTab)

	projects := c.groupProjects(ctx, tabs, sessions, todos)
	c.readPullRequests(ctx, projects)
	c.readLinks(ctx, projects)
	for _, p := range projects {
		if path, ok := cfg.Vaults[p.ID]; ok {
			p.Vault = &Vault{Path: path, Readable: true}
			if _, err := os.ReadDir(path); err != nil {
				c.fail("vault", path, err)
				p.Vault.Readable = false
			}
		}
	}

	excerptFor := func(id string) string {
		f, ok := ix.byID[id]
		if !ok {
			c.fail("transcript", id, errors.New("a todo cites this session but no profile holds it"))
			return ""
		}
		excerpt, _, err := readTranscript(f, cfg.Thresholds.TailBytes, cfg.Thresholds.ExcerptChars)
		if err != nil {
			c.fail("transcript", id, err)
		}
		return excerpt
	}
	judgments := judgeAll(ctx, gate, projects, now, excerptFor)

	snap := Snapshot{
		GeneratedAt: now.UTC(), WindowHours: cfg.Thresholds.WindowHours,
		Workspaces: workspaces, Profiles: ix.infos, Judgments: judgments, Unreadable: c.bad,
	}
	for _, p := range projects {
		snap.Projects = append(snap.Projects, *p)
	}
	slices.SortFunc(snap.Unreadable, func(a, b Unreadable) int {
		return cmp.Or(cmp.Compare(a.Source, b.Source), cmp.Compare(a.Target, b.Target), cmp.Compare(a.Error, b.Error))
	})
	return snap
}

// bindTabs sets each agent tab's age from its transcript and lists the tabs
// with no readable transcript. A kind of agent that herdr gives no session ID
// is listed once, not once per tab. It returns the tab ID of every bound
// session.
func (c *collector) bindTabs(tabs []Tab, ix *transcriptIndex) map[string]string {
	bound := map[string]string{}
	noSession := map[string][]string{} // agent kind -> labels of its tabs without a session ID
	for i := range tabs {
		t := &tabs[i]
		if t.Agent == "" {
			continue
		}
		if t.SessionID == "" {
			noSession[t.Agent] = append(noSession[t.Agent], t.Label)
			continue
		}
		bound[t.SessionID] = t.ID
		f, ok := ix.byID[t.SessionID]
		if !ok {
			c.fail("transcript", t.Label, errors.New("no profile holds a transcript for session "+t.SessionID))
			continue
		}
		t.AgeS = new(int(c.now.Sub(f.Modified).Seconds()))
	}
	for _, kind := range slices.Sorted(maps.Keys(noSession)) {
		c.fail("transcript", kind, fmt.Errorf("herdr reports no session ID for %d %s tabs (%s), so their age is unknown",
			len(noSession[kind]), kind, strings.Join(noSession[kind], ", ")))
	}
	return bound
}

// readSessions reads the tail of every transcript written within the window,
// newest first.
func (c *collector) readSessions(ix *transcriptIndex, boundTab map[string]string) []Session {
	recent := ix.recent(c.now, c.window)
	var sessions []Session
	for _, id := range slices.Sorted(maps.Keys(recent)) {
		f := recent[id]
		excerpt, cwd, err := readTranscript(f, c.cfg.Thresholds.TailBytes, c.cfg.Thresholds.ExcerptChars)
		if err != nil {
			c.fail("transcript", id, err)
			continue
		}
		sessions = append(sessions, Session{
			Profile: f.Profile, ID: id, Modified: f.Modified.UTC(), AgeS: int(c.now.Sub(f.Modified).Seconds()),
			CWD: cwd, TabID: boundTab[id], Excerpt: excerpt,
		})
	}
	slices.SortFunc(sessions, func(a, b Session) int {
		return cmp.Or(b.Modified.Compare(a.Modified), cmp.Compare(a.ID, b.ID))
	})
	return sessions
}

// groupProjects places tabs, sessions and todos in projects. A tab or session
// goes to the project of its working directory, a todo to the project of the
// first GitHub repository it cites, and anything else to the none group.
func (c *collector) groupProjects(ctx context.Context, tabs []Tab, sessions []Session, todos []Todo) []*Project {
	var cwds []string
	for _, t := range tabs {
		if t.CWD != "" {
			cwds = append(cwds, t.CWD)
		}
	}
	for _, s := range sessions {
		if s.TabID == "" && s.CWD != "" {
			cwds = append(cwds, s.CWD)
		}
	}
	slices.Sort(cwds)
	cwds = slices.Compact(cwds)

	roots := make([]rootResult, len(cwds))
	parallel(cwds, func(i int, cwd string) {
		root, isRepo, err := c.r.gitRoot(ctx, cwd)
		if err != nil {
			c.fail("git", cwd, err)
		}
		roots[i] = rootResult{root: root, isRepo: isRepo}
	})
	var repoRoots []string
	for _, r := range roots {
		if r.isRepo {
			repoRoots = append(repoRoots, r.root)
		}
	}
	slices.Sort(repoRoots)
	repoRoots = slices.Compact(repoRoots)
	details := make([]gitDetails, len(repoRoots))
	parallel(repoRoots, func(i int, root string) {
		d, err := c.r.gitDetails(ctx, root)
		if err != nil {
			c.fail("git", root, err)
		}
		details[i] = d
	})

	byID := map[string]*Project{}
	get := func(id, label string) *Project {
		if p, ok := byID[id]; ok {
			return p
		}
		p := &Project{ID: id, Label: label, Commits: []Commit{}, PRs: []PR{}, Tabs: []Tab{}, Todos: []Todo{}, Sessions: []Session{}}
		byID[id] = p
		return p
	}
	projectOf := map[string]string{}
	for i, cwd := range cwds {
		if !roots[i].isRepo {
			projectOf[cwd] = get(cwd, filepath.Base(cwd)).ID
			continue
		}
		root := roots[i].root
		d := details[slices.Index(repoRoots, root)]
		id := cmp.Or(d.Slug, root)
		p := get(id, filepath.Base(id))
		p.slug = d.Slug
		p.Root, p.Branch, p.DefaultBranch = root, d.Branch, d.DefaultBranch
		p.Dirty, p.Ahead, p.Behind = d.Dirty, d.Ahead, d.Behind
		if len(d.Commits) > 0 {
			p.Commits = d.Commits
		}
		projectOf[cwd] = id
	}
	none := func() *Project { return get(noProject, "no project") }
	place := func(id string) *Project {
		if p, ok := byID[id]; ok {
			return p
		}
		return none()
	}
	tabProject := map[string]string{}
	for _, t := range tabs {
		p := place(projectOf[t.CWD])
		tabProject[t.ID] = p.ID
		p.Tabs = append(p.Tabs, t)
	}
	for _, s := range sessions {
		id := projectOf[s.CWD]
		if s.TabID != "" {
			id = tabProject[s.TabID]
		}
		p := place(id)
		p.Sessions = append(p.Sessions, s)
	}
	for _, t := range todos {
		p := none()
		if slug := t.slug(); slug != "" {
			p = get(slug, filepath.Base(slug))
			p.slug = slug
		}
		p.Todos = append(p.Todos, t)
	}

	return slices.SortedFunc(maps.Values(byID), func(a, b *Project) int {
		return cmp.Or(compareBool(a.ID == noProject, b.ID == noProject), cmp.Compare(a.Label, b.Label), cmp.Compare(a.ID, b.ID))
	})
}

// compareBool orders false before true.
func compareBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case b:
		return -1
	}
	return 1
}

// readPullRequests reads the pull requests of every project that has a GitHub
// repository.
func (c *collector) readPullRequests(ctx context.Context, projects []*Project) {
	var withSlug []*Project
	for _, p := range projects {
		if p.slug != "" {
			withSlug = append(withSlug, p)
		}
	}
	parallel(withSlug, func(_ int, p *Project) {
		prs, err := c.r.pullRequests(ctx, p.slug)
		if err != nil {
			c.fail("gh pr list", p.slug, err)
			return
		}
		p.PRs = prs
	})
}

// linkRef addresses one link of one todo of one project.
type linkRef struct {
	p    *Project
	todo int
	link int
}

// readLinks reads the state of every issue and pull request an open +cto todo
// cites, then checks its cited commits against the project's default branch
// and marks the todos that are ready to dispatch.
func (c *collector) readLinks(ctx context.Context, projects []*Project) {
	var refs []linkRef
	for _, p := range projects {
		for ti, t := range p.Todos {
			if t.Assignee != "cto" || !open(t) {
				continue
			}
			for li, l := range t.Links {
				switch l.Kind {
				case "issue", "pr":
					refs = append(refs, linkRef{p, ti, li})
				case "commit":
					if p.Root != "" {
						p.Todos[ti].Links[li].State = c.r.onDefault(ctx, p.Root, p.DefaultBranch, l.SHA)
					}
				}
			}
		}
	}
	parallel(refs, func(_ int, ref linkRef) {
		l := &ref.p.Todos[ref.todo].Links[ref.link]
		if err := c.r.fillLink(ctx, l); err != nil {
			l.Unreadable = true
			c.fail("gh "+l.Kind+" view", l.URL, err)
		}
	})
	for _, p := range projects {
		for ti, t := range p.Todos {
			p.Todos[ti].Ready = slices.ContainsFunc(t.Links, func(l Link) bool {
				return l.Kind == "issue" && l.State == "OPEN" && slices.Contains(l.Labels, "ready-for-agent")
			})
		}
	}
}
