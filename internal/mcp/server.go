package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/metrics"
)

// Server wraps the MCP server and the Obeya engine.
type Server struct {
	mcp    *mcpserver.MCPServer
	engine *engine.Engine
	userID string // resolved identity for audit trail
}

// New creates an MCP server wired to the given engine.
func New(eng *engine.Engine) *Server {
	s := &Server{engine: eng}
	s.resolveIdentity()

	s.mcp = mcpserver.NewMCPServer(
		"obeya",
		"1.0.0",
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(true, false),
		mcpserver.WithPromptCapabilities(true),
	)

	s.registerTools()
	s.registerResources()
	s.registerPrompts()

	return s
}

// MCPServer returns the underlying MCPServer for transport binding.
func (s *Server) MCPServer() *mcpserver.MCPServer {
	return s.mcp
}

// resolveIdentity finds or creates the agent identity for audit trail.
func (s *Server) resolveIdentity() {
	board, err := s.engine.ListBoard()
	if err != nil {
		return
	}

	// Check env vars
	if name := os.Getenv("OBEYA_USER"); name != "" {
		if id, err := board.ResolveUserID(name); err == nil {
			s.userID = id
			return
		}
	}
	if name := os.Getenv("OBEYA_AGENT_NAME"); name != "" {
		if id, err := board.ResolveUserID(name); err == nil {
			s.userID = id
			return
		}
	}

	// Fall back to first agent, then first user
	for _, u := range board.Users {
		if u.Type == domain.IdentityAgent {
			s.userID = u.ID
			return
		}
	}
	for _, u := range board.Users {
		s.userID = u.ID
		return
	}
}

// sessionID returns a session identifier for history records.
func (s *Server) sessionID() string {
	if sid := os.Getenv("OB_SESSION"); sid != "" {
		return sid
	}
	return fmt.Sprintf("mcp-%d", os.Getpid())
}

func (s *Server) registerTools() {
	s.registerItemTools()
	s.registerUserTools()
	s.registerPlanTools()
	s.registerBoardTools()
	s.registerMetricsTools()
	s.registerWorkflowTools()
}

// --- Item Tools ---

func (s *Server) registerItemTools() {
	s.mcp.AddTool(mcplib.NewTool("list_items",
		mcplib.WithDescription("List items on the Obeya board with optional filters. Returns all items matching the criteria."),
		mcplib.WithString("status", mcplib.Description("Filter by column status (e.g., 'backlog', 'in-progress', 'done')")),
		mcplib.WithString("assignee", mcplib.Description("Filter by assignee name or ID")),
		mcplib.WithString("type", mcplib.Description("Filter by item type"), mcplib.Enum("epic", "story", "task")),
		mcplib.WithString("tag", mcplib.Description("Filter by tag")),
		mcplib.WithBoolean("blocked", mcplib.Description("If true, only show blocked items")),
	), s.handleListItems)

	s.mcp.AddTool(mcplib.NewTool("get_item",
		mcplib.WithDescription("Get detailed information about a specific board item including description, history, and children."),
		mcplib.WithString("ref", mcplib.Required(), mcplib.Description("Item reference — display number (e.g., '5') or UUID")),
	), s.handleGetItem)

	s.mcp.AddTool(mcplib.NewTool("create_item",
		mcplib.WithDescription("Create a new item (epic, story, or task) on the board. Items start in the first column."),
		mcplib.WithString("type", mcplib.Required(), mcplib.Description("Item type"), mcplib.Enum("epic", "story", "task")),
		mcplib.WithString("title", mcplib.Required(), mcplib.Description("Short, descriptive title")),
		mcplib.WithString("assignee", mcplib.Required(), mcplib.Description("Name or ID of the user/agent to assign to")),
		mcplib.WithString("description", mcplib.Description("Detailed description with context and acceptance criteria")),
		mcplib.WithString("priority", mcplib.Description("Priority level"), mcplib.Enum("low", "medium", "high", "critical")),
		mcplib.WithString("parent", mcplib.Description("Parent item reference for nesting")),
	), s.handleCreateItem)

	s.mcp.AddTool(mcplib.NewTool("move_item",
		mcplib.WithDescription("Move an item to a different column. Common flow: backlog → todo → in-progress → review → done."),
		mcplib.WithString("ref", mcplib.Required(), mcplib.Description("Item reference — display number or UUID")),
		mcplib.WithString("status", mcplib.Required(), mcplib.Description("Target column name (e.g., 'in-progress', 'done')")),
	), s.handleMoveItem)

	s.mcp.AddTool(mcplib.NewTool("edit_item",
		mcplib.WithDescription("Edit an item's title, description, or priority. Provide only fields to change."),
		mcplib.WithString("ref", mcplib.Required(), mcplib.Description("Item reference")),
		mcplib.WithString("title", mcplib.Description("New title")),
		mcplib.WithString("description", mcplib.Description("New description")),
		mcplib.WithString("priority", mcplib.Description("New priority"), mcplib.Enum("low", "medium", "high", "critical")),
	), s.handleEditItem)

	s.mcp.AddTool(mcplib.NewTool("delete_item",
		mcplib.WithDescription("Delete an item from the board. Cannot delete items with children."),
		mcplib.WithString("ref", mcplib.Required(), mcplib.Description("Item reference")),
	), s.handleDeleteItem)

	s.mcp.AddTool(mcplib.NewTool("assign_item",
		mcplib.WithDescription("Assign or reassign an item to a user or agent."),
		mcplib.WithString("ref", mcplib.Required(), mcplib.Description("Item reference")),
		mcplib.WithString("assignee", mcplib.Required(), mcplib.Description("Name or ID of the assignee")),
	), s.handleAssignItem)

	s.mcp.AddTool(mcplib.NewTool("block_item",
		mcplib.WithDescription("Mark an item as blocked by another item."),
		mcplib.WithString("ref", mcplib.Required(), mcplib.Description("Item to block")),
		mcplib.WithString("blocker", mcplib.Required(), mcplib.Description("Blocking item reference")),
	), s.handleBlockItem)

	s.mcp.AddTool(mcplib.NewTool("unblock_item",
		mcplib.WithDescription("Remove a blocker from an item."),
		mcplib.WithString("ref", mcplib.Required(), mcplib.Description("Blocked item")),
		mcplib.WithString("blocker", mcplib.Required(), mcplib.Description("Blocker to remove")),
	), s.handleUnblockItem)
}

