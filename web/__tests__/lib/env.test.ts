import { describe, it, expect, vi, beforeEach } from "vitest";

describe("env", () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
    vi.resetModules();
  });

  it("returns validated env when all vars present", async () => {
    vi.stubEnv("APPWRITE_ENDPOINT", "https://cloud.appwrite.io/v1");
    vi.stubEnv("APPWRITE_PROJECT_ID", "test-project");
    vi.stubEnv("APPWRITE_API_KEY", "test-key");
    vi.stubEnv("APPWRITE_DATABASE_ID", "obeya");
    vi.stubEnv("NEXT_PUBLIC_APP_URL", "http://localhost:3000");
    const { getEnv } = await import("@/lib/env");
    const env = getEnv();
    expect(env.APPWRITE_ENDPOINT).toBe("https://cloud.appwrite.io/v1");
    expect(env.APPWRITE_PROJECT_ID).toBe("test-project");
  });

  it("throws when required var is missing", async () => {
    vi.stubEnv("APPWRITE_ENDPOINT", "");
    vi.stubEnv("APPWRITE_PROJECT_ID", "");
    vi.stubEnv("APPWRITE_API_KEY", "");
    vi.stubEnv("APPWRITE_DATABASE_ID", "");
    const { getEnv } = await import("@/lib/env");
    expect(() => getEnv()).toThrow();
  });
});
