package main

import (
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

// readVault checks that the vault at path is readable and finds its newest
// page edit. Directories that start with a dot are skipped.
func readVault(path string) (Vault, error) {
	v := Vault{Path: path}
	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != path && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".md") {
			info, err := d.Info()
			if err != nil {
				return err
			}
			if info.ModTime().After(v.LastPageEdit) {
				v.LastPageEdit = info.ModTime().UTC().Truncate(time.Second)
			}
		}
		return nil
	})
	if err != nil {
		return Vault{Path: path}, err
	}
	v.Readable = true
	return v, nil
}