// --- Item Handlers ---

func (s *Server) handleListItems(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	filter := engine.ListFilter{
		Status:   stringArg(req, "status"),
		Assignee: stringArg(req, "assignee"),
		Type:     stringArg(req, "type"),
		Tag:      stringArg(req, "tag"),
		Blocked:  boolArg(req, "blocked"),
	}

	items, err := s.engine.ListItems(filter)
	if err != nil {
		return toolError(err), nil
	}

	board, _ := s.engine.ListBoard()
	result := make([]itemSummary, 0, len(items))
	for _, item := range items {
		result = append(result, toItemSummary(item, board))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Num < result[j].Num })

	return toolJSON(result)
}

func (s *Server) handleGetItem(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	ref := stringArg(req, "ref")
	item, err := s.engine.GetItem(ref)
	if err != nil {
		return toolError(err), nil
	}

	children, _ := s.engine.GetChildren(item.ID)
	plans, _ := s.engine.PlansForItem(item.ID)
	board, _ := s.engine.ListBoard()

	detail := toItemDetail(item, children, plans, board)
	return toolJSON(detail)
}

func (s *Server) handleCreateItem(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	itemType := stringArg(req, "type")
	title := stringArg(req, "title")
	assignee := stringArg(req, "assignee")
	desc := stringArg(req, "description")
	priority := stringArg(req, "priority")
	parent := stringArg(req, "parent")

	item, err := s.engine.CreateItem(itemType, title, parent, desc, priority, assignee, nil)
	if err != nil {
		return toolError(err), nil
	}

	return toolJSON(map[string]interface{}{
		"created":     true,
		"display_num": item.DisplayNum,
		"id":          item.ID,
		"title":       item.Title,
		"type":        string(item.Type),
		"status":      item.Status,
		"assignee":    item.Assignee,
	})
}

func (s *Server) handleMoveItem(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	ref := stringArg(req, "ref")
	status := stringArg(req, "status")

	if err := s.engine.MoveItem(ref, status, s.userID, s.sessionID()); err != nil {
		return toolError(err), nil
	}

	return toolText(fmt.Sprintf("Moved #%s to %s", ref, status)), nil
}

func (s *Server) handleEditItem(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	ref := stringArg(req, "ref")
	title := stringArg(req, "title")
	desc := stringArg(req, "description")
	priority := stringArg(req, "priority")

	if err := s.engine.EditItem(ref, title, desc, priority, s.userID, s.sessionID()); err != nil {
		return toolError(err), nil
	}

	return toolText(fmt.Sprintf("Updated #%s", ref)), nil
}

func (s *Server) handleDeleteItem(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	ref := stringArg(req, "ref")
	if err := s.engine.DeleteItem(ref, s.userID, s.sessionID()); err != nil {
		return toolError(err), nil
	}
	return toolText(fmt.Sprintf("Deleted #%s", ref)), nil
}

func (s *Server) handleAssignItem(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	ref := stringArg(req, "ref")
	assignee := stringArg(req, "assignee")

	if err := s.engine.AssignItem(ref, assignee, s.userID, s.sessionID()); err != nil {
		return toolError(err), nil
	}
	return toolText(fmt.Sprintf("Assigned #%s to %s", ref, assignee)), nil
}

