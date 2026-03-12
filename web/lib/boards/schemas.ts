import { z } from "zod";

export const columnSchema = z.object({
  name: z.string().min(1),
  limit: z.number().int().min(0).default(0),
});

export const createBoardSchema = z.object({
  name: z.string().min(1, "Board name is required").max(255),
  columns: z
    .array(columnSchema)
    .min(1, "At least one column is required")
    .default([
      { name: "backlog", limit: 0 },
      { name: "todo", limit: 0 },
      { name: "in-progress", limit: 3 },
      { name: "done", limit: 0 },
    ]),
  org_id: z.string().optional(),
  agent_role: z.string().max(50).default("worker"),
});

export const updateBoardSchema = z.object({
  name: z.string().min(1).max(255).optional(),
  columns: z.array(columnSchema).min(1).optional(),
  agent_role: z.string().max(50).optional(),
});
