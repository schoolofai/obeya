import { z } from "zod";
import { AppError, ErrorCode } from "@/lib/errors";

export async function validateBody<T>(
  request: Request,
  schema: z.ZodType<T>
): Promise<T> {
  let raw: unknown;
  try {
    raw = await request.json();
  } catch {
    throw new AppError(
      ErrorCode.VALIDATION_ERROR,
      "Request body must be valid JSON"
    );
  }
  const result = schema.safeParse(raw);
  if (!result.success) {
    const details = result.error.issues
      .map((i) => `${i.path.join(".")}: ${i.message}`)
      .join("; ");
    throw new AppError(ErrorCode.VALIDATION_ERROR, `Validation failed: ${details}`);
  }
  return result.data;
}

export function validateParams<T>(
  params: Record<string, unknown>,
  schema: z.ZodType<T>
): T {
  const result = schema.safeParse(params);
  if (!result.success) {
    const details = result.error.issues
      .map((i) => `${i.path.join(".")}: ${i.message}`)
      .join("; ");
    throw new AppError(
      ErrorCode.VALIDATION_ERROR,
      `Invalid parameters: ${details}`
    );
  }
  return result.data;
}
