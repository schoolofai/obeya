import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ReviewQueuePanel } from "@/components/board/review-queue-panel";
import type { BoardItem } from "@/lib/api-client";

function makeItem(overrides: Partial<BoardItem>): BoardItem {
  return {
    $id: "item-1",
    board_id: "board-1",
    display_num: 1,
    type: "task",
    title: "Test task",
    description: "",
    status: "done",
    priority: "medium",
    parent_id: null,
    assignee_id: "agent-1",
    blocked_by: [],
    tags: [],
    project: null,
    created_at: "2026-03-10T00:00:00Z",
    updated_at: "2026-03-11T00:00:00Z",
    ...overrides,
  };
}

const agentItems: BoardItem[] = [
  makeItem({
    $id: "item-high",
    display_num: 1,
    title: "High confidence task",
    confidence: 90,
    review_context: { purpose: "High conf" },
    human_review: null,
  }),
  makeItem({
    $id: "item-low",
    display_num: 2,
    title: "Low confidence task",
    confidence: 30,
    review_context: { purpose: "Low conf" },
    human_review: null,
  }),
  makeItem({
    $id: "item-mid",
    display_num: 3,
    title: "Medium confidence task",
    confidence: 60,
    review_context: { purpose: "Mid conf" },
    human_review: null,
  }),
];

describe("ReviewQueuePanel", () => {
  const noop = () => {};

  it("renders nothing when no items match review queue criteria", () => {
    const items = [makeItem({ status: "in-progress" })];
    const { container } = render(
      <ReviewQueuePanel items={items} onCardClick={noop} onMarkReviewed={noop} onHide={noop} />
    );
    expect(container.innerHTML).toBe("");
  });

  it("renders review queue header with count", () => {
    render(
      <ReviewQueuePanel items={agentItems} onCardClick={noop} onMarkReviewed={noop} onHide={noop} />
    );
    expect(screen.getByTestId("review-queue-panel")).toBeDefined();
    expect(screen.getByText("(3)")).toBeDefined();
  });

  it("sorts items by confidence ascending (lowest first)", () => {
    render(
      <ReviewQueuePanel items={agentItems} onCardClick={noop} onMarkReviewed={noop} onHide={noop} />
    );
    const cards = screen.getAllByText(/confidence task/);
    expect(cards[0].textContent).toBe("Low confidence task");
    expect(cards[1].textContent).toBe("Medium confidence task");
    expect(cards[2].textContent).toBe("High confidence task");
  });

  it("filters out hidden items", () => {
    const withHidden = [
      ...agentItems,
      makeItem({
        $id: "item-hidden",
        display_num: 4,
        title: "Hidden task",
        confidence: 10,
        review_context: { purpose: "Hidden" },
        human_review: { status: "hidden" },
      }),
    ];
    render(
      <ReviewQueuePanel items={withHidden} onCardClick={noop} onMarkReviewed={noop} onHide={noop} />
    );
    expect(screen.queryByText("Hidden task")).toBeNull();
  });

  it("filters out items without review_context", () => {
    const withoutContext = [
      ...agentItems,
      makeItem({
        $id: "no-ctx",
        display_num: 5,
        title: "No context task",
        review_context: null,
      }),
    ];
    render(
      <ReviewQueuePanel items={withoutContext} onCardClick={noop} onMarkReviewed={noop} onHide={noop} />
    );
    expect(screen.queryByText("No context task")).toBeNull();
  });

  it("calls onCardClick when card is clicked", () => {
    const onCardClick = vi.fn();
    render(
      <ReviewQueuePanel items={agentItems} onCardClick={onCardClick} onMarkReviewed={noop} onHide={noop} />
    );
    fireEvent.click(screen.getByText("Low confidence task"));
    expect(onCardClick).toHaveBeenCalledWith(
      expect.objectContaining({ $id: "item-low" })
    );
  });

  it("calls onMarkReviewed when Mark Reviewed button is clicked", () => {
    const onMarkReviewed = vi.fn();
    render(
      <ReviewQueuePanel items={[agentItems[0]]} onCardClick={noop} onMarkReviewed={onMarkReviewed} onHide={noop} />
    );
    fireEvent.click(screen.getByText("Mark Reviewed"));
    expect(onMarkReviewed).toHaveBeenCalledWith(
      expect.objectContaining({ $id: "item-high" })
    );
  });

  it("calls onHide when Hide button is clicked", () => {
    const onHide = vi.fn();
    render(
      <ReviewQueuePanel items={[agentItems[0]]} onCardClick={noop} onMarkReviewed={noop} onHide={onHide} />
    );
    fireEvent.click(screen.getByText("Hide"));
    expect(onHide).toHaveBeenCalledWith(
      expect.objectContaining({ $id: "item-high" })
    );
  });

  it("shows green checkmark for reviewed items", () => {
    const reviewed = [
      makeItem({
        $id: "reviewed-1",
        display_num: 10,
        title: "Reviewed task",
        confidence: 80,
        review_context: { purpose: "Done" },
        human_review: { status: "reviewed", reviewed_by: "niladri" },
      }),
    ];
    render(
      <ReviewQueuePanel items={reviewed} onCardClick={noop} onMarkReviewed={noop} onHide={noop} />
    );
    expect(screen.getByTestId("reviewed-checkmark")).toBeDefined();
  });

  it("shows sponsor on card", () => {
    const withSponsor = [
      makeItem({
        $id: "sponsored",
        display_num: 11,
        title: "Sponsored task",
        confidence: 50,
        sponsor: "niladri",
        review_context: { purpose: "Needs review" },
        human_review: null,
      }),
    ];
    render(
      <ReviewQueuePanel items={withSponsor} onCardClick={noop} onMarkReviewed={noop} onHide={noop} />
    );
    expect(screen.getByText("sponsor: @niladri")).toBeDefined();
  });

  it("treats null confidence items as lowest priority (first in list)", () => {
    const withNull = [
      makeItem({
        $id: "with-conf",
        display_num: 1,
        title: "Alpha task",
        confidence: 50,
        review_context: { purpose: "Has conf" },
        human_review: null,
      }),
      makeItem({
        $id: "no-conf",
        display_num: 2,
        title: "Beta task",
        confidence: null,
        review_context: { purpose: "No conf" },
        human_review: null,
      }),
    ];
    render(
      <ReviewQueuePanel items={withNull} onCardClick={noop} onMarkReviewed={noop} onHide={noop} />
    );
    const reviewCards = screen.getByTestId("review-queue-panel");
    const cardElements = reviewCards.querySelectorAll("[data-testid^='review-card-']");
    expect(cardElements[0].getAttribute("data-testid")).toBe("review-card-2");
    expect(cardElements[1].getAttribute("data-testid")).toBe("review-card-1");
  });

  it("does not call onMarkReviewed when clicking disabled Reviewed button", () => {
    const onMarkReviewed = vi.fn();
    const reviewed = [
      makeItem({
        $id: "reviewed-1",
        display_num: 10,
        title: "Already reviewed",
        confidence: 80,
        review_context: { purpose: "Done" },
        human_review: { status: "reviewed", reviewed_by: "niladri" },
      }),
    ];
    render(
      <ReviewQueuePanel items={reviewed} onCardClick={noop} onMarkReviewed={onMarkReviewed} onHide={noop} />
    );
    const btn = screen.getByText("Reviewed");
    fireEvent.click(btn);
    expect(onMarkReviewed).not.toHaveBeenCalled();
  });
});