func (s *Server) handleBlockItem(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	ref := stringArg(req, "ref")
	blocker := stringArg(req, "blocker")

	if err := s.engine.BlockItem(ref, blocker, s.userID, s.sessionID()); err != nil {
		return toolError(err), nil
	}
	return toolText(fmt.Sprintf("Blocked #%s by #%s", ref, blocker)), nil
}

func (s *Server) handleUnblockItem(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	ref := stringArg(req, "ref")
	blocker := stringArg(req, "blocker")

	if err := s.engine.UnblockItem(ref, blocker, s.userID, s.sessionID()); err != nil {
		return toolError(err), nil
	}
	return toolText(fmt.Sprintf("Unblocked #%s from #%s", ref, blocker)), nil
}

// --- User Tools ---

func (s *Server) registerUserTools() {
	s.mcp.AddTool(mcplib.NewTool("list_users",
		mcplib.WithDescription("List all registered users and agents on the board."),
	), s.handleListUsers)

	s.mcp.AddTool(mcplib.NewTool("add_user",
		mcplib.WithDescription("Register a new user or agent identity on the board."),
		mcplib.WithString("name", mcplib.Required(), mcplib.Description("Display name")),
		mcplib.WithString("type", mcplib.Required(), mcplib.Description("Identity type"), mcplib.Enum("human", "agent")),
		mcplib.WithString("provider", mcplib.Description("Agent provider (e.g., 'claude-code', 'cursor')")),
	), s.handleAddUser)
}

func (s *Server) handleListUsers(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	board, err := s.engine.ListBoard()
	if err != nil {
		return toolError(err), nil
	}

	users := make([]map[string]string, 0, len(board.Users))
	for _, u := range board.Users {
		users = append(users, map[string]string{
			"id":       u.ID,
			"name":     u.Name,
			"type":     string(u.Type),
			"provider": u.Provider,
		})
	}
	return toolJSON(users)
}

func (s *Server) handleAddUser(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	name := stringArg(req, "name")
	userType := stringArg(req, "type")
	provider := stringArg(req, "provider")

	if err := s.engine.AddUser(name, userType, provider); err != nil {
		return toolError(err), nil
	}
	return toolText(fmt.Sprintf("Added %s user: %s", userType, name)), nil
}

// --- Plan Tools ---

func (s *Server) registerPlanTools() {
	s.mcp.AddTool(mcplib.NewTool("list_plans",
		mcplib.WithDescription("List all plans on the board."),
	), s.handleListPlans)

	s.mcp.AddTool(mcplib.NewTool("get_plan",
		mcplib.WithDescription("Get a plan's full content and linked items."),
		mcplib.WithString("ref", mcplib.Required(), mcplib.Description("Plan reference — display number or UUID")),
	), s.handleGetPlan)

	s.mcp.AddTool(mcplib.NewTool("create_plan",
		mcplib.WithDescription("Create a new plan document on the board."),
		mcplib.WithString("title", mcplib.Required(), mcplib.Description("Plan title")),
		mcplib.WithString("content", mcplib.Required(), mcplib.Description("Plan content in markdown")),
	), s.handleCreatePlan)

	s.mcp.AddTool(mcplib.NewTool("link_plan",
		mcplib.WithDescription("Link board items to a plan for traceability."),
		mcplib.WithString("plan_ref", mcplib.Required(), mcplib.Description("Plan reference")),
		mcplib.WithString("item_refs", mcplib.Required(), mcplib.Description("Comma-separated item references to link")),
	), s.handleLinkPlan)
}

func (s *Server) handleListPlans(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	plans, err := s.engine.ListPlans()
	if err != nil {
		return toolError(err), nil
	}

	result := make([]map[string]interface{}, 0, len(plans))
	for _, p := range plans {
		result = append(result, map[string]interface{}{
			"display_num":  p.DisplayNum,
			"title":        p.Title,
			"linked_items": len(p.LinkedItems),
		})
	}
	return toolJSON(result)
}

func (s *Server) handleGetPlan(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	ref := stringArg(req, "ref")
	plan, err := s.engine.ShowPlan(ref)
	if err != nil {
		return toolError(err), nil
	}
	return toolJSON(map[string]interface{}{
		"display_num":  plan.DisplayNum,
		"title":        plan.Title,
		"content":      plan.Content,
		"source_file":  plan.SourceFile,
		"linked_items": plan.LinkedItems,
		"created_at":   plan.CreatedAt.Format(time.RFC3339),
		"updated_at":   plan.UpdatedAt.Format(time.RFC3339),
	})
}

func (s *Server) handleCreatePlan(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	title := stringArg(req, "title")
	content := stringArg(req, "content")

	plan, err := s.engine.CreatePlan(title, content, "")
	if err != nil {
		return toolError(err), nil
	}
	return toolJSON(map[string]interface{}{
		"created":     true,
		"display_num": plan.DisplayNum,
		"title":       plan.Title,
	})
}

