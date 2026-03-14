"use client";

type PixelLogoSize = "sm" | "md" | "lg";

interface PixelLogoProps {
  size?: PixelLogoSize;
}

const SIZE_CONFIG: Record<PixelLogoSize, { cell: number; gap: number }> = {
  sm: { cell: 3, gap: 0.5 },
  md: { cell: 4, gap: 1 },
  lg: { cell: 6, gap: 1 },
};

const COLORS = {
  border: "#7aa2f7",
  purpleHeader: "#bb9af7",
  blueHeader: "#7aa2f7",
  greenHeader: "#9ece6a",
  divider: "#292e42",
  dark: "#161b22",
  purpleCard1: "#bb9af7",
  purpleCard2: "#c4b5fd",
  blueCard1: "#7aa2f7",
  blueCard2: "#93c5fd",
  greenCard1: "#9ece6a",
  greenCard2: "#bbf7d0",
  amberCard1: "#e0af68",
  amberCard2: "#fcd34d",
  grayCard1: "#8b949e",
  grayCard2: "#c9d1d9",
} as const;

const B = COLORS.border;
const P = COLORS.purpleHeader;
const G = COLORS.greenHeader;
const D = COLORS.divider;
const X = COLORS.dark;

// 15 columns x 12 rows
const GRID: string[][] = [
  // Row 0: top border
  [B, B, B, B, B, B, B, B, B, B, B, B, B, B, B],
  // Row 1: column headers
  [B, P, P, P, D, B, B, B, D, G, G, G, G, G, B],
  // Row 2: gap
  [B, X, X, X, D, X, X, X, D, X, X, X, X, X, B],
  // Row 3-4: first cards
  [B, COLORS.purpleCard1, COLORS.purpleCard1, COLORS.purpleCard2, D, COLORS.blueCard1, COLORS.blueCard1, COLORS.blueCard2, D, COLORS.greenCard1, COLORS.greenCard1, COLORS.greenCard1, COLORS.greenCard1, COLORS.greenCard2, B],
  [B, COLORS.purpleCard1, COLORS.purpleCard1, COLORS.purpleCard2, D, COLORS.blueCard1, COLORS.blueCard1, COLORS.blueCard2, D, COLORS.greenCard1, COLORS.greenCard1, COLORS.greenCard1, COLORS.greenCard1, COLORS.greenCard2, B],
  // Row 5: gap
  [B, X, X, X, D, X, X, X, D, X, X, X, X, X, B],
  // Row 6-7: second cards
  [B, COLORS.amberCard1, COLORS.amberCard1, COLORS.amberCard2, D, COLORS.blueCard1, COLORS.blueCard1, COLORS.blueCard2, D, X, X, X, X, X, B],
  [B, COLORS.amberCard1, COLORS.amberCard1, COLORS.amberCard2, D, COLORS.blueCard1, COLORS.blueCard1, COLORS.blueCard2, D, X, X, X, X, X, B],
  // Row 8: gap
  [B, X, X, X, D, X, X, X, D, X, X, X, X, X, B],
  // Row 9-10: third card (col1 only)
  [B, COLORS.grayCard1, COLORS.grayCard1, COLORS.grayCard2, D, X, X, X, D, X, X, X, X, X, B],
  [B, COLORS.grayCard1, COLORS.grayCard1, COLORS.grayCard2, D, X, X, X, D, X, X, X, X, X, B],
  // Row 11: bottom border
  [B, B, B, B, B, B, B, B, B, B, B, B, B, B, B],
];

export function PixelLogo({ size = "md" }: PixelLogoProps) {
  const { cell, gap } = SIZE_CONFIG[size];
  const cols = 15;
  const rows = 12;
  const totalWidth = cols * cell + (cols - 1) * gap;
  const totalHeight = rows * cell + (rows - 1) * gap;

  return (
    <div
      role="img"
      aria-label="Obeya kanban board logo"
      style={{
        display: "inline-grid",
        gridTemplateColumns: `repeat(${cols}, ${cell}px)`,
        gridTemplateRows: `repeat(${rows}, ${cell}px)`,
        gap: `${gap}px`,
        width: totalWidth,
        height: totalHeight,
      }}
    >
      {GRID.flatMap((row, ri) =>
        row.map((color, ci) => (
          <div
            key={`${ri}-${ci}`}
            style={{ backgroundColor: color, borderRadius: gap > 0.5 ? 1 : 0 }}
          />
        ))
      )}
    </div>
  );
}
