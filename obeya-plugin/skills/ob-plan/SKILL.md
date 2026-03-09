---
description: Manage Obeya plan documents — import, link, show
disable-model-invocation: false
user-invocable: true
---

# /ob:plan — Plan Management

Run `ob plan list --format json` to see all plans.

If `$ARGUMENTS` is provided:
- If it starts with "import", run `ob plan import $ARGUMENTS`
- If it starts with "show", run `ob plan show $ARGUMENTS --format json`
- If it starts with "link", run `ob plan link $ARGUMENTS`
- Otherwise, show the plan list

Format the output as a readable summary showing plan title, linked item count, and source file.
