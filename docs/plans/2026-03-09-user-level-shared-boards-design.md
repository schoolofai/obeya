# User-Level Shared Boards — Design

**Goal:** Allow users to share a single Obeya board across multiple projects by initializing named boards at `~/.obeya/boards/<name>/` and linking projects to them.

**Use cases:** Cross-project visibility, mono-workflow across repos, agent coordination across Claude Code sessions.

**Approach:** Hybrid — link file in project for fast discovery, board-side project registry for cross-project queries.

---

## Directory Structure

```
~/.obeya/
└── boards/
    └── <board-name>/
        ├── board.json      # Same schema as local boards + projects map
        └── board.lock      # File-based locking (same as local)
```

**Project-side:**
```
<git-root>/
└── .obeya-link             # Plain text, single line: board name
```

## Data Model Changes

### Board — new `projects` field

```go
type LinkedProject struct {
    Name       string `json:"name"`        // e.g. "api-server"
    LocalPath  string `json:"local_path"`  // e.g. "/Users/niladri/code/api-server"
    GitRemote  string `json:"git_remote"`  // e.g. "git@github.com:niladri/api-server.git"
    LinkedAt   string `json:"linked_at"`   // ISO timestamp
}

// Board.Projects: map[string]LinkedProject (keyed by project name)
```

### Item — new `project` field

```go
// Item.Project: string (matches LinkedProject.Name, empty for untagged)
```

Project is set automatically on task creation based on which linked project the command runs from.

## Commands

### `ob init --shared <board-name>`

- Creates `~/.obeya/boards/<board-name>/board.json` with default columns
- Errors if board already exists: "Board '<name>' already exists. Use `ob link <name>` to connect this project."
- Does NOT auto-link the current project

### `ob link <board-name>`

- Validates board exists at `~/.obeya/boards/<board-name>/`
- Resolves current project: git root, git remote origin, directory name
- Writes `.obeya-link` at git root containing the board name
- Registers project in board's `projects` map
- Errors if already linked: "This project is already linked to board '<name>'."
- Errors if board doesn't exist: "Board '<name>' not found. Run `ob init --shared <name>` first."
- If local board exists with tasks: prompts "This project has N tasks on a local board. Migrate them to '<name>'? [y/n]"
  - If yes: copies all items to global board, tags each with project name, remaps display numbers, renames `.obeya/` to `.obeya-local-backup/`
  - If no: errors out, no partial state
  - `--migrate` flag skips the prompt (for scripting/agents)
- If local board exists but is empty: migrates silently (nothing to lose)

### `ob unlink`

- Reads `.obeya-link` to find board name
- Removes project from board's `projects` map
- Deletes `.obeya-link` file
- Errors if not linked: "This project is not linked to any shared board."

### `ob boards`

- Lists all boards under `~/.obeya/boards/` with linked project count
- Example: `client-work   3 projects`

### `ob board prune <name>`

- Removes dead project entries from board's `projects` map (checks if `local_path` exists)

## Discovery Logic

Updated `FindProjectRoot` — three-pass priority:

```
Pass 1: Walk up looking for .obeya-link
  → read board name, resolve to ~/.obeya/boards/<name>/
  → validate board.json exists (hard error if stale, no fallback)

Pass 2: Walk up looking for .obeya/board.json (local board, existing behavior)

Pass 3: Walk up looking for .git (existing behavior, for ob init)

None found: error
```

**Key rules:**
- `.obeya-link` wins over local `.obeya/` — linked board is authoritative
- No silent fallback. Stale link = hard error with guidance to `ob unlink`
- A project has either `.obeya/` or `.obeya-link`, not both

## Edge Cases & Error Handling

**Stale links:**
- Board deleted but `.obeya-link` exists → hard error: "Linked board 'X' not found at ~/.obeya/boards/X/. Run `ob unlink` to remove the stale link."
- Project deleted but in board's `projects` map → `ob board prune <name>` cleans dead entries

**Name collisions:**
- Two projects with same directory name on same board → fall back to `<remote-org>/<repo-name>` as project name
- Project with no git remote → use directory name, `git_remote` is empty string

**Migration (on link):**
- `ob link` detects local `.obeya/board.json` and offers to migrate tasks to the global board
- Migration copies items, re-tags with project name, remaps display numbers to avoid collisions
- Local board renamed to `.obeya-local-backup/` (not deleted) for safety
- `ob unlink` does NOT reverse migration — tasks stay on the global board
- No ongoing sync — migration is a one-time operation

**Concurrency:**
- Same `board.lock` file-based locking. Multiple agents across projects serialized by lock.
