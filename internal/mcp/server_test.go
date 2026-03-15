package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcptest "github.com/mark3labs/mcp-go/mcptest"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/niladribose/obeya/internal/engine"
	"github.com/niladribose/obeya/internal/store"
)

// setupTestServer creates a temp board, engine, and MCP server for testing.
func setupTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	boardDir := filepath.Join(dir, ".obeya")
	if err := os.MkdirAll(boardDir, 0755); err != nil {
		t.Fatal(err)
	}

	s := store.NewJSONStore(dir)
	if err := s.InitBoard("test-board", []string{"backlog", "todo", "in-progress", "review", "done"}); err != nil {
		t.Fatal(err)
	}

	eng := engine.New(s)
	// Add a test user
	if err := eng.AddUser("test-agent", "agent", "claude-code"); err != nil {
		t.Fatal(err)
	}

	srv := New(eng)
	return srv, dir
}

// callTool invokes a tool on the server and returns the text result.
func callTool(t *testing.T, srv *Server, name string, args map[string]interface{}) string {
	t.Helper()
	req := mcplib.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	result, err := findAndCallTool(srv, name, context.Background(), req)
	if err != nil {
		t.Fatalf("tool %s error: %v", name, err)
	}
	if result.IsError {
		text := extractText(result)
		t.Fatalf("tool %s returned error: %s", name, text)
	}
	return extractText(result)
}

// callToolExpectError invokes a tool expecting an error result.
func callToolExpectError(t *testing.T, srv *Server, name string, args map[string]interface{}) string {
	t.Helper()
	req := mcplib.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	result, err := findAndCallTool(srv, name, context.Background(), req)
	if err != nil {
		t.Fatalf("tool %s unexpected error: %v", name, err)
	}
	if !result.IsError {
		t.Fatalf("expected error from %s, got success: %s", name, extractText(result))
	}
	return extractText(result)
}

// findAndCallTool finds a tool handler and calls it.
func findAndCallTool(srv *Server, name string, ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
	// We access the handler directly via our server struct methods
	switch name {
	case "list_items":
		return srv.handleListItems(ctx, req)
	case "get_item":
		return srv.handleGetItem(ctx, req)
	case "create_item":
		return srv.handleCreateItem(ctx, req)
	case "move_item":
		return srv.handleMoveItem(ctx, req)
	case "edit_item":
		return srv.handleEditItem(ctx, req)
	case "delete_item":
		return srv.handleDeleteItem(ctx, req)
	case "assign_item":
		return srv.handleAssignItem(ctx, req)
	case "block_item":
		return srv.handleBlockItem(ctx, req)
	case "unblock_item":
		return srv.handleUnblockItem(ctx, req)
	case "list_users":
		return srv.handleListUsers(ctx, req)
	case "add_user":
		return srv.handleAddUser(ctx, req)
	case "list_plans":
		return srv.handleListPlans(ctx, req)
	case "get_plan":
		return srv.handleGetPlan(ctx, req)
	case "create_plan":
		return srv.handleCreatePlan(ctx, req)
	case "link_plan":
		return srv.handleLinkPlan(ctx, req)
	case "list_columns":
		return srv.handleListColumns(ctx, req)
	case "add_column":
		return srv.handleAddColumn(ctx, req)
	case "get_metrics":
		return srv.handleGetMetrics(ctx, req)
	case "get_burndown":
		return srv.handleGetBurndown(ctx, req)
	case "pick_next_task":
		return srv.handlePickNextTask(ctx, req)
	case "complete_task":
		return srv.handleCompleteTask(ctx, req)
	case "board_summary":
		return srv.handleBoardSummary(ctx, req)
	default:
		return nil, nil
	}
}

func extractText(result *mcplib.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(mcplib.TextContent); ok {
		return tc.Text
	}
	return ""
}

// --- Tests ---

