package stashmeta

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shadmeoli/kairos/internal/store"
)

const metaVersion = 1

// Entry is kairos metadata for one git stash (keyed by stash commit SHA).
type Entry struct {
	StashSHA     string    `json:"stash_sha"`
	ParentBranch string    `json:"parent_branch,omitempty"`
	Detached     bool      `json:"detached"`
	ParentHEAD   string    `json:"parent_head,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	Files        []string  `json:"files"`
	Message      string    `json:"message,omitempty"`
}

type file struct {
	Version int               `json:"version"`
	Entries map[string]Entry `json:"entries"`
}

func metaPath(repoTop string) string {
	return filepath.Join(store.KairosDir(repoTop), "stash-meta.json")
}

func loadFile(repoTop string) (file, error) {
	path := metaPath(repoTop)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return file{Version: metaVersion, Entries: map[string]Entry{}}, nil
		}
		return file{}, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return file{}, fmt.Errorf("stash-meta: %w", err)
	}
	if f.Version != metaVersion {
		return file{}, fmt.Errorf("stash-meta: unsupported version %d", f.Version)
	}
	if f.Entries == nil {
		f.Entries = map[string]Entry{}
	}
	return f, nil
}

func saveFile(repoTop string, f file) error {
	if err := os.MkdirAll(store.KairosDir(repoTop), 0o755); err != nil {
		return err
	}
	f.Version = metaVersion
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := metaPath(repoTop)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Upsert writes metadata for a stash commit hash.
func Upsert(repoTop string, e Entry) error {
	if e.StashSHA == "" {
		return errors.New("stash-meta: empty stash_sha")
	}
	_ = store.EnsureGitignore(repoTop)
	f, err := loadFile(repoTop)
	if err != nil {
		return err
	}
	f.Entries[e.StashSHA] = e
	return saveFile(repoTop, f)
}

// Get returns metadata for this stash commit SHA.
func Get(repoTop, stashSHA string) (Entry, bool) {
	f, err := loadFile(repoTop)
	if err != nil {
		return Entry{}, false
	}
	e, ok := f.Entries[stashSHA]
	return e, ok
}

// Delete removes metadata for a stash that was popped/dropped.
func Delete(repoTop, stashSHA string) error {
	if stashSHA == "" {
		return nil
	}
	f, err := loadFile(repoTop)
	if err != nil {
		return err
	}
	delete(f.Entries, stashSHA)
	return saveFile(repoTop, f)
}
