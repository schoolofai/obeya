# Obeya Product Backlog — Prioritized Stories with ROI

**Date**: 2026-03-15
**Thesis**: Obeya is the task management system that AI agents actually want to use. Every story below is prioritized by ROI — the ratio of competitive differentiation and revenue potential to implementation effort.

---

## Priority 1 — Highest ROI (Build Now)

### Story 1: Agent Attribution Metrics
**Priority**: Critical
**Effort**: Small (2-3 days)
**ROI**: ★★★★★

**What it does**:
Slice all existing metrics (cycle time, lead time, throughput, dwell time) by identity type — human vs agent. Surface a new `ob metrics --breakdown` view showing:
- Tasks completed by agents vs humans (count + %)
- Cycle time comparison (agent vs human)
- Human review time on agent-created work
- Automation ratio (% of work done by agents)

**Why it matters**:
This is the single metric no competitor can provide. Jira, Linear, ClickUp all track "team velocity" — they cannot distinguish who did the work. Every engineering leader adopting AI agents is asking: "How much is AI actually doing for us?" Obeya answers this out of the box.

**ROI justification**:
- Near-zero implementation cost (data already captured in `ChangeRecord` with `UserID` → `Identity.Type`)
- Immediate differentiation — no competitor has this
- Direct sales enablement: "See your agent ROI in 5 minutes"
- Press/content angle: "The first task manager that measures AI contribution"

**Acceptance criteria**:
- `ob metrics` shows agent vs human breakdown by default
- `ob metrics --format json` includes `agent_stats` and `human_stats` sections
- TUI dashboard shows agent/human split in WIP and velocity panels
- Works with zero config — auto-detects from existing identity data

---

### Story 2: MCP Server
**Priority**: Critical
**Effort**: Medium (5-7 days)
**ROI**: ★★★★★

**What it does**:
Expose Obeya as a Model Context Protocol (MCP) server so any AI agent framework — Claude Code, Cursor, Windsurf, custom agents — can discover and use Obeya without a plugin. MCP tools:
- `list_tasks` — list/filter board items
- `create_task` — create with type, title, description, assignee
- `move_task` — change status
- `get_task` — show details
- `get_metrics` — board analytics
- `pick_task` — claim next unassigned task
- `complete_task` — mark done with notes

**Why it matters**:
MCP is becoming the standard protocol for AI agent tool discovery. Right now Obeya only works with Claude Code via a custom plugin. MCP makes Obeya work with every agent framework simultaneously. It's the difference between being a Claude-only plugin and being the universal task backend for agentic coding.

**ROI justification**:
- Expands addressable market from "Claude Code users" to "all AI agent users"
- Protocol-level integration is stickier than plugin-level
- Each MCP host (Cursor, Windsurf, custom agents) becomes a new distribution channel at zero marginal cost
- Anthropic actively promotes MCP adoption — riding a wave

**Acceptance criteria**:
- `ob mcp serve` starts an MCP server (stdio transport)
- All core operations available as MCP tools with JSON schema
- Tool descriptions are agent-optimized (clear, actionable, with examples)
- Works with Claude Code MCP config, Cursor, and generic MCP clients
- Error responses include actionable next steps

---

### Story 3: GitHub Integration (Auto-Link PRs and Commits)
**Priority**: High
**Effort**: Medium (4-5 days)
**ROI**: ★★★★☆

**What it does**:
Automatically link GitHub PRs and commits to Obeya tasks:
- `ob link-pr <task-id>` — associate current branch/PR with a task
- Auto-detect: when a commit message contains `ob#123`, link it
- Auto-move: when a PR is merged, move linked task to `review` or `done`
- `ob show <id>` displays linked PRs and commit count
- `ob metrics` includes "PR merge time" and "commits per task"

**Why it matters**:
Jira's #1 advantage is deep SDLC integration (Bitbucket, GitHub, CI/CD). Obeya needs this to be taken seriously by teams. The difference: Obeya's integration is CLI-native and agent-friendly, not a web UI bolt-on.

**ROI justification**:
- Removes the biggest objection: "But Jira connects to our PRs"
- Enables new metrics: code-to-task correlation, PR throughput
- Agents already create PRs — auto-linking means zero-friction tracking
- `gh` CLI already available in most dev environments

