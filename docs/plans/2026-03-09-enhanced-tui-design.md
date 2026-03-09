# Enhanced TUI Design — Trello-style Board with Quick Actions

## Overview

Rebuild the minimal `ob tui` into a full interactive Kanban board with Trello-style columns, color-coded cards, epic grouping with collapse/expand, detail overlays, picker modals, and inline quick actions.

## Architecture

Component-based Bubble Tea TUI:

```
internal/tui/
├── app.go        # Root model — state machine, routes keys
├── board.go      # Column board view, card rendering, epic grouping
├── detail.go     # Item detail overlay panel
├── picker.go     # Reusable list picker (columns, users, items)
├── input.go      # Text input modal (create, search)
├── styles.go     # Lipgloss styles — colors, borders, cards
└── keys.go       # Key binding definitions
```

## State Machine

```
BOARD (default)
  ├── Enter → DETAIL (Esc returns)
  │            ├── m → COLUMN_PICKER
  │            ├── a → USER_PICKER
  │            └── p → immediate priority cycle
  ├── m → COLUMN_PICKER (Enter selects, Esc cancels)
  ├── a → USER_PICKER (Enter selects, Esc cancels)
  ├── c → TYPE_PICKER → TEXT_INPUT (Enter creates, Esc cancels)
  ├── d → CONFIRM (y/n)
  ├── b → ITEM_PICKER (Enter blocks, Esc cancels)
  ├── / → TEXT_INPUT (filters board live)
  ├── Space → toggle epic collapse (immediate)
  └── q/Ctrl+C → quit
```

## Key Bindings

| Key | Action |
|---|---|
| h/l ←/→ | Move between columns |
| j/k ↑/↓ | Move between items |
| Tab | Jump to next column |
| Enter | Show detail overlay |
| m | Move → column picker |
| a | Assign → user picker |
| c | Create → type picker → title input |
| d | Delete (confirm y/n) |
| b | Block/unblock → item picker |
| p | Cycle priority |
| Space | Collapse/expand epic |
| / | Search/filter |
| r | Reload board |
| ? | Help |
| q Ctrl+C | Quit |

## Visual Design

### Card Format

```
┌────────────────────┐
│ #3 Design Form     │
│ task ● med         │
│ ↑ #1 Auth System   │
│ @claude        [!] │
└────────────────────┘
```

### Color Scheme

| Element | Color |
|---|---|
| Selected card border | Cyan |
| Epic text | Purple |
| Story text | Blue |
| Task text | White |
| Priority high/critical | Red |
| Priority medium | Yellow |
| Priority low | Green |
| Blocked [!] | Red bold |
| Active column header | Bright white underlined |
| Inactive column header | Dim white |
| Assignee | Dim cyan |
| Help bar | Dim |

### Hierarchy — Epic Grouping

- Items grouped by epic within each column
- Space toggles collapse/expand
- Collapsed: `▶ #1 Auth System (3 items)`
- Expanded: `▼ #1 Auth System` with children below
- Items in different column than epic: parent badge `↑ #1 Epic Name`
- Orphan items (no parent): ungrouped at column bottom

### Detail Overlay

```
╭─ #3 Design Form ──────────────────────────────╮
│ Type:     task          Priority: ●● medium    │
│ Status:   in-progress   Assignee: @claude      │
│ Parent:   #2 Login Flow                        │
│ Tags:     frontend                             │
│ Blocked:  —                                    │
│ Children: none                                 │
│ History:                                       │
│   15:54  created task: Design Form             │
│   15:54  moved: backlog → in-progress          │
│ [m]ove  [a]ssign  [p]riority  [Esc]close       │
╰────────────────────────────────────────────────╯
```

### Picker Modal

```
╭─ Move #3 to: ─────────╮
│   backlog              │
│   todo                 │
│ ▸ review               │
│   done                 │
│ Enter:select Esc:cancel│
╰────────────────────────╯
```

## Stories

1. Refactor TUI architecture into components
2. Trello-style column board with styled cards
3. Epic grouping with collapse/expand
4. Detail overlay panel
5. Picker and input modals
6. Quick action key bindings
7. Search/filter functionality
