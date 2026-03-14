import { describe, it, expect } from "vitest";
import {
  buildBoardPermissions,
  buildItemPermissions,
  type BoardMember,
} from "@/lib/appwrite/doc-permissions";

describe("buildBoardPermissions", () => {
  it("grants owner read, update, delete", () => {
    const perms = buildBoardPermissions("owner-1", []);
    expect(perms).toContain('read("user:owner-1")');
    expect(perms).toContain('update("user:owner-1")');
    expect(perms).toContain('delete("user:owner-1")');
  });

  it("grants viewer members read only", () => {
    const members: BoardMember[] = [{ userId: "viewer-1", role: "viewer" }];
    const perms = buildBoardPermissions("owner-1", members);
    expect(perms).toContain('read("user:viewer-1")');
    expect(perms).not.toContain('update("user:viewer-1")');
  });

  it("grants editor members read and update", () => {
    const members: BoardMember[] = [{ userId: "editor-1", role: "editor" }];
    const perms = buildBoardPermissions("owner-1", members);
    expect(perms).toContain('read("user:editor-1")');
    expect(perms).toContain('update("user:editor-1")');
  });

  it("deduplicates when owner is also in members list", () => {
    const members: BoardMember[] = [{ userId: "owner-1", role: "owner" }];
    const perms = buildBoardPermissions("owner-1", members);
    const readCount = perms.filter(
      (p) => p === 'read("user:owner-1")',
    ).length;
    expect(readCount).toBe(1);
  });

  it("handles multiple members", () => {
    const members: BoardMember[] = [
      { userId: "v1", role: "viewer" },
      { userId: "e1", role: "editor" },
      { userId: "o1", role: "owner" },
    ];
    const perms = buildBoardPermissions("owner-1", members);
    expect(perms).toContain('read("user:v1")');
    expect(perms).toContain('read("user:e1")');
    expect(perms).toContain('update("user:e1")');
    expect(perms).toContain('read("user:o1")');
    expect(perms).toContain('update("user:o1")');
  });
});

describe("buildItemPermissions", () => {
  it("grants owner read, update, delete", () => {
    const perms = buildItemPermissions("owner-1", []);
    expect(perms).toContain('read("user:owner-1")');
    expect(perms).toContain('update("user:owner-1")');
    expect(perms).toContain('delete("user:owner-1")');
  });

  it("grants all members read for realtime", () => {
    const members: BoardMember[] = [
      { userId: "v1", role: "viewer" },
      { userId: "e1", role: "editor" },
    ];
    const perms = buildItemPermissions("owner-1", members);
    expect(perms).toContain('read("user:v1")');
    expect(perms).toContain('read("user:e1")');
  });
});
