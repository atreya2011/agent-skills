package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const goodThresholds = "[thresholds]\nfloor = 0.6\nlow = 0.6\nhigh = 0.8\nwindow_hours = 6\ntail_bytes = 65536\nexcerpt_chars = 800\n"

// localFile joins markdown prose and toml blocks the way a real local file
// interleaves them.
func localFile(blocks ...string) string {
	out := "# CTO local file\n\nProse the tool ignores.\n\n"
	for _, b := range blocks {
		out += "## Section\n\n```toml\n" + b + "```\n\nMore prose.\n\n"
	}
	return out
}

func TestLoadConfig(t *testing.T) {
	base := []string{"[commands]\ntask = \"/opt/task\"\n", "[gate]\nlog = \"/var/gate.log\"\n", goodThresholds}
	tests := []struct {
		name    string
		file    string
		wantErr string
		check   func(t *testing.T, c Config)
	}{
		{
			name: "blocks across sections join into one config with defaults",
			file: localFile(append(base,
				"[pinned]\ntabs = [\"notes\"]\nworkspaces = [\"home\"]\n[vaults]\n\"example/alpha\" = \"/vaults/alpha\"\n[[profiles]]\nname = \"main\"\nkind = \"claude\"\ndir = \"/p/main\"\n")...),
			check: func(t *testing.T, c Config) {
				assert.Equal(t, "/opt/task", c.Commands.Task)
				assert.Equal(t, "herdr", c.Commands.Herdr)
				assert.Equal(t, "gh", c.Commands.Gh)
				assert.Equal(t, "git", c.Commands.Git)
				assert.Equal(t, defaultGateURL, c.Gate.URL)
				assert.Equal(t, defaultGateModel, c.Gate.Model)
				assert.Equal(t, 0.8, c.Thresholds.High)
				assert.Equal(t, []Profile{{Name: "main", Kind: "claude", Dir: "/p/main"}}, c.Profiles)
				assert.Equal(t, "/vaults/alpha", c.Vaults["example/alpha"])
				assert.Equal(t, Pinned{Tabs: []string{"notes"}, Workspaces: []string{"home"}}, c.Pinned)
			},
		},
		{
			name:    "unknown field is rejected",
			file:    localFile(append(base, "mystery = 1\n")...),
			wantErr: "mystery",
		},
		{
			name:    "an api key in the file is rejected",
			file:    localFile(base[0], "[gate]\nlog = \"/var/gate.log\"\napi_key = \"x\"\n", goodThresholds),
			wantErr: "api_key",
		},
		{
			name:    "missing task command and log",
			file:    localFile(goodThresholds),
			wantErr: "commands.task is required",
		},
		{
			name:    "thresholds out of order",
			file:    localFile(base[0], base[1], "[thresholds]\nfloor = 0.6\nlow = 0.5\nhigh = 0.8\nwindow_hours = 6\ntail_bytes = 1\nexcerpt_chars = 1\n"),
			wantErr: "thresholds need",
		},
		{
			name:    "missing bounds",
			file:    localFile(base[0], base[1], "[thresholds]\nfloor = 0.6\nlow = 0.6\nhigh = 0.8\n"),
			wantErr: "must be positive",
		},
		{
			name:    "bad profile kind",
			file:    localFile(append(base, "[[profiles]]\nname = \"p\"\nkind = \"cursor\"\ndir = \"/p\"\n")...),
			wantErr: "kind claude or codex",
		},
		{
			name:    "duplicate profile",
			file:    localFile(append(base, "[[profiles]]\nname = \"p\"\nkind = \"codex\"\ndir = \"/a\"\n[[profiles]]\nname = \"p\"\nkind = \"codex\"\ndir = \"/b\"\n")...),
			wantErr: "listed twice",
		},
		{
			name:    "invalid toml",
			file:    localFile("not toml ==\n"),
			wantErr: "local file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "cto.md")
			require.NoError(t, os.WriteFile(path, []byte(tt.file), 0o600))
			c, err := loadConfig(path)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			tt.check(t, c)
		})
	}

	t.Run("missing file", func(t *testing.T) {
		_, err := loadConfig(filepath.Join(t.TempDir(), "absent.md"))
		require.ErrorIs(t, err, os.ErrNotExist)
	})
}

// TestSkillExampleParses keeps the settings example in the skill text valid:
// the tool must accept the block the Local file section tells users to write.
func TestSkillExampleParses(t *testing.T) {
	c, err := loadConfig(filepath.Join("..", "SKILL.md"))
	require.NoError(t, err)
	assert.Equal(t, 0.8, c.Thresholds.High)
	require.Len(t, c.Profiles, 1)
	assert.Equal(t, "claude", c.Profiles[0].Kind)
	assert.Equal(t, defaultGateURL, c.Gate.URL)
}
