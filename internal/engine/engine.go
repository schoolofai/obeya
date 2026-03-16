package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/niladribose/obeya/internal/domain"
	"github.com/niladribose/obeya/internal/store"
)

type Engine struct {
	store store.Store
}

func New(s store.Store) *Engine {
	return &Engine{store: s}
}

// BoardFilePath returns the store's board file path (for file watching).
func (e *Engine) BoardFilePath() string {
	return e.store.BoardFilePath()
}

func (e *Engine) CreateItem(itemType, title, parentRef, description, priority, assignee string, tags []string, sponsor string) (*domain.Item, error) {
	if err := validateCreateInput(itemType, title, priority); err != nil {
		return nil, err
	}
	if priority == "" {
		priority = "medium"
	}

	var created *domain.Item
	err := e.store.Transaction(func(board *domain.Board) error {
		parentID, err := resolveParent(board, parentRef)
		if err != nil {
			return err
		}

		if assignee == "" {
			return fmt.Errorf("assignee is required. Every item must have an owner.\n\n" +
				"Run 'ob user list' to see registered users.")
		}
		resolvedAssignee, err := board.ResolveUserID(assignee)
		if err != nil {
			return fmt.Errorf("unknown assignee %q: %w\nRun 'ob user list' to see registered users", assignee, err)
		}

		resolvedSponsor, err := resolveSponsor(board, resolvedAssignee, sponsor, parentID)
		if err != nil {
			return err
		}

		item := buildItem(board, itemType, title, description, priority, resolvedAssignee, parentID, tags)
		item.Sponsor = resolvedSponsor
		board.Items[item.ID] = item
		board.DisplayMap[item.DisplayNum] = item.ID
		board.NextDisplay++

		created = item
		return nil
	})

	return created, err
}

func (e *Engine) CompleteItemWithContext(ref string, ctx domain.ReviewContext, confidence int, userID string, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		item := board.Items[id]
		if err := CheckAssignee(item); err != nil {
			return err
		}

		oldStatus := item.Status
		item.Status = "done"
		item.ReviewContext = &ctx
		item.Confidence = &confidence
		item.HumanReview = &domain.HumanReview{Status: "pending"}
		item.UpdatedAt = time.Now()
		appendHistory(item, userID, sessionID, "complete-with-context",
			fmt.Sprintf("status: %s -> done, purpose: %s", oldStatus, ctx.Purpose))

		return nil
	})
}

func (e *Engine) ReviewItem(ref string, status string, userID string, sessionID string) error {
	if status != "reviewed" && status != "hidden" {
		return fmt.Errorf("invalid review status %q: must be 'reviewed' or 'hidden'", status)
	}

	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		actorType := resolveActorTypeFromBoard(board, userID)
		if actorType == "agent" {
			return fmt.Errorf("agents cannot review items — only humans can mark items as reviewed")
		}

		item := board.Items[id]
		item.HumanReview = &domain.HumanReview{
			Status:     status,
			ReviewedBy: userID,
			ReviewedAt: time.Now(),
		}
		item.UpdatedAt = time.Now()
		appendHistory(item, userID, sessionID, "human-review", status)

		return nil
	})
}

func (e *Engine) MoveItem(ref, status, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		if !board.HasColumn(status) {
			return fmt.Errorf("invalid status %q — available columns: %s", status, columnNames(board))
		}

		item := board.Items[id]
		if err := CheckAssignee(item); err != nil {
			return err
		}
		oldStatus := item.Status
		item.Status = status
		item.UpdatedAt = time.Now()
		appendHistory(item, userID, sessionID, "moved", fmt.Sprintf("status: %s -> %s", oldStatus, status))

		return nil
	})
}

func (e *Engine) AssignItem(ref, userRef, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		assigneeID, err := board.ResolveUserID(userRef)
		if err != nil {
			return err
		}

		item := board.Items[id]
		item.Assignee = assigneeID
		item.UpdatedAt = time.Now()
		appendHistory(item, userID, sessionID, "assigned", fmt.Sprintf("assigned to %s", assigneeID))

		return nil
	})
}

