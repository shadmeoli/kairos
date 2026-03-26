package timeline

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/shadmeoli/kairos/internal/config"
	"github.com/shadmeoli/kairos/internal/gitexec"
	"github.com/shadmeoli/kairos/internal/store"
)

var ErrDirtyTree = errors.New("working tree has local changes; pass --stash to auto-stash or commit/stash manually")

type Navigator struct {
	Repo string
}

func (n Navigator) CaptureCurrent() (store.Checkpoint, error) {
	root := gitexec.RepoRoot(n.Repo)
	head, err := gitexec.HEAD(root)
	if err != nil {
		return store.Checkpoint{}, err
	}
	detached, err := gitexec.IsDetached(root)
	if err != nil {
		return store.Checkpoint{}, err
	}
	branch, err := gitexec.CurrentBranch(root)
	if err != nil {
		return store.Checkpoint{}, err
	}
	return store.NewCheckpoint(head, branch, detached, "", ""), nil
}

// Save appends a checkpoint after the cursor (truncates forward history).
func Save(repoTop string, label, note string) (store.State, error) {
	s, err := store.Load(repoTop)
	if err != nil {
		return store.State{}, err
	}
	_ = store.EnsureGitignore(repoTop)

	nav := Navigator{Repo: repoTop}
	cp, err := nav.CaptureCurrent()
	if err != nil {
		return store.State{}, err
	}
	cp.Label = strings.TrimSpace(label)
	cp.Note = strings.TrimSpace(note)

	if s.Cursor < 0 {
		s.Checkpoints = append(s.Checkpoints, cp)
		s.Cursor = 0
	} else {
		// truncate redo stack
		s.Checkpoints = append(s.Checkpoints[:s.Cursor+1], cp)
		s.Cursor = len(s.Checkpoints) - 1
	}
	max, err := config.EffectiveMaxHistory(repoTop)
	if err != nil {
		return store.State{}, err
	}
	store.TrimHistoryWindow(&s, max)
	return s, store.Save(repoTop, s)
}

func applyCheckpoint(repoTop string, cp store.Checkpoint, stash bool) ([]string, error) {
	root := gitexec.RepoRoot(repoTop)
	clean, err := gitexec.IsClean(root)
	if err != nil {
		return nil, err
	}
	var stashes []string
	if !clean {
		if !stash {
			return nil, ErrDirtyTree
		}
		msg := fmt.Sprintf("kairos:nav:%s", cp.ID)
		if err := gitexec.StashPush(root, msg); err != nil {
			return nil, err
		}
		stashes = append(stashes, msg)
	}

	target := cp.Branch
	if cp.Detached || target == "" {
		target = cp.HEAD
	}
	if err := gitexec.Checkout(root, target); err != nil {
		return stashes, err
	}
	return stashes, nil
}

func Back(repoTop string, stash bool) (store.State, error) {
	s, err := store.Load(repoTop)
	if err != nil {
		return store.State{}, err
	}
	if s.Cursor <= 0 {
		return s, errors.New("no previous checkpoint")
	}
	next := s.Checkpoints[s.Cursor-1]
	stashes, err := applyCheckpoint(repoTop, next, stash)
	if err != nil {
		return s, err
	}
	if len(stashes) > 0 {
		next.PrevStashRefs = append(next.PrevStashRefs, stashes...)
		s.Checkpoints[s.Cursor-1] = next
	}
	s.Cursor--
	return s, store.Save(repoTop, s)
}

func Forward(repoTop string, stash bool) (store.State, error) {
	s, err := store.Load(repoTop)
	if err != nil {
		return store.State{}, err
	}
	if s.Cursor < 0 || s.Cursor >= len(s.Checkpoints)-1 {
		return s, errors.New("no forward checkpoint")
	}
	next := s.Checkpoints[s.Cursor+1]
	stashes, err := applyCheckpoint(repoTop, next, stash)
	if err != nil {
		return s, err
	}
	if len(stashes) > 0 {
		next.PrevStashRefs = append(next.PrevStashRefs, stashes...)
		s.Checkpoints[s.Cursor+1] = next
	}
	s.Cursor++
	return s, store.Save(repoTop, s)
}

func Jump(repoTop string, arg string, stash bool) (store.State, error) {
	s, err := store.Load(repoTop)
	if err != nil {
		return store.State{}, err
	}
	if len(s.Checkpoints) == 0 {
		return s, errors.New("no checkpoints")
	}
	idx := -1
	if i, err := strconv.Atoi(arg); err == nil {
		if i < 0 {
			i = len(s.Checkpoints) + i
		}
		if i >= 0 && i < len(s.Checkpoints) {
			idx = i
		}
	}
	if idx < 0 {
		argLower := strings.ToLower(strings.TrimSpace(arg))
		for i := range s.Checkpoints {
			l := strings.ToLower(s.Checkpoints[i].Label)
			if l != "" && l == argLower {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		idLower := strings.ToLower(arg)
		for i := range s.Checkpoints {
			if strings.HasPrefix(strings.ToLower(s.Checkpoints[i].ID), idLower) {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		return s, fmt.Errorf("no checkpoint matches %q", arg)
	}
	cp := s.Checkpoints[idx]
	stashes, err := applyCheckpoint(repoTop, cp, stash)
	if err != nil {
		return s, err
	}
	if len(stashes) > 0 {
		cp.PrevStashRefs = append(cp.PrevStashRefs, stashes...)
		s.Checkpoints[idx] = cp
	}
	s.Cursor = idx
	return s, store.Save(repoTop, s)
}

func CurrentRepo(startDir string) (string, error) {
	return gitexec.RepoTopLevel(startDir)
}
