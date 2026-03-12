import { describe, it, expect, vi, beforeEach } from "vitest";
import { apiClient, ApiClientError } from "@/lib/api-client";

const mockFetch = vi.fn();

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch);
  mockFetch.mockReset();
});

function makeFetchResponse(body: object, status = 200) {
  return Promise.resolve({
    status,
    ok: status >= 200 && status < 300,
    json: () => Promise.resolve(body),
  });
}

describe("apiClient.get", () => {
  it("returns data on successful GET", async () => {
    mockFetch.mockReturnValue(
      makeFetchResponse({ ok: true, data: { id: "1", name: "Test Board" } })
    );

    const result = await apiClient.get<{ id: string; name: string }>(
      "/api/boards/1"
    );

    expect(result).toEqual({ id: "1", name: "Test Board" });
    expect(mockFetch).toHaveBeenCalledWith("/api/boards/1", {
      method: "GET",
      headers: { "Content-Type": "application/json" },
    });
  });
});

describe("apiClient.post", () => {
  it("sends POST with body and returns data", async () => {
    mockFetch.mockReturnValue(
      makeFetchResponse({ ok: true, data: { id: "2", name: "New Board" } })
    );

    const result = await apiClient.post<{ id: string; name: string }>(
      "/api/boards",
      { name: "New Board" }
    );

    expect(result).toEqual({ id: "2", name: "New Board" });
    expect(mockFetch).toHaveBeenCalledWith("/api/boards", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: "New Board" }),
    });
  });
});

describe("apiClient.patch", () => {
  it("sends PATCH with body and returns data", async () => {
    mockFetch.mockReturnValue(
      makeFetchResponse({ ok: true, data: { id: "1", name: "Updated" } })
    );

    const result = await apiClient.patch<{ id: string; name: string }>(
      "/api/boards/1",
      { name: "Updated" }
    );

    expect(result).toEqual({ id: "1", name: "Updated" });
    expect(mockFetch).toHaveBeenCalledWith("/api/boards/1", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: "Updated" }),
    });
  });
});

describe("apiClient.delete", () => {
  it("sends DELETE and returns data", async () => {
    mockFetch.mockReturnValue(
      makeFetchResponse({ ok: true, data: { deleted: true } })
    );

    const result = await apiClient.delete<{ deleted: boolean }>(
      "/api/boards/1"
    );

    expect(result).toEqual({ deleted: true });
    expect(mockFetch).toHaveBeenCalledWith("/api/boards/1", {
      method: "DELETE",
      headers: { "Content-Type": "application/json" },
    });
  });
});

describe("ApiClientError", () => {
  it("throws ApiClientError with code, message, statusCode on 404", async () => {
    mockFetch.mockReturnValue(
      makeFetchResponse(
        {
          ok: false,
          error: { code: "NOT_FOUND", message: "Board not found" },
        },
        404
      )
    );

    await expect(apiClient.get("/api/boards/999")).rejects.toThrow(
      ApiClientError
    );

    try {
      await apiClient.get("/api/boards/999");
    } catch (err) {
      expect(err).toBeInstanceOf(ApiClientError);
      const apiErr = err as ApiClientError;
      expect(apiErr.code).toBe("NOT_FOUND");
      expect(apiErr.message).toBe("Board not found");
      expect(apiErr.statusCode).toBe(404);
    }
  });

  it("throws on network failure", async () => {
    mockFetch.mockRejectedValue(new Error("Network error"));

    await expect(apiClient.get("/api/boards/1")).rejects.toThrow(
      "Network error"
    );
  });
});