func (s *Server) handleLinkPlan(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	planRef := stringArg(req, "plan_ref")
	itemRefsStr := stringArg(req, "item_refs")
	itemRefs := strings.Split(itemRefsStr, ",")
	for i := range itemRefs {
		itemRefs[i] = strings.TrimSpace(itemRefs[i])
	}

	if err := s.engine.LinkPlan(planRef, itemRefs); err != nil {
		return toolError(err), nil
	}
	return toolText(fmt.Sprintf("Linked %d items to plan #%s", len(itemRefs), planRef)), nil
}

// --- Board Tools ---

func (s *Server) registerBoardTools() {
	s.mcp.AddTool(mcplib.NewTool("list_columns",
		mcplib.WithDescription("List all columns on the board with WIP limits and current item counts."),
	), s.handleListColumns)

	s.mcp.AddTool(mcplib.NewTool("add_column",
		mcplib.WithDescription("Add a new status column to the board."),
		mcplib.WithString("name", mcplib.Required(), mcplib.Description("Column name (e.g., 'staging', 'qa')")),
	), s.handleAddColumn)
}

func (s *Server) handleListColumns(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	board, err := s.engine.ListBoard()
	if err != nil {
		return toolError(err), nil
	}

	wip := metrics.WIPStatus(board)
	result := make([]map[string]interface{}, 0, len(wip))
	for _, w := range wip {
		result = append(result, map[string]interface{}{
			"name":  w.Name,
			"count": w.Count,
			"limit": w.Limit,
			"level": w.Level,
		})
	}
	// Include done column count
	doneCount := 0
	for _, item := range board.Items {
		if item.Status == "done" {
			doneCount++
		}
	}
	result = append(result, map[string]interface{}{
		"name":  "done",
		"count": doneCount,
		"limit": 0,
		"level": "ok",
	})

	return toolJSON(result)
}

func (s *Server) handleAddColumn(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	name := stringArg(req, "name")
	if err := s.engine.AddColumn(name); err != nil {
		return toolError(err), nil
	}
	return toolText(fmt.Sprintf("Added column: %s", name)), nil
}

// --- Metrics Tools ---

func (s *Server) registerMetricsTools() {
	s.mcp.AddTool(mcplib.NewTool("get_metrics",
		mcplib.WithDescription("Get board analytics: cycle time, lead time, throughput, WIP, dwell times, daily velocity."),
		mcplib.WithNumber("days", mcplib.Description("Number of days for velocity history (default: 14)")),
	), s.handleGetMetrics)

	s.mcp.AddTool(mcplib.NewTool("get_burndown",
		mcplib.WithDescription("Get burndown chart data for an epic showing actual vs ideal progress."),
		mcplib.WithString("epic_ref", mcplib.Required(), mcplib.Description("Epic reference — display number or UUID")),
	), s.handleGetBurndown)
}

func (s *Server) handleGetMetrics(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	days := intArg(req, "days")
	if days <= 0 {
		days = 14
	}

	board, err := s.engine.ListBoard()
	if err != nil {
		return toolError(err), nil
	}

	items := metrics.BoardItems(board)
	now := time.Now()
	m := metrics.Compute(items, now)
	wip := metrics.WIPStatus(board)
	velocity := metrics.DailyVelocity(items, days, now)

	result := map[string]interface{}{
		"total_items": m.TotalItems,
		"done_items":  m.DoneItems,
		"throughput": map[string]interface{}{
			"this_week": m.Throughput.ThisWeek,
			"last_week": m.Throughput.LastWeek,
			"total":     m.Throughput.Total,
			"per_week":  m.Throughput.PerWeek,
		},
	}

	if m.CycleTime != nil {
		result["cycle_time"] = m.CycleTime.Display
	}
	if m.LeadTime != nil {
		result["lead_time"] = m.LeadTime.Display
	}

	wipResult := make([]map[string]interface{}, 0, len(wip))
	for _, w := range wip {
		wipResult = append(wipResult, map[string]interface{}{
			"name": w.Name, "count": w.Count, "limit": w.Limit, "level": w.Level,
		})
	}
	result["wip"] = wipResult

	dwellResult := map[string]interface{}{}
	for col, d := range m.Dwell {
		dwellResult[col] = map[string]interface{}{
			"avg":   metrics.FormatDuration(d.Average),
			"count": d.Count,
		}
	}
	result["dwell"] = dwellResult

	velocityResult := make([]map[string]interface{}, 0, len(velocity))
	for _, v := range velocity {
		velocityResult = append(velocityResult, map[string]interface{}{
			"date":  v.Date.Format("2006-01-02"),
			"count": v.Count,
		})
	}
	result["daily_velocity"] = velocityResult

	return toolJSON(result)
}