func TestCreateAndListItems(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Create a task
	text := callTool(t, srv, "create_item", map[string]interface{}{
		"type":     "task",
		"title":    "Fix the build",
		"assignee": "test-agent",
		"priority": "high",
	})

	var created map[string]interface{}
	if err := json.Unmarshal([]byte(text), &created); err != nil {
		t.Fatalf("failed to parse create result: %v", err)
	}
	if created["title"] != "Fix the build" {
		t.Errorf("expected title 'Fix the build', got %v", created["title"])
	}
	if created["created"] != true {
		t.Errorf("expected created=true")
	}

	// List items
	text = callTool(t, srv, "list_items", map[string]interface{}{})
	var items []itemSummary
	if err := json.Unmarshal([]byte(text), &items); err != nil {
		t.Fatalf("failed to parse list result: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Title != "Fix the build" {
		t.Errorf("expected 'Fix the build', got %s", items[0].Title)
	}
}

func TestMoveItem(t *testing.T) {
	srv, _ := setupTestServer(t)

	callTool(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "Deploy", "assignee": "test-agent",
	})

	text := callTool(t, srv, "move_item", map[string]interface{}{
		"ref": "1", "status": "in-progress",
	})
	if text != "Moved #1 to in-progress" {
		t.Errorf("unexpected move result: %s", text)
	}

	// Verify status
	detailText := callTool(t, srv, "get_item", map[string]interface{}{"ref": "1"})
	var detail itemDetail
	if err := json.Unmarshal([]byte(detailText), &detail); err != nil {
		t.Fatalf("failed to parse detail: %v", err)
	}
	if detail.Status != "in-progress" {
		t.Errorf("expected status 'in-progress', got %s", detail.Status)
	}
}

func TestEditItem(t *testing.T) {
	srv, _ := setupTestServer(t)

	callTool(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "Original", "assignee": "test-agent",
	})

	callTool(t, srv, "edit_item", map[string]interface{}{
		"ref": "1", "title": "Updated Title", "priority": "critical",
	})

	detailText := callTool(t, srv, "get_item", map[string]interface{}{"ref": "1"})
	var detail itemDetail
	json.Unmarshal([]byte(detailText), &detail)
	if detail.Title != "Updated Title" {
		t.Errorf("expected 'Updated Title', got %s", detail.Title)
	}
	if detail.Priority != "critical" {
		t.Errorf("expected priority 'critical', got %s", detail.Priority)
	}
}

func TestDeleteItem(t *testing.T) {
	srv, _ := setupTestServer(t)

	callTool(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "To Delete", "assignee": "test-agent",
	})

	callTool(t, srv, "delete_item", map[string]interface{}{"ref": "1"})

	// List should be empty
	text := callTool(t, srv, "list_items", map[string]interface{}{})
	var items []itemSummary
	json.Unmarshal([]byte(text), &items)
	if len(items) != 0 {
		t.Errorf("expected 0 items after delete, got %d", len(items))
	}
}

func TestAssignItem(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Add second user
	callTool(t, srv, "add_user", map[string]interface{}{
		"name": "alice", "type": "human",
	})

	callTool(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "Reassign Me", "assignee": "test-agent",
	})

	callTool(t, srv, "assign_item", map[string]interface{}{
		"ref": "1", "assignee": "alice",
	})

	detailText := callTool(t, srv, "get_item", map[string]interface{}{"ref": "1"})
	var detail itemDetail
	json.Unmarshal([]byte(detailText), &detail)
	if detail.Assignee != "alice" {
		t.Errorf("expected assignee 'alice', got %s", detail.Assignee)
	}
}