**Acceptance criteria**:
- `ob github link <task-id>` links current branch
- Commit messages with `ob#<num>` auto-detected on `ob metrics`
- `ob show` displays PR status (open/merged/closed)
- Works via `gh` CLI — no OAuth/token setup required for basic use

---

## Priority 2 — High ROI (Build Next)

### Story 4: Multi-Agent Coordination (Task Claiming & Locking)
**Priority**: High
**Effort**: Medium (4-5 days)
**ROI**: ★★★★☆

**What it does**:
Prevent multiple agents from working the same task simultaneously:
- `ob claim <id> --as agent-1` — atomic claim (fails if already claimed)
- `ob release <id>` — release claim
- `ob list --available` — show unclaimed, unblocked tasks
- Automatic stale claim expiry (configurable timeout)
- `ob handoff <id> --to agent-2 --context "notes"` — structured agent-to-agent handoff

**Why it matters**:
Multi-agent workflows are the next frontier (see: Claude Code parallel agents, AutoGen, CrewAI). When 3 agents work a board simultaneously, they need coordination primitives. No task manager provides this. Obeya becomes the coordination layer for multi-agent systems.

**ROI justification**:
- Enables a use case no competitor supports
- Direct enabler for enterprise multi-agent adoption
- Low complexity (extends existing `assign` + `block` mechanics)
- "Multi-agent Kanban" is a compelling product category to own

**Acceptance criteria**:
- `ob claim` is atomic (file lock or CAS)
- Claimed tasks show lock owner and timestamp
- `ob list --available` filters out claimed + blocked items
- Stale claims auto-expire after configurable duration
- `ob handoff` records context in task history

---

### Story 5: Predictive Analytics (Monte Carlo Forecasting)
**Priority**: High
**Effort**: Medium (3-4 days)
**ROI**: ★★★★☆

**What it does**:
Use historical throughput data to forecast completion dates:
- `ob forecast <epic-id>` — "Epic X will complete between March 28 and April 5 (85% confidence)"
- `ob forecast --board` — "All in-progress work completes by April 12 (50th percentile)"
- Monte Carlo simulation using actual daily throughput distribution
- Separate forecasts for agent-driven vs human-driven work
- TUI dashboard shows forecast range on burndown chart

**Why it matters**:
Traditional velocity is meaningless in agentic workflows — agent throughput is bursty and non-linear. Monte Carlo uses actual distribution, not averages, giving realistic ranges. When you can forecast agent + human work separately, you can answer: "If we add another agent, when does this ship?"

**ROI justification**:
- Direct competitor to Jira's "Release Burndown" but with agent awareness
- Enables data-driven resourcing decisions (add agents vs humans)
- Small implementation (statistical sampling over existing throughput data)
- High perceived value by engineering managers

**Acceptance criteria**:
- `ob forecast <epic-id>` shows P50, P85, P95 completion dates
- Uses last 30 days of throughput data for simulation
- `--format json` for programmatic consumption
- TUI burndown panel shows forecast cone
- Distinguishes agent vs human contribution to forecast

---

### Story 6: Review Overhead Metrics
**Priority**: High
**Effort**: Small (2-3 days)
**ROI**: ★★★★☆

**What it does**:
Measure the human cost of supervising agent work:
- Time tasks spend in `review` column when created by agents vs humans
- Rejection rate: tasks moved backward (e.g., review → in-progress) by agent vs human origin
- Rework cycles: how many times a task bounces between columns
- `ob metrics --review` shows review overhead dashboard

**Why it matters**:
DORA 2025 found 91% increase in code review time with AI adoption. This is the hidden cost of agentic coding. Teams need to see it to manage it. No tool surfaces this today.

**ROI justification**:
- Directly addresses the #1 pain point of AI adoption (review bottleneck)
- Data already exists in history — just needs new analysis
- High media/content value: "Are your agents creating more work than they save?"
- Enables optimization: identify which agent configurations produce least rework

**Acceptance criteria**:
- Review dwell time split by originator type (agent/human)
- Rejection rate (backward moves) by originator
- Rework cycle count per task
- `ob metrics --review --format json` for dashboards

---

## Priority 3 — Medium ROI (Build After Core)

