import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ConfidenceGauge } from "@/components/board/confidence-gauge";

describe("ConfidenceGauge", () => {
  it("renders percentage text", () => {
    render(<ConfidenceGauge confidence={45} />);
    expect(screen.getByText("45%")).toBeDefined();
  });

  it("shows LOW label for confidence <= 50", () => {
    render(<ConfidenceGauge confidence={30} />);
    expect(screen.getByText("LOW")).toBeDefined();
  });

  it("does not show LOW label for confidence > 50", () => {
    render(<ConfidenceGauge confidence={65} />);
    expect(screen.queryByText("LOW")).toBeNull();
  });

  it("uses red color for low confidence", () => {
    render(<ConfidenceGauge confidence={25} />);
    const gauge = screen.getByTestId("confidence-gauge");
    expect(gauge.innerHTML).toContain("text-red-400");
  });

  it("uses yellow color for medium confidence", () => {
    render(<ConfidenceGauge confidence={60} />);
    const gauge = screen.getByTestId("confidence-gauge");
    expect(gauge.innerHTML).toContain("text-yellow-400");
  });

  it("uses green color for high confidence", () => {
    render(<ConfidenceGauge confidence={85} />);
    const gauge = screen.getByTestId("confidence-gauge");
    expect(gauge.innerHTML).toContain("text-green-400");
  });

  it("sets progress bar width based on confidence", () => {
    render(<ConfidenceGauge confidence={70} />);
    const gauge = screen.getByTestId("confidence-gauge");
    expect(gauge.innerHTML).toContain("width: 70%");
  });

  it("renders 0% confidence correctly", () => {
    render(<ConfidenceGauge confidence={0} />);
    expect(screen.getByText("0%")).toBeDefined();
    expect(screen.getByText("LOW")).toBeDefined();
  });

  it("renders 100% confidence correctly", () => {
    render(<ConfidenceGauge confidence={100} />);
    expect(screen.getByText("100%")).toBeDefined();
  });

  it("boundary: 50 is red/LOW", () => {
    render(<ConfidenceGauge confidence={50} />);
    const gauge = screen.getByTestId("confidence-gauge");
    expect(gauge.innerHTML).toContain("text-red-400");
    expect(screen.getByText("LOW")).toBeDefined();
  });

  it("boundary: 51 is yellow, no LOW label", () => {
    render(<ConfidenceGauge confidence={51} />);
    const gauge = screen.getByTestId("confidence-gauge");
    expect(gauge.innerHTML).toContain("text-yellow-400");
    expect(screen.queryByText("LOW")).toBeNull();
  });

  it("boundary: 75 is yellow", () => {
    render(<ConfidenceGauge confidence={75} />);
    const gauge = screen.getByTestId("confidence-gauge");
    expect(gauge.innerHTML).toContain("text-yellow-400");
  });

  it("boundary: 76 is green", () => {
    render(<ConfidenceGauge confidence={76} />);
    const gauge = screen.getByTestId("confidence-gauge");
    expect(gauge.innerHTML).toContain("text-green-400");
  });
});
