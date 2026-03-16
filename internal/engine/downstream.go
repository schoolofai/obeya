package engine

import "github.com/niladribose/obeya/internal/domain"

// ResolveDownstream returns IDs of items directly blocked by the given item.
func ResolveDownstream(itemID string, board *domain.Board) []string {
	var downstream []string
	for _, item := range board.Items {
		for _, blockerID := range item.BlockedBy {
			if blockerID == itemID {
				downstream = append(downstream, item.ID)
				break
			}
		}
	}
	return downstream
}
