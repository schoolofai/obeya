---
description: Mark an agent-completed item as reviewed or hidden. Use when a human has reviewed agent work and wants to approve it or hide it from the review queue. Only humans can mark items as reviewed.
disable-model-invocation: false
user-invocable: true
---

# Review Agent Work

Mark an agent-completed item as reviewed (approved) or hidden (dismissed from review queue).

## Steps

1. If `$ARGUMENTS` is provided, use it as the item ID
2. Otherwise, run `ob list --status done --format json` and filter to items with `review_context` set
3. Show the items pending review with their confidence scores
4. Ask which item to review and what status to set

## Usage

```bash
# Mark as reviewed (approved)
ob review <id> --status reviewed

# Mark as hidden (dismiss from review queue)
ob review <id> --status hidden
```

## TUI Shortcuts

In the TUI review queue column:
- **R** — Mark the selected item as reviewed
- **x** — Hide the selected item from the review queue
- **V** — Expand review context accordion on the selected card
- **P** — Open the Past Reviews pane

## Important

- Only human identities can review items. Agents cannot mark items as reviewed.
- Reviewed items remain in the "done" column but get a green border in the TUI.
- Hidden items are removed from the review queue but remain in "done".
- Use **P** in the TUI to see a hierarchical tree of all past reviewed items.
