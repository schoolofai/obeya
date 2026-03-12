import { describe, it, expect } from "vitest";
import { generateToken, hashToken, verifyToken } from "@/lib/auth/tokens";

describe("generateToken", () => {
  it("returns a string prefixed with ob_tok_", () => {
    const token = generateToken();
    expect(token.startsWith("ob_tok_")).toBe(true);
  });

  it("returns a token longer than 20 characters", () => {
    const token = generateToken();
    expect(token.length).toBeGreaterThan(20);
  });

  it("returns unique tokens on each call", () => {
    const a = generateToken();
    const b = generateToken();
    expect(a).not.toBe(b);
  });
});

describe("hashToken", () => {
  it("returns a bcrypt hash string", async () => {
    const token = generateToken();
    const hash = await hashToken(token);
    expect(hash.startsWith("$2")).toBe(true);
    expect(hash.length).toBeGreaterThan(50);
  });
});

describe("verifyToken", () => {
  it("returns true for a matching token and hash", async () => {
    const token = generateToken();
    const hash = await hashToken(token);
    const result = await verifyToken(token, hash);
    expect(result).toBe(true);
  });

  it("returns false for a mismatched token", async () => {
    const token = generateToken();
    const hash = await hashToken(token);
    const result = await verifyToken("ob_tok_wrong", hash);
    expect(result).toBe(false);
  });
});
