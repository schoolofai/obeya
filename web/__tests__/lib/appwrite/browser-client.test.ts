import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("appwrite", () => {
  const mockSetEndpoint = vi.fn().mockReturnThis();
  const mockSetProject = vi.fn().mockReturnThis();

  function MockClient(this: any) {
    this.setEndpoint = mockSetEndpoint;
    this.setProject = mockSetProject;
    this.subscribe = vi.fn();
  }

  return {
    Client: MockClient,
  };
});

describe("browser-client", () => {
  beforeEach(() => {
    vi.resetModules();
    process.env.NEXT_PUBLIC_APPWRITE_ENDPOINT = "https://cloud.appwrite.io/v1";
    process.env.NEXT_PUBLIC_APPWRITE_PROJECT_ID = "test-project-id";
  });

  it("getBrowserClient returns a configured Appwrite Client", async () => {
    const { getBrowserClient } = await import("@/lib/appwrite/browser-client");
    const client = getBrowserClient();
    expect(client).toBeDefined();
    expect(client.setEndpoint).toBeDefined();
    expect(client.setProject).toBeDefined();
  });

  it("getBrowserClient returns the same singleton on repeated calls", async () => {
    const { getBrowserClient } = await import("@/lib/appwrite/browser-client");
    const client1 = getBrowserClient();
    const client2 = getBrowserClient();
    expect(client1).toBe(client2);
  });
});
