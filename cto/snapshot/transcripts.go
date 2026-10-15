package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

type transcriptFile struct {
	Profile  string
	Kind     string
	Path     string
	Modified time.Time
}

type transcriptIndex struct {
	byID  map[string]transcriptFile
	infos []ProfileInfo
}

const uuidLen = 36

func buildIndex(profiles []Profile, now time.Time, window time.Duration) (*transcriptIndex, []Unreadable) {
	ix := &transcriptIndex{byID: map[string]transcriptFile{}}
	var bad []Unreadable
	owner := map[string]string{}
	for _, p := range profiles {
		info := ProfileInfo{Name: p.Name, Kind: p.Kind}
		real, err := filepath.EvalSymlinks(p.Dir)
		if err != nil {
			bad = append(bad, Unreadable{Source: "transcripts", Target: p.Name, Error: err.Error()})
			ix.infos = append(ix.infos, info)
			continue
		}
		if first, shared := owner[real]; shared {
			info.SameAs = first
			ix.infos = append(ix.infos, info)
			continue
		}
		owner[real] = p.Name
		files, errs := scanTranscripts(real, p.Kind)
		if len(errs) > 0 {
			msg := errs[0].Error()
			if len(errs) > 1 {
				msg += fmt.Sprintf(" (and %d more)", len(errs)-1)
			}
			bad = append(bad, Unreadable{Source: "transcripts", Target: p.Name, Error: msg})
		}
		for id, f := range files {
			f.Profile, f.Kind = p.Name, p.Kind
			ix.byID[id] = f
			if f.Modified.After(info.LastActivity) {
				info.LastActivity = f.Modified.UTC()
			}
			if now.Sub(f.Modified) <= window {
				info.SessionsInWindow++
			}
		}
		ix.infos = append(ix.infos, info)
	}
	return ix, bad
}

func scanTranscripts(dir, kind string) (map[string]transcriptFile, []error) {
	found := map[string]transcriptFile{}
	var errs []error
	add := func(id, path string, d fs.DirEntry) {
		info, err := d.Info()
		if err != nil {
			errs = append(errs, err)
			return
		}
		found[id] = transcriptFile{Path: path, Modified: info.ModTime()}
	}
	if kind == "claude" {
		projects, err := os.ReadDir(dir)
		if err != nil {
			return nil, []error{err}
		}
		for _, proj := range projects {
			if !proj.IsDir() {
				continue
			}
			entries, err := os.ReadDir(filepath.Join(dir, proj.Name()))
			if err != nil {
				errs = append(errs, err)
				continue
			}
			for _, e := range entries {
				if id, ok := strings.CutSuffix(e.Name(), ".jsonl"); ok && !e.IsDir() {
					add(id, filepath.Join(dir, proj.Name(), e.Name()), e)
				}
			}
		}
		return found, errs
	}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		stem, ok := strings.CutSuffix(d.Name(), ".jsonl")
		if ok && !d.IsDir() && strings.HasPrefix(stem, "rollout-") && len(stem) >= uuidLen {
			add(stem[len(stem)-uuidLen:], path, d)
		}
		return nil
	})
	if err != nil {
		errs = append(errs, err)
	}
	return found, errs
}

func (ix *transcriptIndex) recent(now time.Time, window time.Duration) map[string]transcriptFile {
	out := map[string]transcriptFile{}
	for id, f := range ix.byID {
		if now.Sub(f.Modified) <= window {
			out[id] = f
		}
	}
	return out
}

func readTranscript(f transcriptFile, tailBytes, excerptChars int) (excerpt, cwd string, err error) {
	file, err := os.Open(f.Path)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = file.Close() }()
	st, err := file.Stat()
	if err != nil {
		return "", "", err
	}
	start := max(0, st.Size()-int64(tailBytes))
	buf := make([]byte, st.Size()-start)
	if _, err := file.ReadAt(buf, start); err != nil && !errors.Is(err, io.EOF) {
		return "", "", err
	}
	if start > 0 {
		_, rest, found := bytes.Cut(buf, []byte("\n"))
		if !found {
			return "", "", nil
		}
		buf = rest
	}
	var parts []string
	for line := range bytes.SplitSeq(buf, []byte("\n")) {
		role, text, lineCWD := parseLine(f.Kind, line)
		if lineCWD != "" {
			cwd = lineCWD
		}
		if text != "" {
			parts = append(parts, role+": "+text)
		}
	}
	runes := []rune(strings.Join(parts, "\n"))
	if len(runes) > excerptChars {
		runes = runes[len(runes)-excerptChars:]
	}
	return string(runes), cwd, nil
}

func parseLine(kind string, line []byte) (role, text, cwd string) {
	var contentPath string
	switch kind {
	case "claude":
		cwd = gjson.GetBytes(line, "cwd").String()
		role = gjson.GetBytes(line, "type").String()
		contentPath = "message.content"
	default:
		cwd = gjson.GetBytes(line, "payload.cwd").String()
		if gjson.GetBytes(line, "type").String() == "response_item" && gjson.GetBytes(line, "payload.type").String() == "message" {
			role = gjson.GetBytes(line, "payload.role").String()
		}
		contentPath = "payload.content"
	}
	if !slices.Contains([]string{"user", "assistant"}, role) {
		return "", "", cwd
	}
	content := gjson.GetBytes(line, contentPath)
	if content.Type == gjson.String {
		return role, strings.TrimSpace(content.String()), cwd
	}
	var texts []string
	content.ForEach(func(_, item gjson.Result) bool {
		if t := strings.TrimSpace(item.Get("text").String()); t != "" {
			texts = append(texts, t)
		}
		return true
	})
	return role, strings.Join(texts, "\n"), cwd
}
