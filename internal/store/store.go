package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/shadmeoli/kairos/internal/config"
)

const DirName = ".kairos"
const StateFile = "state.json"

type Checkpoint struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"created_at"`
	HEAD          string    `json:"head"`
	Branch        string    `json:"branch,omitempty"`
	Detached      bool      `json:"detached"`
	Label         string    `json:"label,omitempty"`
	Note          string    `json:"note,omitempty"`
	StashMessage  string    `json:"stash_message,omitempty"`
	PrevStashRefs []string  `json:"prev_stash_refs,omitempty"`
}

type State struct {
	Version     int            `json:"version"`
	Checkpoints []Checkpoint   `json:"checkpoints"`
	Cursor      int            `json:"cursor"`
	RepoTop     string         `json:"repo_top"`
	CreatedWith string         `json:"created_with,omitempty"`
}

func DefaultState(repoTop string) State {
	return State{
		Version:     1,
		Checkpoints: nil,
		Cursor:      -1,
		RepoTop:     repoTop,
		CreatedWith: "kairos",
	}
}

func KairosDir(repoTop string) string {
	return filepath.Join(repoTop, DirName)
}

func StatePath(repoTop string) string {
	return filepath.Join(KairosDir(repoTop), StateFile)
}

func Load(repoTop string) (State, error) {
	path := StatePath(repoTop)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultState(repoTop), nil
		}
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, fmt.Errorf("parse state: %w", err)
	}
	if s.Version != 1 {
		return State{}, fmt.Errorf("unsupported state version %d", s.Version)
	}
	if s.RepoTop != "" && s.RepoTop != repoTop {
		return State{}, fmt.Errorf("state file repo_top %q does not match %q", s.RepoTop, repoTop)
	}
	s.RepoTop = repoTop
	if s.Cursor >= len(s.Checkpoints) {
		s.Cursor = len(s.Checkpoints) - 1
	}
	max, err := config.EffectiveMaxHistory(repoTop)
	if err != nil {
		return State{}, err
	}
	trimmed := TrimHistoryWindow(&s, max)
	if trimmed {
		if err := Save(repoTop, s); err != nil {
			return State{}, err
		}
	}
	return s, nil
}

// TrimHistoryWindow drops the oldest checkpoints so len(s.Checkpoints) <= max
// and moves the cursor to stay on the same logical checkpoint when possible.
// It returns true if any checkpoint was removed. max must be >= 1.
func TrimHistoryWindow(s *State, max int) bool {
	if max < 1 {
		max = 1
	}
	if len(s.Checkpoints) <= max {
		return false
	}
	drop := len(s.Checkpoints) - max
	s.Checkpoints = s.Checkpoints[drop:]
	s.Cursor -= drop
	if len(s.Checkpoints) == 0 {
		s.Cursor = -1
		return true
	}
	if s.Cursor < 0 {
		s.Cursor = 0
	}
	if s.Cursor >= len(s.Checkpoints) {
		s.Cursor = len(s.Checkpoints) - 1
	}
	return true
}

func Save(repoTop string, s State) error {
	dir := KairosDir(repoTop)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	s.RepoTop = repoTop
	s.Version = 1
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := StatePath(repoTop) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, StatePath(repoTop))
}

func EnsureGitignore(repoTop string) error {
	gi := filepath.Join(repoTop, ".gitignore")
	line := "/" + DirName + "/"
	b, err := os.ReadFile(gi)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	content := string(b)
	if strings.Contains(content, line) || strings.Contains(content, DirName+"/") {
		return nil
	}
	appendix := line + "\n"
	if len(b) > 0 && !strings.HasSuffix(content, "\n") {
		appendix = "\n" + appendix
	}
	f, err := os.OpenFile(gi, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(appendix)
	return err
}

func NewCheckpoint(head, branch string, detached bool, label, note string) Checkpoint {
	return Checkpoint{
		ID:        uuid.NewString(),
		CreatedAt: time.Now().UTC(),
		HEAD:      head,
		Branch:    branch,
		Detached:  detached,
		Label:     strings.TrimSpace(label),
		Note:      strings.TrimSpace(note),
	}
}