func TestBlockUnblock(t *testing.T) {
	srv, _ := setupTestServer(t)

	callTool(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "Blocked Task", "assignee": "test-agent",
	})
	callTool(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "Blocker", "assignee": "test-agent",
	})

	callTool(t, srv, "block_item", map[string]interface{}{
		"ref": "1", "blocker": "2",
	})

	// Verify blocked
	detailText := callTool(t, srv, "get_item", map[string]interface{}{"ref": "1"})
	var detail itemDetail
	json.Unmarshal([]byte(detailText), &detail)
	if detail.BlockedBy != 1 {
		t.Errorf("expected 1 blocker, got %d", detail.BlockedBy)
	}

	// Unblock
	callTool(t, srv, "unblock_item", map[string]interface{}{
		"ref": "1", "blocker": "2",
	})

	detailText = callTool(t, srv, "get_item", map[string]interface{}{"ref": "1"})
	json.Unmarshal([]byte(detailText), &detail)
	if detail.BlockedBy != 0 {
		t.Errorf("expected 0 blockers after unblock, got %d", detail.BlockedBy)
	}
}

func TestListUsers(t *testing.T) {
	srv, _ := setupTestServer(t)

	text := callTool(t, srv, "list_users", map[string]interface{}{})
	var users []map[string]string
	if err := json.Unmarshal([]byte(text), &users); err != nil {
		t.Fatalf("failed to parse users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0]["name"] != "test-agent" {
		t.Errorf("expected 'test-agent', got %s", users[0]["name"])
	}
}

func TestPlanOperations(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Create plan
	text := callTool(t, srv, "create_plan", map[string]interface{}{
		"title": "Sprint Plan", "content": "# Sprint Plan\n\nGoals for the sprint.",
	})
	var plan map[string]interface{}
	json.Unmarshal([]byte(text), &plan)
	if plan["title"] != "Sprint Plan" {
		t.Errorf("expected 'Sprint Plan', got %v", plan["title"])
	}

	// List plans
	text = callTool(t, srv, "list_plans", map[string]interface{}{})
	var plans []map[string]interface{}
	json.Unmarshal([]byte(text), &plans)
	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}
}

func TestListColumns(t *testing.T) {
	srv, _ := setupTestServer(t)

	text := callTool(t, srv, "list_columns", map[string]interface{}{})
	var cols []map[string]interface{}
	if err := json.Unmarshal([]byte(text), &cols); err != nil {
		t.Fatalf("failed to parse columns: %v", err)
	}
	// Should have all 5 columns (4 non-done from WIP + done)
	if len(cols) != 5 {
		t.Errorf("expected 5 columns, got %d", len(cols))
	}
}

func TestGetMetrics(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Create and complete some items
	callTool(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "Task 1", "assignee": "test-agent",
	})
	callTool(t, srv, "move_item", map[string]interface{}{"ref": "1", "status": "done"})

	text := callTool(t, srv, "get_metrics", map[string]interface{}{"days": 7})
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		t.Fatalf("failed to parse metrics: %v", err)
	}
	if m["total_items"].(float64) != 1 {
		t.Errorf("expected 1 total item, got %v", m["total_items"])
	}
	if m["done_items"].(float64) != 1 {
		t.Errorf("expected 1 done item, got %v", m["done_items"])
	}
}

func TestBoardSummary(t *testing.T) {
	srv, _ := setupTestServer(t)

	callTool(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "Active Task", "assignee": "test-agent",
	})

	text := callTool(t, srv, "board_summary", map[string]interface{}{})
	var summary map[string]interface{}
	if err := json.Unmarshal([]byte(text), &summary); err != nil {
		t.Fatalf("failed to parse summary: %v", err)
	}
	if summary["board_name"] != "test-board" {
		t.Errorf("expected board name 'test-board', got %v", summary["board_name"])
	}
	if summary["total_items"].(float64) != 1 {
		t.Errorf("expected 1 total item, got %v", summary["total_items"])
	}
}

func TestPickNextTask(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Create unassigned task — need a second user to create, then reassign...
	// Actually, create_item requires assignee. So pick won't find unassigned items.
	// Let's test the empty case.
	text := callTool(t, srv, "pick_next_task", map[string]interface{}{})
	if text != "No available unassigned, unblocked tasks in 'todo'" {
		t.Errorf("unexpected pick result: %s", text)
	}
}

