package engine

import "github.com/niladribose/obeya/internal/domain"

type ListFilter struct {
	Status   string
	Assignee string
	Type     string
	Tag      string
	Blocked  bool
	Flat     bool
}

func (e *Engine) ListItems(filter ListFilter) ([]*domain.Item, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}

	var items []*domain.Item
	for _, item := range board.Items {
		if matchesFilter(item, filter) {
			items = append(items, item)
		}
	}

	return items, nil
}

func (e *Engine) GetChildren(parentID string) ([]*domain.Item, error) {
	board, err := e.store.LoadBoard()
	if err != nil {
		return nil, err
	}

	var children []*domain.Item
	for _, item := range board.Items {
		if item.ParentID == parentID {
			children = append(children, item)
		}
	}
	return children, nil
}

func matchesFilter(item *domain.Item, f ListFilter) bool {
	if f.Status != "" && item.Status != f.Status {
		return false
	}
	if f.Assignee != "" && item.Assignee != f.Assignee {
		return false
	}
	if f.Type != "" && string(item.Type) != f.Type {
		return false
	}
	if f.Tag != "" && !containsString(item.Tags, f.Tag) {
		return false
	}
	if f.Blocked && len(item.BlockedBy) == 0 {
		return false
	}
	return true
}
