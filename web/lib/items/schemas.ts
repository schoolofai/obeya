import { z } from "zod";

export const createItemSchema = z.object({
  type: z.enum(["epic", "story", "task"]),
  title: z.string().min(1, "Title is required").max(500),
  description: z.string().max(50000).default(""),
  status: z.string().min(1).default("backlog"),
  priority: z.enum(["low", "medium", "high", "critical"]).default("medium"),
  parent_id: z.string().nullable().optional(),
  assignee_id: z.string().nullable().optional(),
  blocked_by: z.array(z.string()).default([]),
  tags: z.array(z.string()).default([]),
  project: z.string().nullable().optional(),
});

export const updateItemSchema = z.object({
  title: z.string().min(1).max(500).optional(),
  description: z.string().max(50000).optional(),
  priority: z.enum(["low", "medium", "high", "critical"]).optional(),
  parent_id: z.string().nullable().optional(),
  tags: z.array(z.string()).optional(),
  project: z.string().nullable().optional(),
});

export const listItemsQuerySchema = z.object({
  status: z.string().optional(),
  type: z.enum(["epic", "story", "task"]).optional(),
  assignee: z.string().optional(),
});
