import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/appwrite/server", () => ({
  getDatabases: vi.fn(),
}));

vi.mock("@/lib/env", () => ({
  getEnv: () => ({
    APPWRITE_DATABASE_ID: "obeya",
  }),
}));

import { generateSlug, ensureUniqueSlug } from "@/lib/slugs";
import { getDatabases } from "@/lib/appwrite/server";

describe("generateSlug", () => {
  it("converts name to lowercase kebab-case", () => {
    expect(generateSlug("Hello World")).toBe("hello-world");
  });

  it("strips special characters", () => {
    expect(generateSlug("Hello, World!")).toBe("hello-world");
  });

  it("collapses multiple hyphens", () => {
    expect(generateSlug("Hello---World")).toBe("hello-world");
  });

  it("trims leading and trailing hyphens", () => {
    expect(generateSlug("-Hello World-")).toBe("hello-world");
  });

  it("transliterates ü to ue", () => {
    expect(generateSlug("München")).toBe("muenchen");
  });

  it("transliterates ö to oe", () => {
    expect(generateSlug("Döner")).toBe("doener");
  });

  it("transliterates ä to ae", () => {
    expect(generateSlug("Äpfel")).toBe("aepfel");
  });

  it("transliterates ß to ss", () => {
    expect(generateSlug("Straße")).toBe("strasse");
  });

  it("returns 'org' for empty input", () => {
    expect(generateSlug("")).toBe("org");
  });

  it("returns 'org' for input that results in empty string after processing", () => {
    expect(generateSlug("!!!")).toBe("org");
  });

  it("handles numbers in name", () => {
    expect(generateSlug("Team 42")).toBe("team-42");
  });
});

describe("ensureUniqueSlug", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns base slug when it is unique", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 0, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const result = await ensureUniqueSlug("my-org");
    expect(result).toBe("my-org");
    expect(mockDb.listDocuments).toHaveBeenCalledWith(
      "obeya",
      "orgs",
      expect.arrayContaining([])
    );
  });

  it("appends -1 when base slug is taken", async () => {
    const mockDb = {
      listDocuments: vi.fn()
        .mockResolvedValueOnce({ total: 1, documents: [{ slug: "my-org" }] })
        .mockResolvedValueOnce({ total: 0, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const result = await ensureUniqueSlug("my-org");
    expect(result).toBe("my-org-1");
  });

  it("appends -2 when -1 is also taken", async () => {
    const mockDb = {
      listDocuments: vi.fn()
        .mockResolvedValueOnce({ total: 1, documents: [{ slug: "my-org" }] })
        .mockResolvedValueOnce({ total: 1, documents: [{ slug: "my-org-1" }] })
        .mockResolvedValueOnce({ total: 0, documents: [] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    const result = await ensureUniqueSlug("my-org");
    expect(result).toBe("my-org-2");
  });

  it("throws after 20 attempts", async () => {
    const mockDb = {
      listDocuments: vi.fn().mockResolvedValue({ total: 1, documents: [{ slug: "taken" }] }),
    };
    vi.mocked(getDatabases).mockReturnValue(mockDb as any);

    await expect(ensureUniqueSlug("taken")).rejects.toThrow(
      "Could not generate unique slug after 20 attempts"
    );
  });
});
