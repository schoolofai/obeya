package tui

import (
	"sort"

	"github.com/niladribose/obeya/internal/domain"
)

// dagNode represents a positioned item in the DAG.
type dagNode struct {
	item    *domain.Item
	id      string
	gridCol int // column in the layout grid (0 = root)
	gridRow int // row within the lane
	x       int // character x-offset on the canvas
	y       int // character y-offset on the canvas
	w       int // rendered width in chars
	h       int // rendered height in chars
}

// dagEdge connects two nodes.
type dagEdge struct {
	fromIdx  int
	toIdx    int
	edgeKind string // "parent" or "blocker"
}

// dagLane groups nodes under a top-level epic (or orphan group).
type dagLane struct {
	label    string
	epicID   string   // "" for orphan lane
	nodeIdxs []int    // indices into the nodes slice
	yStart   int      // canvas y where this lane begins
	height   int      // total height of this lane on canvas
}

// dagGraph holds the complete positioned DAG.
type dagGraph struct {
	nodes  []dagNode
	edges  []dagEdge
	lanes  []dagLane
	width  int // total canvas width
	height int // total canvas height
}

const (
	dagNodeW    = 22 // width of a DAG node box
	dagNodeH    = 5  // height of a DAG node box
	dagGapX     = 6  // horizontal gap between columns (for arrows)
	dagGapY     = 1  // vertical gap between rows within a lane
	dagLaneGapY = 2  // vertical gap between lanes
)

