import { describe, it, expect } from "vitest";
import { z } from "zod";
import { validateBody, validateParams } from "@/lib/validation";
import { AppError, ErrorCode } from "@/lib/errors";

function jsonRequest(body: unknown): Request {
  return new Request("http://localhost/test", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

function invalidJsonRequest(): Request {
  return new Request("http://localhost/test", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: "not-json{{{",
  });
}

const userSchema = z.object({
  name: z.string(),
  email: z.string().email(),
});

describe("validateBody()", () => {
  it("returns parsed data for valid JSON body", async () => {
    const req = jsonRequest({ name: "Alice", email: "alice@example.com" });
    const data = await validateBody(req, userSchema);
    expect(data).toEqual({ name: "Alice", email: "alice@example.com" });
  });

  it("throws VALIDATION_ERROR for invalid JSON", async () => {
    const req = invalidJsonRequest();
    await expect(validateBody(req, userSchema)).rejects.toThrow(AppError);
    try {
      await validateBody(invalidJsonRequest(), userSchema);
    } catch (err) {
      expect(err).toBeInstanceOf(AppError);
      const appErr = err as AppError;
      expect(appErr.code).toBe(ErrorCode.VALIDATION_ERROR);
      expect(appErr.message).toBe("Request body must be valid JSON");
    }
  });

  it("throws VALIDATION_ERROR when schema validation fails", async () => {
    const req = jsonRequest({ name: 123, email: "not-an-email" });
    await expect(validateBody(req, userSchema)).rejects.toThrow(AppError);
    try {
      await validateBody(
        jsonRequest({ name: 123, email: "not-an-email" }),
        userSchema
      );
    } catch (err) {
      const appErr = err as AppError;
      expect(appErr.code).toBe(ErrorCode.VALIDATION_ERROR);
      expect(appErr.message).toContain("Validation failed:");
    }
  });
});

const paramsSchema = z.object({
  orgSlug: z.string().min(1),
  boardId: z.string().uuid(),
});

describe("validateParams()", () => {
  it("returns parsed data for valid params", () => {
    const params = {
      orgSlug: "acme",
      boardId: "550e8400-e29b-41d4-a716-446655440000",
    };
    const data = validateParams(params, paramsSchema);
    expect(data).toEqual(params);
  });

  it("throws VALIDATION_ERROR for invalid params", () => {
    const params = { orgSlug: "", boardId: "not-a-uuid" };
    expect(() => validateParams(params, paramsSchema)).toThrow(AppError);
    try {
      validateParams(params, paramsSchema);
    } catch (err) {
      const appErr = err as AppError;
      expect(appErr.code).toBe(ErrorCode.VALIDATION_ERROR);
      expect(appErr.message).toContain("Invalid parameters:");
    }
  });
});