func TestCompleteTask(t *testing.T) {
	srv, _ := setupTestServer(t)

	callTool(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "Complete Me", "assignee": "test-agent",
	})

	text := callTool(t, srv, "complete_task", map[string]interface{}{
		"ref": "1", "notes": "All done!",
	})
	if text != "Completed #1" {
		t.Errorf("unexpected complete result: %s", text)
	}

	// Verify done
	detailText := callTool(t, srv, "get_item", map[string]interface{}{"ref": "1"})
	var detail itemDetail
	json.Unmarshal([]byte(detailText), &detail)
	if detail.Status != "done" {
		t.Errorf("expected status 'done', got %s", detail.Status)
	}
}

func TestFilterByStatus(t *testing.T) {
	srv, _ := setupTestServer(t)

	callTool(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "Backlog Item", "assignee": "test-agent",
	})
	callTool(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "Another Item", "assignee": "test-agent",
	})
	callTool(t, srv, "move_item", map[string]interface{}{"ref": "2", "status": "done"})

	// Filter backlog only
	text := callTool(t, srv, "list_items", map[string]interface{}{"status": "backlog"})
	var items []itemSummary
	json.Unmarshal([]byte(text), &items)
	if len(items) != 1 {
		t.Errorf("expected 1 backlog item, got %d", len(items))
	}

	// Filter done
	text = callTool(t, srv, "list_items", map[string]interface{}{"status": "done"})
	json.Unmarshal([]byte(text), &items)
	if len(items) != 1 {
		t.Errorf("expected 1 done item, got %d", len(items))
	}
}

func TestMCPServerRegistration(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Verify the MCPServer was created
	if srv.MCPServer() == nil {
		t.Fatal("MCPServer() returned nil")
	}
}

func TestAddColumn(t *testing.T) {
	srv, _ := setupTestServer(t)

	callTool(t, srv, "add_column", map[string]interface{}{"name": "staging"})

	text := callTool(t, srv, "list_columns", map[string]interface{}{})
	var cols []map[string]interface{}
	json.Unmarshal([]byte(text), &cols)

	found := false
	for _, col := range cols {
		if col["name"] == "staging" {
			found = true
			break
		}
	}
	if !found {
		t.Error("staging column not found after add_column")
	}
}

func TestCreateItemError(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Missing assignee
	errText := callToolExpectError(t, srv, "create_item", map[string]interface{}{
		"type": "task", "title": "No Assignee",
	})
	if errText == "" {
		t.Error("expected error for missing assignee")
	}
}

// TestMCPTestIntegration uses the mcptest package for end-to-end testing.
func TestMCPTestIntegration(t *testing.T) {
	srv, _ := setupTestServer(t)

	// Collect server tools
	var tools []mcpserver.ServerTool
	tools = append(tools, mcpserver.ServerTool{
		Tool: mcplib.NewTool("list_items",
			mcplib.WithDescription("List items"),
		),
		Handler: srv.handleListItems,
	})
	tools = append(tools, mcpserver.ServerTool{
		Tool: mcplib.NewTool("board_summary",
			mcplib.WithDescription("Board summary"),
		),
		Handler: srv.handleBoardSummary,
	})

	ts, err := mcptest.NewServer(t, tools...)
	if err != nil {
		t.Fatalf("failed to start mcptest server: %v", err)
	}
	defer ts.Close()

	// Call list_items through the MCP protocol
	result, err := ts.Client().CallTool(context.Background(), mcplib.CallToolRequest{
		Params: mcplib.CallToolParams{
			Name:      "list_items",
			Arguments: map[string]interface{}{},
		},
	})
	if err != nil {
		t.Fatalf("mcptest call error: %v", err)
	}
	if result.IsError {
		t.Fatalf("mcptest call returned error: %v", result.Content)
	}
}
