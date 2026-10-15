package main

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Commands    Commands          `toml:"commands"`
	Profiles    []Profile         `toml:"profiles"`
	Pinned      Pinned            `toml:"pinned"`
	Vaults      map[string]string `toml:"vaults"`
	Thresholds  Thresholds        `toml:"thresholds"`
	OwnerPolicy string            `toml:"-"`
	Context     ContextConfig     `toml:"context"`
	Gate        GateConfig        `toml:"gate"`
}

type Pinned struct {
	Tabs       []string `toml:"tabs"`
	Workspaces []string `toml:"workspaces"`
}

type Commands struct {
	Task  string `toml:"task"`
	Herdr string `toml:"herdr"`
	Gh    string `toml:"gh"`
	Git   string `toml:"git"`
}

type Profile struct {
	Name string `toml:"name"`
	Kind string `toml:"kind"`
	Dir  string `toml:"dir"`
}

type Thresholds struct {
	Floor        float64 `toml:"floor"`
	Low          float64 `toml:"low"`
	High         float64 `toml:"high"`
	WindowHours  int     `toml:"window_hours"`
	TailBytes    int     `toml:"tail_bytes"`
	ExcerptChars int     `toml:"excerpt_chars"`
}

type GateConfig struct {
	TodoContextEnabled bool   `toml:"todo_context_enabled"`
	URL                string `toml:"url"`
	Log                string `toml:"log"`
}

const defaultGateURL = "https://api.typesafe.ai/v1/systemone"

var tomlFence = regexp.MustCompile("(?ms)^\x60\x60\x60toml[ \t]*\r?\n(.*?)^\x60\x60\x60[ \t]*\r?$")

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("local file: %w", err)
	}
	var doc bytes.Buffer
	for _, m := range tomlFence.FindAllSubmatch(data, -1) {
		doc.Write(m[1])
		doc.WriteByte('\n')
	}
	var cfg Config
	dec := toml.NewDecoder(&doc)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		if strict, ok := errors.AsType[*toml.StrictMissingError](err); ok {
			return Config{}, fmt.Errorf("local file %s: unknown field:\n%s", path, strict.String())
		}
		return Config{}, fmt.Errorf("local file %s: %w", path, err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if _, policy, ok := strings.Cut(line, "Assignee split:"); ok {
			cfg.OwnerPolicy = strings.TrimSpace(policy)
		}
	}
	cfg.Commands.Herdr = cmp.Or(cfg.Commands.Herdr, "herdr")
	cfg.Commands.Gh = cmp.Or(cfg.Commands.Gh, "gh")
	cfg.Commands.Git = cmp.Or(cfg.Commands.Git, "git")
	cfg.Gate.URL = cmp.Or(cfg.Gate.URL, defaultGateURL)
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("local file %s: %w", path, err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	var errs []error
	if c.Commands.Task == "" {
		errs = append(errs, errors.New("commands.task is required"))
	}
	if c.Gate.Log == "" {
		errs = append(errs, errors.New("gate.log is required"))
	}
	t := c.Thresholds
	if t.Floor <= 0 || t.Low < t.Floor || t.High < t.Low || t.High > 1 {
		errs = append(errs, errors.New("thresholds need 0 < floor <= low <= high <= 1"))
	}
	if t.WindowHours <= 0 || t.TailBytes <= 0 || t.ExcerptChars <= 0 {
		errs = append(errs, errors.New("thresholds.window_hours, tail_bytes and excerpt_chars must be positive"))
	}
	seen := map[string]bool{}
	for _, p := range c.Profiles {
		if p.Name == "" || p.Dir == "" || !slices.Contains([]string{"claude", "codex"}, p.Kind) {
			errs = append(errs, fmt.Errorf("profile %q needs a name, a dir and kind claude or codex", p.Name))
		}
		if seen[p.Name] {
			errs = append(errs, fmt.Errorf("profile %q is listed twice", p.Name))
		}
		seen[p.Name] = true
	}
	return errors.Join(errs...)
}

type ContextConfig struct {
	HandoffDir string `toml:"handoff_dir"`
}