func (s *Server) handleGetBurndown(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	epicRef := stringArg(req, "epic_ref")
	epic, err := s.engine.GetItem(epicRef)
	if err != nil {
		return toolError(err), nil
	}

	children, err := s.engine.GetChildren(epic.ID)
	if err != nil {
		return toolError(err), nil
	}

	points := metrics.EpicBurndown(epic, children, time.Now())
	result := make([]map[string]interface{}, 0, len(points))
	for _, p := range points {
		result = append(result, map[string]interface{}{
			"date":      p.Date.Format(time.RFC3339),
			"remaining": p.Remaining,
			"ideal":     p.Ideal,
		})
	}
	return toolJSON(result)
}

// --- Workflow Tools ---

func (s *Server) registerWorkflowTools() {
	s.mcp.AddTool(mcplib.NewTool("pick_next_task",
		mcplib.WithDescription("Pick the next available unassigned, unblocked task. Assigns it to you and moves to in-progress."),
		mcplib.WithString("from_column", mcplib.Description("Column to pick from (default: 'todo')")),
	), s.handlePickNextTask)

	s.mcp.AddTool(mcplib.NewTool("complete_task",
		mcplib.WithDescription("Mark a task as done, optionally adding completion notes."),
		mcplib.WithString("ref", mcplib.Required(), mcplib.Description("Item reference")),
		mcplib.WithString("notes", mcplib.Description("Optional completion notes")),
	), s.handleCompleteTask)

	s.mcp.AddTool(mcplib.NewTool("board_summary",
		mcplib.WithDescription("Get a concise board overview: items per column, blockers, key metrics. Ideal for standup context."),
	), s.handleBoardSummary)
}

func (s *Server) handlePickNextTask(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	column := stringArg(req, "from_column")
	if column == "" {
		column = "todo"
	}

	items, err := s.engine.ListItems(engine.ListFilter{Status: column})
	if err != nil {
		return toolError(err), nil
	}

	// Filter unassigned and unblocked
	var candidates []*domain.Item
	for _, item := range items {
		if item.Assignee == "" && len(item.BlockedBy) == 0 {
			candidates = append(candidates, item)
		}
	}

	if len(candidates) == 0 {
		return toolText(fmt.Sprintf("No available unassigned, unblocked tasks in '%s'", column)), nil
	}

	// Sort by priority (critical > high > medium > low)
	priorityOrder := map[domain.Priority]int{
		domain.PriorityCritical: 0,
		domain.PriorityHigh:     1,
		domain.PriorityMedium:   2,
		domain.PriorityLow:      3,
	}
	sort.Slice(candidates, func(i, j int) bool {
		return priorityOrder[candidates[i].Priority] < priorityOrder[candidates[j].Priority]
	})

	pick := candidates[0]

	// Assign to self and move to in-progress
	if s.userID != "" {
		_ = s.engine.AssignItem(fmt.Sprintf("%d", pick.DisplayNum), s.userID, s.userID, s.sessionID())
	}
	_ = s.engine.MoveItem(fmt.Sprintf("%d", pick.DisplayNum), "in-progress", s.userID, s.sessionID())

	board, _ := s.engine.ListBoard()
	return toolJSON(toItemSummary(pick, board))
}

func (s *Server) handleCompleteTask(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	ref := stringArg(req, "ref")
	notes := stringArg(req, "notes")

	if notes != "" {
		item, err := s.engine.GetItem(ref)
		if err != nil {
			return toolError(err), nil
		}
		newDesc := item.Description
		if newDesc != "" {
			newDesc += "\n\n---\n"
		}
		newDesc += notes
		_ = s.engine.EditItem(ref, "", newDesc, "", s.userID, s.sessionID())
	}

	if err := s.engine.MoveItem(ref, "done", s.userID, s.sessionID()); err != nil {
		return toolError(err), nil
	}
	return toolText(fmt.Sprintf("Completed #%s", ref)), nil
}

func (s *Server) handleBoardSummary(_ context.Context, _ mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	board, err := s.engine.ListBoard()
	if err != nil {
		return toolError(err), nil
	}

	items := metrics.BoardItems(board)
	now := time.Now()
	m := metrics.Compute(items, now)
	wip := metrics.WIPStatus(board)

	// Count per column
	columnCounts := map[string]int{}
	for _, item := range board.Items {
		columnCounts[item.Status]++
	}

	// Find blocked items
	var blockedItems []map[string]interface{}
	for _, item := range board.Items {
		if len(item.BlockedBy) > 0 {
			blockedItems = append(blockedItems, map[string]interface{}{
				"num":        item.DisplayNum,
				"title":      item.Title,
				"blocked_by": len(item.BlockedBy),
			})
		}
	}

	// WIP alerts
	var alerts []string
	for _, w := range wip {
		if w.Level == "over" {
			alerts = append(alerts, fmt.Sprintf("'%s' is over WIP limit (%d/%d)", w.Name, w.Count, w.Limit))
		}
	}

	summary := map[string]interface{}{
		"board_name":    board.Name,
		"total_items":   m.TotalItems,
		"done_items":    m.DoneItems,
		"columns":       columnCounts,
		"blocked_items": blockedItems,
		"alerts":        alerts,
		"throughput": map[string]interface{}{
			"this_week": m.Throughput.ThisWeek,
			"last_week": m.Throughput.LastWeek,
		},
	}
	if m.CycleTime != nil {
		summary["cycle_time"] = m.CycleTime.Display
	}

	return toolJSON(summary)
}