func (e *Engine) BlockItem(ref, blockerRef, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, blockerID, err := resolvePair(board, ref, blockerRef)
		if err != nil {
			return err
		}

		if id == blockerID {
			return fmt.Errorf("an item cannot block itself")
		}

		item := board.Items[id]
		if err := CheckAssignee(item); err != nil {
			return err
		}
		if containsString(item.BlockedBy, blockerID) {
			return fmt.Errorf("item #%d is already blocked by #%d", item.DisplayNum, board.Items[blockerID].DisplayNum)
		}

		item.BlockedBy = append(item.BlockedBy, blockerID)
		item.UpdatedAt = time.Now()
		appendHistory(item, userID, sessionID, "blocked", fmt.Sprintf("blocked by %s", blockerID))

		return nil
	})
}

func (e *Engine) UnblockItem(ref, blockerRef, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, blockerID, err := resolvePair(board, ref, blockerRef)
		if err != nil {
			return err
		}

		item := board.Items[id]
		if err := CheckAssignee(item); err != nil {
			return err
		}
		filtered, found := removeString(item.BlockedBy, blockerID)
		if !found {
			return fmt.Errorf("item is not blocked by %s", blockerID)
		}

		item.BlockedBy = filtered
		item.UpdatedAt = time.Now()
		appendHistory(item, userID, sessionID, "unblocked", fmt.Sprintf("unblocked from %s", blockerID))

		return nil
	})
}

func (e *Engine) EditItem(ref, title, description, priority, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		item := board.Items[id]
		if err := CheckAssignee(item); err != nil {
			return err
		}
		changes, err := applyEdits(item, title, description, priority)
		if err != nil {
			return err
		}

		item.UpdatedAt = time.Now()
		for _, change := range changes {
			appendHistory(item, userID, sessionID, "edited", change)
		}

		return nil
	})
}

func (e *Engine) DeleteItem(ref, userID, sessionID string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveID(ref)
		if err != nil {
			return err
		}

		if hasChildren(board, id) {
			return fmt.Errorf("cannot delete item #%d: it has children — delete children first", board.Items[id].DisplayNum)
		}

		item := board.Items[id]
		if err := CheckAssignee(item); err != nil {
			return err
		}
		delete(board.Items, id)
		delete(board.DisplayMap, item.DisplayNum)

		return nil
	})
}

func (e *Engine) GetItem(ref string) (*domain.Item, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}

	id, err := board.ResolveID(ref)
	if err != nil {
		return nil, err
	}

	return board.Items[id], nil
}

func (e *Engine) ListBoard() (*domain.Board, error) {
	return e.store.LoadBoard()
}

func (e *Engine) AddUser(name, identityType, provider string) (bool, error) {
	if err := domain.IdentityType(identityType).Validate(); err != nil {
		return false, err
	}

	added := false
	err := e.store.Transaction(func(board *domain.Board) error {
		for _, u := range board.Users {
			if strings.EqualFold(u.Name, name) {
				return nil
			}
		}
		identity := &domain.Identity{
			ID:       domain.GenerateID(),
			Name:     name,
			Type:     domain.IdentityType(identityType),
			Provider: provider,
		}
		board.Users[identity.ID] = identity
		added = true
		return nil
	})
	return added, err
}

func (e *Engine) RemoveUser(ref string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		id, err := board.ResolveUserID(ref)
		if err != nil {
			return err
		}
		delete(board.Users, id)
		return nil
	})
}

func (e *Engine) AddColumn(name string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		if board.HasColumn(name) {
			return fmt.Errorf("column %q already exists", name)
		}
		board.Columns = append(board.Columns, domain.Column{Name: name})
		return nil
	})
}

func (e *Engine) RemoveColumn(name string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		if !board.HasColumn(name) {
			return fmt.Errorf("column %q does not exist", name)
		}
		if columnHasItems(board, name) {
			return fmt.Errorf("column %q has items — move or delete them first", name)
		}
		board.Columns = filterColumns(board.Columns, name)
		return nil
	})
}

func (e *Engine) ReorderColumns(names []string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		if len(names) != len(board.Columns) {
			return fmt.Errorf("must specify all %d columns, got %d", len(board.Columns), len(names))
		}
		reordered, err := buildReorderedColumns(board, names)
		if err != nil {
			return err
		}
		board.Columns = reordered
		return nil
	})
}

// --- helpers ---

func validateCreateInput(itemType, title, priority string) error {
	if err := domain.ItemType(itemType).Validate(); err != nil {
		return err
	}
	if title == "" {
		return fmt.Errorf("title is required")
	}
	if priority != "" {
		if err := domain.Priority(priority).Validate(); err != nil {
			return err
		}
	}
	return nil
}

