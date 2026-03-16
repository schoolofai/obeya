import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { DiffsViewer } from "@/components/board/diffs-viewer";
import type { FileChange } from "@/lib/types";

const filesWithDiffs: FileChange[] = [
  {
    path: "auth/middleware.go",
    added: 82,
    removed: 41,
    diff: `@@ -15,8 +15,12 @@ func AuthMiddleware(...)
-    cookie := r.Cookie("session")
+    token := r.Header.Get("Authorization")`,
  },
  {
    path: "auth/session.go",
    added: 15,
    removed: 8,
    diff: `@@ -42,4 +42,11 @@ func NewSessionStore(...)
-    return &CookieStore{}
+    return &JWTStore{}`,
  },
];

describe("DiffsViewer", () => {
  it("renders file headers with stats", () => {
    render(<DiffsViewer files={filesWithDiffs} />);
    expect(screen.getByTestId("diffs-viewer")).toBeDefined();
    expect(screen.getByTestId("file-diff-auth/middleware.go")).toBeDefined();
    expect(screen.getByTestId("file-diff-auth/session.go")).toBeDefined();
  });

  it("shows added/removed counts", () => {
    render(<DiffsViewer files={[filesWithDiffs[0]]} />);
    expect(screen.getByText("+82")).toBeDefined();
    expect(screen.getByText("-41")).toBeDefined();
  });

  it("renders diff lines with correct coloring", () => {
    render(<DiffsViewer files={[filesWithDiffs[0]]} />);
    const viewer = screen.getByTestId("diffs-viewer");
    expect(viewer.innerHTML).toContain("text-cyan-400");
    expect(viewer.innerHTML).toContain("text-red-400");
    expect(viewer.innerHTML).toContain("text-green-400");
  });

  it("shows 'No diffs available' when no files have diffs", () => {
    const noDiffs: FileChange[] = [
      { path: "foo.go", added: 1, removed: 0 },
    ];
    render(<DiffsViewer files={noDiffs} />);
    expect(screen.getByText("No diffs available")).toBeDefined();
  });

  it("shows file sidebar when multiple files have diffs", () => {
    render(<DiffsViewer files={filesWithDiffs} />);
    expect(screen.getByTestId("diff-file-sidebar")).toBeDefined();
  });

  it("does not show file sidebar for single file", () => {
    render(<DiffsViewer files={[filesWithDiffs[0]]} />);
    expect(screen.queryByTestId("diff-file-sidebar")).toBeNull();
  });

  it("filters out files without diffs from display", () => {
    const mixed: FileChange[] = [
      ...filesWithDiffs,
      { path: "readme.md", added: 1, removed: 0 },
    ];
    render(<DiffsViewer files={mixed} />);
    expect(screen.queryByTestId("file-diff-readme.md")).toBeNull();
  });
});