// buildDAGGraph constructs a positioned DAG from a board.
func buildDAGGraph(board *domain.Board) dagGraph {
	if board == nil || len(board.Items) == 0 {
		return dagGraph{}
	}

	// Index: children by parent, items by ID
	childrenOf := map[string][]*domain.Item{}
	for _, item := range board.Items {
		if item.ParentID != "" {
			childrenOf[item.ParentID] = append(childrenOf[item.ParentID], item)
		}
	}

	// Sort children by display number for deterministic layout
	for k := range childrenOf {
		sort.Slice(childrenOf[k], func(i, j int) bool {
			return childrenOf[k][i].DisplayNum < childrenOf[k][j].DisplayNum
		})
	}

	// Find top-level epics and orphans
	var epics []*domain.Item
	var orphans []*domain.Item
	for _, item := range board.Items {
		if item.Type == domain.ItemTypeEpic && item.ParentID == "" {
			epics = append(epics, item)
		} else if item.ParentID == "" && item.Type != domain.ItemTypeEpic {
			orphans = append(orphans, item)
		}
	}
	sort.Slice(epics, func(i, j int) bool {
		return epics[i].DisplayNum < epics[j].DisplayNum
	})
	sort.Slice(orphans, func(i, j int) bool {
		return orphans[i].DisplayNum < orphans[j].DisplayNum
	})

	var g dagGraph
	nodeIndex := map[string]int{} // item ID -> index in g.nodes

	addNode := func(item *domain.Item, gridCol, gridRow int) int {
		idx := len(g.nodes)
		g.nodes = append(g.nodes, dagNode{
			item:    item,
			id:      item.ID,
			gridCol: gridCol,
			gridRow: gridRow,
			w:       dagNodeW,
			h:       dagNodeH,
		})
		nodeIndex[item.ID] = idx
		return idx
	}

	// Build a blockedBy lookup within siblings
	// If child B is in child A's BlockedBy, B depends on A sequentially
	isBlockedBySibling := func(child *domain.Item, siblings []*domain.Item) *domain.Item {
		sibIDs := map[string]*domain.Item{}
		for _, s := range siblings {
			sibIDs[s.ID] = s
		}
		for _, bid := range child.BlockedBy {
			if blocker, ok := sibIDs[bid]; ok {
				return blocker
			}
		}
		return nil
	}

	// Layout a set of children under a parent, assigning grid positions.
	// Independent children get the same gridCol; blocked ones get gridCol+1.
	layoutChildren := func(children []*domain.Item, baseCol int) ([]int, int) {
		if len(children) == 0 {
			return nil, 0
		}

		// Determine column assignments via dependency chains
		colAssign := map[string]int{}
		placed := map[string]bool{}
		maxCol := baseCol

		// Place items with no sibling blockers first
		row := 0
		var queue []*domain.Item
		for _, child := range children {
			blocker := isBlockedBySibling(child, children)
			if blocker == nil {
				colAssign[child.ID] = baseCol
				placed[child.ID] = true
				row++
			} else {
				queue = append(queue, child)
			}
		}

		// Place blocked items after their blocker
		for iterations := 0; iterations < len(children) && len(queue) > 0; iterations++ {
			var remaining []*domain.Item
			for _, child := range queue {
				blocker := isBlockedBySibling(child, children)
				if blocker != nil && placed[blocker.ID] {
					col := colAssign[blocker.ID] + 1
					colAssign[child.ID] = col
					placed[child.ID] = true
					if col > maxCol {
						maxCol = col
					}
				} else {
					remaining = append(remaining, child)
				}
			}
			queue = remaining
		}
		// Any unplaced items (circular deps) go to baseCol
		for _, child := range queue {
			colAssign[child.ID] = baseCol
		}

		// Group children by column, assign rows within each column
		type colGroup struct {
			col      int
			children []*domain.Item
		}
		colGroups := map[int]*colGroup{}
		for _, child := range children {
			c := colAssign[child.ID]
			cg, ok := colGroups[c]
			if !ok {
				cg = &colGroup{col: c}
				colGroups[c] = cg
			}
			cg.children = append(cg.children, child)
		}

		// Now assign gridRow — items at same col get stacked vertically
		rowCounters := map[int]int{}
		maxRow := 0
		var idxs []int
		for _, child := range children {
			c := colAssign[child.ID]
			r := rowCounters[c]
			rowCounters[c]++
			idx := addNode(child, c, r)
			idxs = append(idxs, idx)
			if r > maxRow {
				maxRow = r
			}
		}

		return idxs, maxRow + 1
	}

	// Build lanes
	curY := 0

	// Epic lanes
	for _, epic := range epics {
		lane := dagLane{
			label:  epic.Title,
			epicID: epic.ID,
		}

		// Add the epic node at column 0
		epicIdx := addNode(epic, 0, 0)
		lane.nodeIdxs = append(lane.nodeIdxs, epicIdx)

		// Get all descendants (direct children only for now)
		children := childrenOf[epic.ID]

		// Also include grandchildren (stories -> tasks)
		var allChildren []*domain.Item
		allChildren = append(allChildren, children...)
		for _, child := range children {
			allChildren = append(allChildren, childrenOf[child.ID]...)
		}

		childIdxs, rowCount := layoutChildren(allChildren, 1)
		lane.nodeIdxs = append(lane.nodeIdxs, childIdxs...)

		// Add edges: parent -> child
		for _, cidx := range childIdxs {
			child := g.nodes[cidx].item
			if child.ParentID == epic.ID {
				g.edges = append(g.edges, dagEdge{
					fromIdx:  epicIdx,
					toIdx:    cidx,
					edgeKind: "parent",
				})
			} else if pidx, ok := nodeIndex[child.ParentID]; ok {
				g.edges = append(g.edges, dagEdge{
					fromIdx:  pidx,
					toIdx:    cidx,
					edgeKind: "parent",
				})
			}
			// Add blocker edges
			for _, bid := range child.BlockedBy {
				if bidx, ok := nodeIndex[bid]; ok {
					g.edges = append(g.edges, dagEdge{
						fromIdx:  bidx,
						toIdx:    cidx,
						edgeKind: "blocker",
					})
				}
			}
		}

		// The epic itself sits at row 0; children may need more rows
		epicRow := 0
		if rowCount > 1 {
			epicRow = rowCount / 2 // center epic vertically relative to children
		}
		g.nodes[epicIdx].gridRow = epicRow

		laneHeight := rowCount
		if laneHeight < 1 {
			laneHeight = 1
		}
		lane.yStart = curY
		lane.height = laneHeight
		g.lanes = append(g.lanes, lane)

		curY += laneHeight*(dagNodeH+dagGapY) + dagLaneGapY
	}

	// Orphan lane
	if len(orphans) > 0 {
		lane := dagLane{
			label: "Independent Tasks",
		}

		// Group orphans by blocker chains
		orphanIdxs, rowCount := layoutChildren(orphans, 0)
		lane.nodeIdxs = append(lane.nodeIdxs, orphanIdxs...)

		// Add blocker edges for orphans
		for _, oidx := range orphanIdxs {
			orphan := g.nodes[oidx].item
			for _, bid := range orphan.BlockedBy {
				if bidx, ok := nodeIndex[bid]; ok {
					g.edges = append(g.edges, dagEdge{
						fromIdx:  bidx,
						toIdx:    oidx,
						edgeKind: "blocker",
					})
				}
			}
		}

		laneHeight := rowCount
		if laneHeight < 1 {
			laneHeight = 1
		}
		lane.yStart = curY
		lane.height = laneHeight
		g.lanes = append(g.lanes, lane)

		curY += laneHeight*(dagNodeH+dagGapY) + dagLaneGapY
	}

	// Compute pixel (char) positions from grid positions
	maxGridCol := 0
	for i := range g.nodes {
		if g.nodes[i].gridCol > maxGridCol {
			maxGridCol = g.nodes[i].gridCol
		}
	}

	for i := range g.nodes {
		n := &g.nodes[i]
		n.x = n.gridCol * (dagNodeW + dagGapX)

		// Find this node's lane to compute y offset
		for _, lane := range g.lanes {
			for _, idx := range lane.nodeIdxs {
				if idx == i {
					n.y = lane.yStart*1 + n.gridRow*(dagNodeH+dagGapY)
					// Adjust: lane.yStart is in canvas-line units already for multi-lane
					// We set it as pixel offset during lane building
				}
			}
		}
	}

	// Recompute y using lane yStart (which is already in char units)
	for li := range g.lanes {
		for _, ni := range g.lanes[li].nodeIdxs {
			g.nodes[ni].y = g.lanes[li].yStart + g.nodes[ni].gridRow*(dagNodeH+dagGapY)
		}
	}

	g.width = (maxGridCol + 1) * (dagNodeW + dagGapX)
	g.height = curY
	if g.height < dagNodeH {
		g.height = dagNodeH
	}

	return g
}

// firstInProgressNode returns the index of the leftmost in-progress node, or -1.
func (g *dagGraph) firstInProgressNode() int {
	bestIdx := -1
	bestX := int(^uint(0) >> 1) // max int
	for i, n := range g.nodes {
		if n.item.Status == "in-progress" && n.x < bestX {
			bestX = n.x
			bestIdx = i
		}
	}
	return bestIdx
}