func resolveParent(board *domain.Board, parentRef string) (string, error) {
	if parentRef == "" {
		return "", nil
	}
	resolved, err := board.ResolveID(parentRef)
	if err != nil {
		return "", fmt.Errorf("invalid parent: %w", err)
	}
	return resolved, nil
}

func buildItem(board *domain.Board, itemType, title, description, priority, assignee, parentID string, tags []string) *domain.Item {
	now := time.Now()
	return &domain.Item{
		ID:          domain.GenerateID(),
		DisplayNum:  board.NextDisplay,
		Type:        domain.ItemType(itemType),
		Title:       title,
		Description: description,
		Status:      board.Columns[0].Name,
		Priority:    domain.Priority(priority),
		Assignee:    assignee,
		ParentID:    parentID,
		Tags:        tags,
		CreatedAt:   now,
		UpdatedAt:   now,
		History: []domain.ChangeRecord{
			{Action: "created", Detail: fmt.Sprintf("created %s: %s", itemType, title), Timestamp: now},
		},
	}
}

func resolvePair(board *domain.Board, ref, otherRef string) (string, string, error) {
	id, err := board.ResolveID(ref)
	if err != nil {
		return "", "", err
	}
	otherID, err := board.ResolveID(otherRef)
	if err != nil {
		return "", "", fmt.Errorf("invalid blocker: %w", err)
	}
	return id, otherID, nil
}

func applyEdits(item *domain.Item, title, description, priority string) ([]string, error) {
	var changes []string
	if title != "" {
		item.Title = title
		changes = append(changes, fmt.Sprintf("title changed to %q", title))
	}
	if description != "" {
		item.Description = description
		changes = append(changes, "description updated")
	}
	if priority != "" {
		if err := domain.Priority(priority).Validate(); err != nil {
			return nil, err
		}
		item.Priority = domain.Priority(priority)
		changes = append(changes, fmt.Sprintf("priority changed to %s", priority))
	}
	if len(changes) == 0 {
		return nil, fmt.Errorf("no changes specified")
	}
	return changes, nil
}

func hasChildren(board *domain.Board, id string) bool {
	for _, item := range board.Items {
		if item.ParentID == id {
			return true
		}
	}
	return false
}

func appendHistory(item *domain.Item, userID, sessionID, action, detail string) {
	item.History = append(item.History, domain.ChangeRecord{
		UserID:    userID,
		SessionID: sessionID,
		Action:    action,
		Detail:    detail,
		Timestamp: time.Now(),
	})
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) ([]string, bool) {
	filtered := make([]string, 0, len(slice))
	found := false
	for _, v := range slice {
		if v == s {
			found = true
			continue
		}
		filtered = append(filtered, v)
	}
	return filtered, found
}

func columnNames(board *domain.Board) string {
	names := ""
	for i, c := range board.Columns {
		if i > 0 {
			names += ", "
		}
		names += c.Name
	}
	return names
}

func columnHasItems(board *domain.Board, name string) bool {
	for _, item := range board.Items {
		if item.Status == name {
			return true
		}
	}
	return false
}

func filterColumns(cols []domain.Column, name string) []domain.Column {
	filtered := make([]domain.Column, 0, len(cols)-1)
	for _, c := range cols {
		if c.Name != name {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func buildReorderedColumns(board *domain.Board, names []string) ([]domain.Column, error) {
	colMap := make(map[string]domain.Column, len(board.Columns))
	for _, c := range board.Columns {
		colMap[c.Name] = c
	}

	reordered := make([]domain.Column, 0, len(names))
	for _, name := range names {
		col, ok := colMap[name]
		if !ok {
			return nil, fmt.Errorf("unknown column %q", name)
		}
		reordered = append(reordered, col)
	}
	return reordered, nil
}

// CheckAssignee returns an error if the item has no assignee.
// Must be called inside a Transaction callback.
func CheckAssignee(item *domain.Item) error {
	if item.Assignee == "" {
		return fmt.Errorf("item #%d has no assignee. Assign it first:\n\n"+
			"  ob assign %d --to <user>\n\n"+
			"Examples:\n"+
			"  ob assign %d --to claude\n"+
			"  ob assign %d --to niladri\n\n"+
			"Run 'ob user list' to see registered users.",
			item.DisplayNum, item.DisplayNum, item.DisplayNum, item.DisplayNum)
	}
	return nil
}