### Story 7: Cumulative Flow Diagram (CFD)
**Priority**: Medium
**Effort**: Small (2-3 days)
**ROI**: ★★★☆☆

**What it does**:
Track and visualize how items accumulate across columns over time:
- `ob metrics --cfd` renders a stacked area chart in TUI
- Shows WIP trends: is work piling up in review? Is backlog growing?
- Historical data stored as daily snapshots
- Identifies bottleneck columns automatically

**Why it matters**:
CFD is the most powerful Kanban metric for identifying systemic bottlenecks. It shows at a glance whether your process is flowing or clogged. Combined with agent attribution, it answers: "Are agents flooding review faster than humans can clear it?"

**ROI justification**:
- Standard Kanban metric that teams expect from any serious tool
- Table stakes for enterprise adoption
- Small implementation (daily snapshot + area chart rendering)
- Extends existing TUI dashboard naturally

**Acceptance criteria**:
- Daily column count snapshots stored in board data
- TUI dashboard panel renders CFD
- `ob metrics --cfd --format json` exports data
- Auto-highlights bottleneck columns

---

### Story 8: CI/CD Integration (Auto-Move on Deploy)
**Priority**: Medium
**Effort**: Medium (3-4 days)
**ROI**: ★★★☆☆

**What it does**:
Integrate with CI/CD pipelines to auto-update task status:
- `ob webhook` endpoint for GitHub Actions / GitLab CI callbacks
- Auto-move tasks when: tests pass → review, deploy succeeds → done
- `ob ci report <task-id> --status pass|fail` for pipeline integration
- Failed deploys auto-block related tasks

**Why it matters**:
Closes the loop between code and task status. Agents create tasks, write code, push PRs — CI/CD verifies — task auto-completes. Full autonomous cycle with no human intervention needed for the happy path.

