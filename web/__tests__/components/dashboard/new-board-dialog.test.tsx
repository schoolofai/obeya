import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NewBoardDialog } from "@/components/dashboard/new-board-dialog";

describe("NewBoardDialog", () => {
  it("renders nothing when closed", () => {
    const { container } = render(
      <NewBoardDialog open={false} onClose={vi.fn()} onCreate={vi.fn()} />
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders form when open", () => {
    render(
      <NewBoardDialog open={true} onClose={vi.fn()} onCreate={vi.fn()} />
    );
    expect(screen.getByLabelText("Board Name")).toBeInTheDocument();
  });

  it("calls onCreate with board name on submit", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn();
    render(
      <NewBoardDialog open={true} onClose={vi.fn()} onCreate={onCreate} />
    );

    await user.type(screen.getByLabelText("Board Name"), "My New Board");
    await user.click(screen.getByRole("button", { name: "Create" }));

    expect(onCreate).toHaveBeenCalledWith("My New Board");
  });

  it("calls onClose when Cancel is clicked", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
      <NewBoardDialog open={true} onClose={onClose} onCreate={vi.fn()} />
    );

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("does not call onCreate when name is empty", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn();
    render(
      <NewBoardDialog open={true} onClose={vi.fn()} onCreate={onCreate} />
    );

    await user.click(screen.getByRole("button", { name: "Create" }));
    expect(onCreate).not.toHaveBeenCalled();
  });
});
