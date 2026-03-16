import { z } from "zod";

export const fileChangeSchema = z.object({
  path: z.string().min(1),
  added: z.number().int().min(0),
  removed: z.number().int().min(0),
  diff: z.string().optional(),
});

export const testResultSchema = z.object({
  name: z.string().min(1),
  passed: z.boolean(),
});

export const proofItemSchema = z.object({
  check: z.string().min(1),
  status: z.enum(["pass", "fail", "warn"]),
  detail: z.string().optional(),
});

export const reviewContextSchema = z.object({
  purpose: z.string().min(1, "Purpose is required"),
  files_changed: z.array(fileChangeSchema).optional(),
  tests_written: z.array(testResultSchema).optional(),
  proof: z.array(proofItemSchema).optional(),
  reasoning: z.string().optional(),
  reproduce: z.array(z.string()).optional(),
});

export const completeItemSchema = z.object({
  confidence: z.number().int().min(0).max(100),
  review_context: reviewContextSchema,
});

export const reviewItemSchema = z.object({
  status: z.enum(["reviewed", "hidden"]),
});
