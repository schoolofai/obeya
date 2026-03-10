# Shared Board — User-Level Project Init

**Goal:** Allow users to share a single Obeya board across multiple projects by initializing at the user level (~/.obeya/) instead of per-project.

**Architecture:** Add `ob init --shared` flag that creates a board in `~/.obeya/` and a `ob link` command to connect project directories to the shared board.

---

### Step 1: Add --shared flag to ob init

Modify `cmd/init.go` to accept `--shared`. When set, create `.obeya/board.json` in `~/.obeya/` instead of the current directory. Store the shared board path in a config file.

### Step 2: Add ob link command

Create `cmd/link.go` with a new `ob link` subcommand. This writes a `.obeya-link` file in the current project directory pointing to the shared board path. All subsequent `ob` commands in that directory should resolve through the link.

### Step 3: Update board discovery to check for links

Modify `FindProjectRoot()` in `internal/project/root.go` to check for `.obeya-link` files. If found, follow the link to the shared board instead of walking up directories.

### Step 4: Add ob unlink command

Create `cmd/unlink.go` to remove the `.obeya-link` file from the current directory, disconnecting it from the shared board.
