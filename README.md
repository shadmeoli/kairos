# kairos

Git checkpoint timeline. See `kairos --help` for all commands.

## TUI (`kairos list`)

Run from inside a git working tree:

```text
kairos list
```

By default this opens an interactive view (Bubble Tea). `kairos list --plain` prints the same timeline to stdout with no TUI.

### What you see

- Checkpoints are listed **newest at the top**. The top row uses a `*` prefix; older rows use `│`.
- Each line shows index, branch name (or `detached`), short commit hash, and an optional label in parentheses.
- **`← timeline cursor`** marks the checkpoint index used for `back` / `forward` in the saved timeline (not necessarily the row the keyboard cursor is on).
- **`[HEAD]`** marks the row that matches the repo’s current `HEAD`.
- The **keyboard highlight** (different color) is the row your cursor is on; use this to choose where to jump.

### Keys

| Key | Action |
|-----|--------|
| `j` or Down | Move cursor one row toward newer |
| `k` or Up | Move cursor one row toward older |
| Enter | Quit the TUI and **jump** git state to the highlighted checkpoint (same as `kairos jump <index>`) |
| `q` or Ctrl+C | Quit without jumping |

If there are no checkpoints yet, the TUI only shows a short message suggesting `kairos save`.

### Jumping from the TUI and dirty trees

After you press Enter, kairos runs a normal `jump` to that checkpoint. If your working tree is dirty, that step fails unless you started the TUI with **`kairos list --stash`**, which allows an automatic `git stash push` before checkout (same rules as `kairos jump --stash`).

### Requirements

Use a real terminal (not a minimal log buffer). Color and cursor behavior depend on the terminal and on lipgloss/Bubble Tea defaults.