**ROI justification**:
- Enables fully autonomous agent → code → deploy → done workflow
- Reduces manual status updates (the #1 complaint about all task managers)
- GitHub Actions integration covers majority of users
- Reinforces "agent-first" positioning

**Acceptance criteria**:
- GitHub Actions workflow snippet provided in docs
- `ob ci report` updates task status based on pipeline result
- Failed CI auto-blocks task with failure details
- Works headlessly (no TUI/interactive required)

---

### Story 9: Team Dashboards (Multi-User Analytics)
**Priority**: Medium
**Effort**: Medium (4-5 days)
**ROI**: ★★★☆☆

**What it does**:
Per-user and per-agent analytics for shared boards:
- `ob metrics --by-user` shows individual throughput, cycle time, active tasks
- Agent leaderboard: which agent configuration is most effective?
- Workload distribution: is work evenly spread or concentrated?
- TUI view with per-user swimlanes

**Why it matters**:
Managers adopting agentic workflows need to understand team dynamics. Who's overloaded? Which agent is most productive? Is one human reviewing all agent work? These questions are unanswerable with current tools.

**ROI justification**:
- Prerequisite for enterprise/team adoption
- Unique angle: compare agent and human contributors side by side
- Enables resource optimization decisions
- Builds on existing metrics + user model

**Acceptance criteria**:
- `ob metrics --by-user` shows per-identity breakdown
- Includes agents alongside humans
- Workload heatmap in TUI
- `--format json` for external dashboard integration

---

### Story 10: Webhook & Event System
**Priority**: Medium
**Effort**: Medium (3-4 days)
**ROI**: ★★★☆☆

**What it does**:
Emit events when board state changes, enabling external integrations:
- `ob watch` streams events to stdout (JSON lines)
- Event types: task.created, task.moved, task.assigned, task.blocked, task.completed
- Filterable: `ob watch --type task.completed --assignee agent-1`
- Enables custom agent orchestrators to react to board changes

**Why it matters**:
Agent orchestrators (LangGraph, AutoGen, custom) need to react to task state changes. Webhooks make Obeya the event bus for multi-agent workflows. An orchestrator can watch for task.completed events and dispatch the next task automatically.

**ROI justification**:
- Enables ecosystem integrations without custom code per integration
- Makes Obeya the coordination hub, not just a task list
- Low implementation cost (extends existing file watcher)
- Foundation for Slack/Discord/email notifications later

**Acceptance criteria**:
- `ob watch` streams JSON-line events to stdout
- All board mutations emit events
- Filter by event type, user, status
- Clean shutdown on SIGINT

---

## Priority 4 — Lower ROI (Build for Completeness)

### Story 11: Import from Jira/Linear/GitHub Issues
**Priority**: Low
**Effort**: Medium (4-5 days)
**ROI**: ★★☆☆☆

**What it does**:
One-click migration from incumbent tools:
- `ob import jira --project KEY` — imports issues, epics, sprints
- `ob import linear --team slug` — imports issues and cycles
- `ob import github --repo owner/repo` — imports issues and labels
- Maps statuses to Obeya columns
- Preserves hierarchy (epic → story → task)

**Why it matters**:
Reduces switching friction. Teams won't adopt Obeya if migration is manual. "Try Obeya with your existing data in 60 seconds" is a powerful onboarding hook.

**ROI justification**:
- Reduces #1 adoption barrier (migration cost)
- One-time use per team, but critical for conversion
- Can be built incrementally (GitHub first, then Linear, then Jira)
- Marketing value: "Replace Jira in 60 seconds"

**Acceptance criteria**:
- At minimum `ob import github --repo` works end-to-end
- Status mapping is configurable
- History is preserved where possible
- Dry-run mode shows what would be imported

---

### Story 12: Custom Column Workflows & Templates
**Priority**: Low
**Effort**: Small (2-3 days)
**ROI**: ★★☆☆☆

**What it does**:
Pre-built board templates for common workflows:
- `ob init --template scrum` — backlog, sprint, in-progress, review, done
- `ob init --template kanban` — (current default)
- `ob init --template support` — triage, investigating, waiting, resolved
- Custom column definitions with WIP limits and transition rules
- Column transition constraints (e.g., can't skip review)

**Why it matters**:
Different teams have different workflows. Support teams, marketing teams, and engineering teams all use different column structures. Templates reduce setup time and show Obeya works beyond engineering.

**ROI justification**:
- Low effort, broadens addressable market
- Templates are content, not code
- Each template is a landing page / SEO opportunity
- Enables non-engineering use cases

**Acceptance criteria**:
- 3+ built-in templates
- `ob init --template <name>` works
- Custom templates from YAML/JSON files
- Templates define columns, WIP limits, and optional transition rules

---

## Summary: ROI-Ranked Backlog

| # | Story | Effort | ROI | Why Now |
|---|---|---|---|---|
| 1 | Agent Attribution Metrics | S | ★★★★★ | Zero-cost differentiation, data already exists |
| 2 | MCP Server | M | ★★★★★ | 10x distribution, protocol moat |
| 3 | GitHub Integration | M | ★★★★☆ | Removes #1 objection vs Jira |
| 4 | Multi-Agent Coordination | M | ★★★★☆ | Owns unclaimed category |
| 5 | Predictive Analytics | M | ★★★★☆ | High perceived value, small effort |
| 6 | Review Overhead Metrics | S | ★★★★☆ | Addresses top AI adoption pain |
| 7 | Cumulative Flow Diagram | S | ★★★☆☆ | Table stakes for Kanban |
| 8 | CI/CD Integration | M | ★★★☆☆ | Closes autonomous loop |
| 9 | Team Dashboards | M | ★★★☆☆ | Enterprise prerequisite |
| 10 | Webhook & Event System | M | ★★★☆☆ | Multi-agent orchestration |
| 11 | Import from Jira/Linear/GH | M | ★★☆☆☆ | Migration friction reducer |
| 12 | Custom Workflow Templates | S | ★★☆☆☆ | Broadens market |

**S** = Small (2-3 days) · **M** = Medium (3-7 days)

---

## Execution Order

**Sprint 1** (Week 1): Stories 1 + 6 — Agent attribution + review overhead metrics. Small effort, massive differentiation. Ship together as "Agentic Metrics" release.

**Sprint 2** (Weeks 2-3): Story 2 — MCP Server. Opens every agent framework as a distribution channel.

**Sprint 3** (Weeks 3-4): Stories 3 + 4 — GitHub integration + multi-agent coordination. Makes Obeya viable for real teams.

**Sprint 4** (Week 5): Story 5 + 7 — Forecasting + CFD. Analytics suite becomes best-in-class.

**Sprint 5+**: Stories 8-12 based on user feedback.
