import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import {
  PastReviewsTree,
  buildReviewTree,
  type ReviewTreeItem,
} from "@/components/board/past-reviews-tree";

const makeItem = (overrides: Partial<ReviewTreeItem> & { id: string; display_num: number; title: string }): ReviewTreeItem => ({
  type: "task",
  parent_id: null,
  confidence: null,
  human_review: null,
  ...overrides,
});

const sampleItems: ReviewTreeItem[] = [
  makeItem({ id: "epic-1", display_num: 10, title: "Auth Rewrite", type: "epic" }),
  makeItem({
    id: "story-1", display_num: 15, title: "Session management", type: "story",
    parent_id: "epic-1",
  }),
  makeItem({
    id: "task-1", display_num: 34, title: "Refactor middleware",
    parent_id: "story-1", confidence: 45,
    human_review: { status: "reviewed", reviewed_by: "niladri", reviewed_at: "2026-03-15T10:00:00Z" },
  }),
  makeItem({
    id: "task-2", display_num: 35, title: "Update session store",
    parent_id: "story-1", confidence: 80,
    human_review: { status: "hidden" },
  }),
  makeItem({
    id: "orphan-1", display_num: 44, title: "Orphan task",
    confidence: 60,
    human_review: { status: "reviewed", reviewed_by: "niladri", reviewed_at: "2026-03-15T12:00:00Z" },
  }),
];

describe("buildReviewTree", () => {
  it("creates tree from flat items with parent hierarchy", () => {
    const tree = buildReviewTree(sampleItems);
    expect(tree.length).toBe(2);
    expect(tree[0].item.display_num).toBe(10);
    expect(tree[0].isStructural).toBe(true);
    expect(tree[0].children[0].item.display_num).toBe(15);
    expect(tree[0].children[0].isStructural).toBe(true);
    expect(tree[0].children[0].children.length).toBe(2);
  });

  it("marks non-reviewed ancestors as structural", () => {
    const tree = buildReviewTree(sampleItems);
    expect(tree[0].isStructural).toBe(true);
    expect(tree[0].children[0].isStructural).toBe(true);
    expect(tree[0].children[0].children[0].isStructural).toBe(false);
  });

  it("sorts by display_num", () => {
    const tree = buildReviewTree(sampleItems);
    expect(tree[0].item.display_num).toBe(10);
    expect(tree[1].item.display_num).toBe(44);
  });

  it("returns empty tree when no items have reviews", () => {
    const items = [
      makeItem({ id: "a", display_num: 1, title: "No review" }),
    ];
    const tree = buildReviewTree(items);
    expect(tree.length).toBe(0);
  });
});

describe("PastReviewsTree", () => {
  it("renders the tree with reviewed items", () => {
    render(
      <PastReviewsTree items={sampleItems} onSelect={() => {}} onClose={() => {}} />
    );
    expect(screen.getByTestId("past-reviews-tree")).toBeDefined();
    expect(screen.getByText("Past Reviews")).toBeDefined();
    expect(screen.getByText("Refactor middleware")).toBeDefined();
    expect(screen.getByText("Update session store")).toBeDefined();
    expect(screen.getByText("Orphan task")).toBeDefined();
  });

  it("shows structural ancestors with lower opacity", () => {
    render(
      <PastReviewsTree items={sampleItems} onSelect={() => {}} onClose={() => {}} />
    );
    const epicNode = screen.getByTestId("tree-node-10");
    expect(epicNode.className).toContain("opacity-50");
  });

  it("calls onSelect when item is clicked", () => {
    const onSelect = vi.fn();
    render(
      <PastReviewsTree items={sampleItems} onSelect={onSelect} onClose={() => {}} />
    );
    fireEvent.click(screen.getByText("Refactor middleware"));
    expect(onSelect).toHaveBeenCalledWith("task-1");
  });

  it("calls onClose when Esc button is clicked", () => {
    const onClose = vi.fn();
    render(
      <PastReviewsTree items={sampleItems} onSelect={() => {}} onClose={onClose} />
    );
    fireEvent.click(screen.getByText("Esc"));
    expect(onClose).toHaveBeenCalled();
  });

  it("shows empty state when no items have reviews", () => {
    const noReviews = [
      makeItem({ id: "a", display_num: 1, title: "No review" }),
    ];
    render(
      <PastReviewsTree items={noReviews} onSelect={() => {}} onClose={() => {}} />
    );
    expect(screen.getByText("No reviewed items yet")).toBeDefined();
  });

  it("shows confidence gauge for reviewed items", () => {
    render(
      <PastReviewsTree items={sampleItems} onSelect={() => {}} onClose={() => {}} />
    );
    expect(screen.getByText("45%")).toBeDefined();
  });

  it("collapses parent node and hides children", () => {
    render(
      <PastReviewsTree items={sampleItems} onSelect={() => {}} onClose={() => {}} />
    );
    // Children visible initially
    expect(screen.getByText("Refactor middleware")).toBeDefined();
    expect(screen.getByText("Update session store")).toBeDefined();

    // Click collapse on epic node
    const epicNode = screen.getByTestId("tree-node-10");
    const collapseBtn = epicNode.querySelector("button[aria-label='Collapse']");
    expect(collapseBtn).not.toBeNull();
    fireEvent.click(collapseBtn!);

    // Children should be hidden
    expect(screen.queryByText("Refactor middleware")).toBeNull();
    expect(screen.queryByText("Update session store")).toBeNull();

    // Orphan task still visible
    expect(screen.getByText("Orphan task")).toBeDefined();
  });

  it("expands collapsed parent node to show children again", () => {
    render(
      <PastReviewsTree items={sampleItems} onSelect={() => {}} onClose={() => {}} />
    );
    const epicNode = screen.getByTestId("tree-node-10");
    const collapseBtn = epicNode.querySelector("button[aria-label='Collapse']");
    fireEvent.click(collapseBtn!);

    // Now expand
    const expandBtn = epicNode.querySelector("button[aria-label='Expand']");
    expect(expandBtn).not.toBeNull();
    fireEvent.click(expandBtn!);

    // Children visible again
    expect(screen.getByText("Refactor middleware")).toBeDefined();
  });
});
