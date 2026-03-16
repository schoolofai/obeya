package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/niladribose/obeya/internal/domain"
)

// resolveSponsor determines the sponsor for a new item based on resolution rules.
// Returns empty string for human actors (they are their own owner).
func resolveSponsor(board *domain.Board, assigneeID string, explicitSponsor string, parentRef string) (string, error) {
	actor, ok := board.Users[assigneeID]
	if !ok {
		return "", nil
	}
	if actor.Type == domain.IdentityHuman {
		return "", nil
	}

	if explicitSponsor != "" {
		return validateExplicitSponsor(board, explicitSponsor)
	}

	humans := humanUsers(board)
	if len(humans) == 1 {
		return humans[0].ID, nil
	}

	if parentRef != "" {
		if parent, ok := board.Items[parentRef]; ok && parent.Sponsor != "" {
			return parent.Sponsor, nil
		}
	}

	return "", sponsorRequiredError(humans)
}

func validateExplicitSponsor(board *domain.Board, sponsor string) (string, error) {
	// Try direct ID lookup first
	if u, ok := board.Users[sponsor]; ok {
		if u.Type != domain.IdentityHuman {
			return "", fmt.Errorf("sponsor %q is not a human identity", u.Name)
		}
		return sponsor, nil
	}
	// Fall back to name resolution (users pass names like --sponsor niladri)
	resolved, err := board.ResolveUserID(sponsor)
	if err != nil {
		return "", fmt.Errorf("unknown sponsor %q: %w", sponsor, err)
	}
	u := board.Users[resolved]
	if u.Type != domain.IdentityHuman {
		return "", fmt.Errorf("sponsor %q is not a human identity", u.Name)
	}
	return resolved, nil
}

func sponsorRequiredError(humans []*domain.Identity) error {
	names := make([]string, len(humans))
	for i, h := range humans {
		names[i] = h.Name
	}
	sort.Strings(names)
	return fmt.Errorf("board has %d humans. Specify --sponsor: %s", len(humans), strings.Join(names, ", "))
}

func humanUsers(board *domain.Board) []*domain.Identity {
	var humans []*domain.Identity
	for _, u := range board.Users {
		if u.Type == domain.IdentityHuman {
			humans = append(humans, u)
		}
	}
	return humans
}