// --- Resources ---

func (s *Server) registerResources() {
	s.mcp.AddResource(mcplib.NewResource(
		"obeya://board/summary",
		"Board Summary",
		mcplib.WithMIMEType("application/json"),
	), s.handleBoardSummaryResource)

	s.mcp.AddResource(mcplib.NewResource(
		"obeya://board/items",
		"All Board Items",
		mcplib.WithMIMEType("application/json"),
	), s.handleBoardItemsResource)

	s.mcp.AddResource(mcplib.NewResource(
		"obeya://metrics",
		"Board Metrics",
		mcplib.WithMIMEType("application/json"),
	), s.handleMetricsResource)

	s.mcp.AddResource(mcplib.NewResource(
		"obeya://users",
		"Board Users",
		mcplib.WithMIMEType("application/json"),
	), s.handleUsersResource)

	s.mcp.AddResource(mcplib.NewResource(
		"obeya://plans",
		"Board Plans",
		mcplib.WithMIMEType("application/json"),
	), s.handlePlansResource)
}

func (s *Server) handleBoardSummaryResource(_ context.Context, _ mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	board, err := s.engine.ListBoard()
	if err != nil {
		return nil, err
	}

	items := metrics.BoardItems(board)
	m := metrics.Compute(items, time.Now())

	columnCounts := map[string]int{}
	for _, item := range board.Items {
		columnCounts[item.Status]++
	}

	summary := map[string]interface{}{
		"name":        board.Name,
		"total_items": m.TotalItems,
		"done_items":  m.DoneItems,
		"columns":     columnCounts,
	}

	data, _ := json.MarshalIndent(summary, "", "  ")
	return []mcplib.ResourceContents{
		mcplib.TextResourceContents{
			URI:      "obeya://board/summary",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *Server) handleBoardItemsResource(_ context.Context, _ mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	board, err := s.engine.ListBoard()
	if err != nil {
		return nil, err
	}

	items := make([]itemSummary, 0, len(board.Items))
	for _, item := range board.Items {
		items = append(items, toItemSummary(item, board))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Num < items[j].Num })

	data, _ := json.MarshalIndent(items, "", "  ")
	return []mcplib.ResourceContents{
		mcplib.TextResourceContents{
			URI:      "obeya://board/items",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *Server) handleMetricsResource(_ context.Context, _ mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	board, err := s.engine.ListBoard()
	if err != nil {
		return nil, err
	}

	items := metrics.BoardItems(board)
	m := metrics.Compute(items, time.Now())

	result := map[string]interface{}{
		"total_items": m.TotalItems,
		"done_items":  m.DoneItems,
		"throughput": map[string]interface{}{
			"this_week": m.Throughput.ThisWeek,
			"last_week": m.Throughput.LastWeek,
			"per_week":  m.Throughput.PerWeek,
		},
	}
	if m.CycleTime != nil {
		result["cycle_time"] = m.CycleTime.Display
	}
	if m.LeadTime != nil {
		result["lead_time"] = m.LeadTime.Display
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return []mcplib.ResourceContents{
		mcplib.TextResourceContents{
			URI:      "obeya://metrics",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *Server) handleUsersResource(_ context.Context, _ mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	board, err := s.engine.ListBoard()
	if err != nil {
		return nil, err
	}

	users := make([]map[string]string, 0, len(board.Users))
	for _, u := range board.Users {
		users = append(users, map[string]string{
			"id": u.ID, "name": u.Name, "type": string(u.Type), "provider": u.Provider,
		})
	}

	data, _ := json.MarshalIndent(users, "", "  ")
	return []mcplib.ResourceContents{
		mcplib.TextResourceContents{
			URI:      "obeya://users",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *Server) handlePlansResource(_ context.Context, _ mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
	plans, err := s.engine.ListPlans()
	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, 0, len(plans))
	for _, p := range plans {
		result = append(result, map[string]interface{}{
			"display_num":  p.DisplayNum,
			"title":        p.Title,
			"linked_items": len(p.LinkedItems),
		})
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return []mcplib.ResourceContents{
		mcplib.TextResourceContents{
			URI:      "obeya://plans",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

// --- Prompts ---

func (s *Server) registerPrompts() {
	s.mcp.AddPrompt(mcplib.NewPrompt("daily_standup",
		mcplib.WithPromptDescription("Generate a daily standup report from the board state."),
	), s.handleDailyStandup)

	s.mcp.AddPrompt(mcplib.NewPrompt("sprint_planning",
		mcplib.WithPromptDescription("Analyze the backlog and suggest items for the next sprint."),
		mcplib.WithArgument("capacity",
			mcplib.ArgumentDescription("Number of items the team can complete this sprint"),
		),
	), s.handleSprintPlanning)

	s.mcp.AddPrompt(mcplib.NewPrompt("triage_new_work",
		mcplib.WithPromptDescription("Help triage new work into properly structured board items."),
		mcplib.WithArgument("description",
			mcplib.ArgumentDescription("Description of the work to triage"),
			mcplib.RequiredArgument(),
		),
	), s.handleTriageNewWork)

	s.mcp.AddPrompt(mcplib.NewPrompt("retrospective",
		mcplib.WithPromptDescription("Generate a retrospective analysis using board metrics."),
	), s.handleRetrospective)
}

func (s *Server) handleDailyStandup(_ context.Context, _ mcplib.GetPromptRequest) (*mcplib.GetPromptResult, error) {
	board, err := s.engine.ListBoard()
	if err != nil {
		return nil, err
	}

	items := metrics.BoardItems(board)
	m := metrics.Compute(items, time.Now())

	// Build context
	var inProgress, review, blocked []string
	for _, item := range board.Items {
		switch {
		case item.Status == "in-progress":
			inProgress = append(inProgress, fmt.Sprintf("#%d %s", item.DisplayNum, item.Title))
		case item.Status == "review":
			review = append(review, fmt.Sprintf("#%d %s", item.DisplayNum, item.Title))
		case len(item.BlockedBy) > 0:
			blocked = append(blocked, fmt.Sprintf("#%d %s", item.DisplayNum, item.Title))
		}
	}

	context := fmt.Sprintf("Board: %s | Total: %d | Done: %d | Throughput this week: %d\n\n",
		board.Name, m.TotalItems, m.DoneItems, m.Throughput.ThisWeek)
	context += "In Progress:\n" + strings.Join(inProgress, "\n") + "\n\n"
	context += "In Review:\n" + strings.Join(review, "\n") + "\n\n"
	if len(blocked) > 0 {
		context += "Blocked:\n" + strings.Join(blocked, "\n") + "\n"
	}

	return &mcplib.GetPromptResult{
		Description: "Daily standup report from the Obeya board",
		Messages: []mcplib.PromptMessage{
			{
				Role: mcplib.RoleUser,
				Content: mcplib.TextContent{
					Type: "text",
					Text: "Based on this board state, give a concise standup update covering: what was completed recently, what's in progress, and any blockers.\n\n" + context,
				},
			},
		},
	}, nil
}

func (s *Server) handleSprintPlanning(_ context.Context, req mcplib.GetPromptRequest) (*mcplib.GetPromptResult, error) {
	capacity := ""
	if req.Params.Arguments != nil {
		capacity = req.Params.Arguments["capacity"]
	}

	items, err := s.engine.ListItems(engine.ListFilter{Status: "backlog"})
	if err != nil {
		return nil, err
	}

	var backlogDesc string
	for _, item := range items {
		backlogDesc += fmt.Sprintf("#%d [%s] %s (priority: %s)\n", item.DisplayNum, item.Type, item.Title, item.Priority)
	}

	prompt := "Review the backlog and suggest which items to pull into the next sprint."
	if capacity != "" {
		prompt += fmt.Sprintf(" The team can handle approximately %s items.", capacity)
	}
	prompt += "\n\nBacklog:\n" + backlogDesc

	return &mcplib.GetPromptResult{
		Description: "Sprint planning from backlog",
		Messages: []mcplib.PromptMessage{
			{Role: mcplib.RoleUser, Content: mcplib.TextContent{Type: "text", Text: prompt}},
		},
	}, nil
}

func (s *Server) handleTriageNewWork(_ context.Context, req mcplib.GetPromptRequest) (*mcplib.GetPromptResult, error) {
	desc := ""
	if req.Params.Arguments != nil {
		desc = req.Params.Arguments["description"]
	}

	board, _ := s.engine.ListBoard()
	var columns []string
	for _, c := range board.Columns {
		columns = append(columns, c.Name)
	}

	prompt := fmt.Sprintf(`Triage this work into Obeya board items. For each item, specify:
- type (epic/story/task)
- title
- priority (low/medium/high/critical)
- suggested parent (if any)
- description with acceptance criteria

Board columns: %s

Work to triage:
%s`, strings.Join(columns, ", "), desc)

	return &mcplib.GetPromptResult{
		Description: "Triage new work into board items",
		Messages: []mcplib.PromptMessage{
			{Role: mcplib.RoleUser, Content: mcplib.TextContent{Type: "text", Text: prompt}},
		},
	}, nil
}

func (s *Server) handleRetrospective(_ context.Context, _ mcplib.GetPromptRequest) (*mcplib.GetPromptResult, error) {
	board, err := s.engine.ListBoard()
	if err != nil {
		return nil, err
	}

	items := metrics.BoardItems(board)
	now := time.Now()
	m := metrics.Compute(items, now)

	var metricsDesc string
	metricsDesc += fmt.Sprintf("Total items: %d, Done: %d\n", m.TotalItems, m.DoneItems)
	metricsDesc += fmt.Sprintf("Throughput: %d this week, %d last week, %.1f/week avg\n",
		m.Throughput.ThisWeek, m.Throughput.LastWeek, m.Throughput.PerWeek)
	if m.CycleTime != nil {
		metricsDesc += fmt.Sprintf("Cycle time: %s\n", m.CycleTime.Display)
	}
	if m.LeadTime != nil {
		metricsDesc += fmt.Sprintf("Lead time: %s\n", m.LeadTime.Display)
	}
	for col, d := range m.Dwell {
		metricsDesc += fmt.Sprintf("Dwell in '%s': avg %s (%d items)\n", col, metrics.FormatDuration(d.Average), d.Count)
	}

	prompt := "Based on these board metrics, provide a retrospective analysis covering:\n" +
		"1. What went well (fast cycle times, good throughput)\n" +
		"2. What needs improvement (bottlenecks, long dwell times)\n" +
		"3. Actionable suggestions for the next sprint\n\n" +
		"Metrics:\n" + metricsDesc

	return &mcplib.GetPromptResult{
		Description: "Retrospective from board metrics",
		Messages: []mcplib.PromptMessage{
			{Role: mcplib.RoleUser, Content: mcplib.TextContent{Type: "text", Text: prompt}},
		},
	}, nil
}

// --- Helpers ---

type itemSummary struct {
	Num         int      `json:"display_num"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	Assignee    string   `json:"assignee"`
	Tags        []string `json:"tags,omitempty"`
	BlockedBy   int      `json:"blocked_by_count"`
	Description string   `json:"description,omitempty"`
}

type itemDetail struct {
	itemSummary
	ID          string                  `json:"id"`
	FullDesc    string                  `json:"description"`
	ParentID    string                  `json:"parent_id,omitempty"`
	Children    []itemSummary           `json:"children,omitempty"`
	LinkedPlans []map[string]interface{} `json:"linked_plans,omitempty"`
	History     []domain.ChangeRecord   `json:"history,omitempty"`
	CreatedAt   string                  `json:"created_at"`
	UpdatedAt   string                  `json:"updated_at"`
}

func toItemSummary(item *domain.Item, board *domain.Board) itemSummary {
	assigneeName := item.Assignee
	if board != nil && item.Assignee != "" {
		if u, ok := board.Users[item.Assignee]; ok {
			assigneeName = u.Name
		}
	}
	return itemSummary{
		Num:       item.DisplayNum,
		Type:      string(item.Type),
		Title:     item.Title,
		Status:    item.Status,
		Priority:  string(item.Priority),
		Assignee:  assigneeName,
		Tags:      item.Tags,
		BlockedBy: len(item.BlockedBy),
	}
}

func toItemDetail(item *domain.Item, children []*domain.Item, plans []*domain.Plan, board *domain.Board) itemDetail {
	d := itemDetail{
		itemSummary: toItemSummary(item, board),
		ID:          item.ID,
		FullDesc:    item.Description,
		ParentID:    item.ParentID,
		History:     item.History,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
	}

	for _, child := range children {
		d.Children = append(d.Children, toItemSummary(child, board))
	}
	for _, plan := range plans {
		d.LinkedPlans = append(d.LinkedPlans, map[string]interface{}{
			"display_num": plan.DisplayNum,
			"title":       plan.Title,
		})
	}

	return d
}

func stringArg(req mcplib.CallToolRequest, key string) string {
	args := req.GetArguments()
	v, ok := args[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	return s
}

func boolArg(req mcplib.CallToolRequest, key string) bool {
	args := req.GetArguments()
	v, ok := args[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func intArg(req mcplib.CallToolRequest, key string) int {
	args := req.GetArguments()
	v, ok := args[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func toolText(text string) *mcplib.CallToolResult {
	return &mcplib.CallToolResult{
		Content: []mcplib.Content{
			mcplib.TextContent{Type: "text", Text: text},
		},
	}
}

func toolError(err error) *mcplib.CallToolResult {
	return &mcplib.CallToolResult{
		IsError: true,
		Content: []mcplib.Content{
			mcplib.TextContent{Type: "text", Text: err.Error()},
		},
	}
}

func toolJSON(v interface{}) (*mcplib.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return toolError(fmt.Errorf("failed to serialize result: %w", err)), nil
	}
	return toolText(string(data)), nil
}
