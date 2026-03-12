import { describe, it, expect } from "vitest";
import {
  serializeBoard,
  deserializeBoard,
  serializeColumns,
  deserializeColumns,
} from "@/lib/boards/serialize";

describe("serializeColumns", () => {
  it("converts column array to JSON string", () => {
    const columns = [
      { name: "todo", limit: 0 },
      { name: "doing", limit: 3 },
    ];
    const result = serializeColumns(columns);
    expect(result).toBe(JSON.stringify(columns));
  });
});

describe("deserializeColumns", () => {
  it("parses JSON string to column array", () => {
    const json = '[{"name":"todo","limit":0},{"name":"done","limit":0}]';
    const result = deserializeColumns(json);
    expect(result).toEqual([
      { name: "todo", limit: 0 },
      { name: "done", limit: 0 },
    ]);
  });

  it("returns empty array for empty string", () => {
    expect(deserializeColumns("")).toEqual([]);
  });
});

describe("serializeBoard", () => {
  it("converts board fields for Appwrite storage", () => {
    const board = {
      name: "My Board",
      columns: [{ name: "todo", limit: 0 }],
      display_map: { "1": "item-abc" },
      users: { agent1: { role: "worker" } },
      projects: {},
    };
    const result = serializeBoard(board);
    expect(result.columns).toBe(JSON.stringify(board.columns));
    expect(result.display_map).toBe(JSON.stringify(board.display_map));
    expect(result.users).toBe(JSON.stringify(board.users));
    expect(result.projects).toBe(JSON.stringify(board.projects));
    expect(result.name).toBe("My Board");
  });
});

describe("deserializeBoard", () => {
  it("parses Appwrite document back to board shape", () => {
    const doc = {
      $id: "board-1",
      name: "My Board",
      owner_id: "user-1",
      org_id: null,
      display_counter: 5,
      columns: '[{"name":"todo","limit":0}]',
      display_map: '{"1":"item-abc"}',
      users: '{"agent1":{"role":"worker"}}',
      projects: "{}",
      agent_role: "worker",
      version: 1,
      created_at: "2026-03-12T00:00:00.000Z",
      updated_at: "2026-03-12T00:00:00.000Z",
    };
    const result = deserializeBoard(doc);
    expect(result.id).toBe("board-1");
    expect(result.columns).toEqual([{ name: "todo", limit: 0 }]);
    expect(result.display_map).toEqual({ "1": "item-abc" });
    expect(result.users).toEqual({ agent1: { role: "worker" } });
  });
});
