package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
)

// breadcrumbPath returns the ancestry path for an item using display numbers.
// Example: "#1 › #4"
// Compact and never needs truncation.
func breadcrumbPath(board *domain.Board, item *domain.Item, maxWidth int) string {
	if item.ParentID == "" {
		return ""
	}

	var segments []string
	visited := map[string]bool{item.ID: true}
	cur := item
	for cur.ParentID != "" {
		if visited[cur.ParentID] {
			break
		}
		visited[cur.ParentID] = true
		parent, ok := board.Items[cur.ParentID]
		if !ok {
			break
		}
		segments = append([]string{fmt.Sprintf("#%d", parent.DisplayNum)}, segments...)
		cur = parent
	}

	if len(segments) == 0 {
		return ""
	}

	return strings.Join(segments, " › ")
}

// childCount returns the total number of descendants of an item.
func childCount(board *domain.Board, itemID string) int {
	count := 0
	for _, item := range board.Items {
		if isDescendantOf(board, item, itemID) {
			count++
		}
	}
	return count
}

// doneCount returns the number of descendants in the board's last column.
func doneCount(board *domain.Board, itemID string) int {
	doneCol := "done"
	if len(board.Columns) > 0 {
		doneCol = board.Columns[len(board.Columns)-1].Name
	}
	count := 0
	for _, item := range board.Items {
		if item.Status == doneCol && isDescendantOf(board, item, itemID) {
			count++
		}
	}
	return count
}

// hasChildItems returns true if any item on the board has itemID as its ParentID.
func hasChildItems(board *domain.Board, itemID string) bool {
	for _, item := range board.Items {
		if item.ParentID == itemID {
			return true
		}
	}
	return false
}

// isDescendantOf checks if item is a descendant of ancestorID (with cycle protection).
func isDescendantOf(board *domain.Board, item *domain.Item, ancestorID string) bool {
	visited := map[string]bool{item.ID: true}
	cur := item
	for cur.ParentID != "" {
		if cur.ParentID == ancestorID {
			return true
		}
		if visited[cur.ParentID] {
			return false
		}
		visited[cur.ParentID] = true
		parent, ok := board.Items[cur.ParentID]
		if !ok {
			return false
		}
		cur = parent
	}
	return false
}

// isHiddenByCollapse checks if an item should be hidden because
// a collapsed ancestor is in the same column.
func isHiddenByCollapse(board *domain.Board, item *domain.Item, collapsed map[string]bool) bool {
	visited := map[string]bool{item.ID: true}
	cur := item
	for cur.ParentID != "" {
		if visited[cur.ParentID] {
			return false
		}
		visited[cur.ParentID] = true
		parent, ok := board.Items[cur.ParentID]
		if !ok {
			return false
		}
		if collapsed[parent.ID] && parent.Status == item.Status {
			return true
		}
		cur = parent
	}
	return false
}

// orderItemsHierarchically sorts items so parents appear before children,
// using depth-first traversal ordered by DisplayNum.
func orderItemsHierarchically(board *domain.Board, items []*domain.Item) []*domain.Item {
	itemSet := map[string]bool{}
	for _, it := range items {
		itemSet[it.ID] = true
	}

	// Find roots: items whose parent is not in this column's item set
	var roots []*domain.Item
	childrenOf := map[string][]*domain.Item{}

	for _, it := range items {
		if it.ParentID == "" || !itemSet[it.ParentID] {
			roots = append(roots, it)
		} else {
			childrenOf[it.ParentID] = append(childrenOf[it.ParentID], it)
		}
	}

	// Sort roots and children by DisplayNum ascending
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].DisplayNum < roots[j].DisplayNum
	})
	for k := range childrenOf {
		children := childrenOf[k]
		sort.Slice(children, func(i, j int) bool {
			return children[i].DisplayNum < children[j].DisplayNum
		})
	}

	// Depth-first emit
	var ordered []*domain.Item
	var walk func(item *domain.Item)
	walk = func(item *domain.Item) {
		ordered = append(ordered, item)
		for _, child := range childrenOf[item.ID] {
			walk(child)
		}
	}
	for _, root := range roots {
		walk(root)
	}

	return ordered
}
