package engine

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/niladribose/obeya/internal/domain"
)

func (e *Engine) CreatePlan(title, content, sourceFile string) (*domain.Plan, error) {
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}

	var created *domain.Plan
	err := e.store.Transaction(func(board *domain.Board) error {
		initPlans(board)

		now := time.Now()
		plan := &domain.Plan{
			ID:          domain.GenerateID(),
			DisplayNum:  board.NextDisplay,
			Title:       title,
			Content:     content,
			SourceFile:  sourceFile,
			LinkedItems: []string{},
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		board.Plans[plan.ID] = plan
		board.DisplayMap[plan.DisplayNum] = plan.ID
		board.NextDisplay++

		created = plan
		return nil
	})

	return created, err
}

func (e *Engine) ImportPlan(content, sourceFile string, linkRefs []string) (*domain.Plan, error) {
	title := extractTitleFromMarkdown(content)
	if title == "" {
		return nil, fmt.Errorf("no markdown heading found in content")
	}

	var created *domain.Plan
	err := e.store.Transaction(func(board *domain.Board) error {
		initPlans(board)

		linkedIDs, err := resolveItemRefs(board, linkRefs)
		if err != nil {
			return err
		}

		now := time.Now()
		plan := &domain.Plan{
			ID:          domain.GenerateID(),
			DisplayNum:  board.NextDisplay,
			Title:       title,
			Content:     content,
			SourceFile:  sourceFile,
			LinkedItems: linkedIDs,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		board.Plans[plan.ID] = plan
		board.DisplayMap[plan.DisplayNum] = plan.ID
		board.NextDisplay++

		created = plan
		return nil
	})

	return created, err
}

func (e *Engine) UpdatePlan(ref, title, content string) error {
	if title == "" && content == "" {
		return fmt.Errorf("no changes specified")
	}

	return e.store.Transaction(func(board *domain.Board) error {
		initPlans(board)

		id, err := board.ResolvePlanID(ref)
		if err != nil {
			return err
		}

		plan := board.Plans[id]
		if title != "" {
			plan.Title = title
		}
		if content != "" {
			plan.Content = content
		}
		plan.UpdatedAt = time.Now()

		return nil
	})
}

func (e *Engine) DeletePlan(ref string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		initPlans(board)

		id, err := board.ResolvePlanID(ref)
		if err != nil {
			return err
		}

		plan := board.Plans[id]
		delete(board.Plans, id)
		delete(board.DisplayMap, plan.DisplayNum)

		return nil
	})
}

func (e *Engine) ShowPlan(ref string) (*domain.Plan, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}

	id, err := board.ResolvePlanID(ref)
	if err != nil {
		return nil, err
	}

	return board.Plans[id], nil
}

func (e *Engine) ListPlans() ([]*domain.Plan, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}

	plans := make([]*domain.Plan, 0, len(board.Plans))
	for _, p := range board.Plans {
		plans = append(plans, p)
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].DisplayNum < plans[j].DisplayNum
	})

	return plans, nil
}

func (e *Engine) LinkPlan(ref string, itemRefs []string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		initPlans(board)

		id, err := board.ResolvePlanID(ref)
		if err != nil {
			return err
		}

		itemIDs, err := resolveItemRefs(board, itemRefs)
		if err != nil {
			return err
		}

		plan := board.Plans[id]
		for _, itemID := range itemIDs {
			if !containsString(plan.LinkedItems, itemID) {
				plan.LinkedItems = append(plan.LinkedItems, itemID)
			}
		}
		plan.UpdatedAt = time.Now()

		return nil
	})
}

func (e *Engine) UnlinkPlan(ref string, itemRefs []string) error {
	return e.store.Transaction(func(board *domain.Board) error {
		initPlans(board)

		id, err := board.ResolvePlanID(ref)
		if err != nil {
			return err
		}

		itemIDs, err := resolveItemRefs(board, itemRefs)
		if err != nil {
			return err
		}

		plan := board.Plans[id]
		for _, itemID := range itemIDs {
			filtered, found := removeString(plan.LinkedItems, itemID)
			if !found {
				return fmt.Errorf("item %s is not linked to this plan", itemID)
			}
			plan.LinkedItems = filtered
		}
		plan.UpdatedAt = time.Now()

		return nil
	})
}

func (e *Engine) PlansForItem(itemID string) ([]*domain.Plan, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}

	var plans []*domain.Plan
	for _, plan := range board.Plans {
		if containsString(plan.LinkedItems, itemID) {
			plans = append(plans, plan)
		}
	}

	sort.Slice(plans, func(i, j int) bool {
		return plans[i].DisplayNum < plans[j].DisplayNum
	})

	return plans, nil
}

// --- helpers ---

func initPlans(board *domain.Board) {
	if board.Plans == nil {
		board.Plans = make(map[string]*domain.Plan)
	}
}

func extractTitleFromMarkdown(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}
	return ""
}

func resolveItemRefs(board *domain.Board, refs []string) ([]string, error) {
	ids := make([]string, 0, len(refs))
	for _, ref := range refs {
		id, err := board.ResolveID(ref)
		if err != nil {
			return nil, fmt.Errorf("invalid item reference %q: %w", ref, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
