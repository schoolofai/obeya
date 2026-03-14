import { describe, it, expect } from "vitest";
import { POST } from "@/app/api/auth/logout/route";

function postRequest(): Request {
  return new Request("http://localhost/api/auth/logout", {
    method: "POST",
  });
}

describe("POST /api/auth/logout", () => {
  it("returns 200 with ok response", async () => {
    const res = await POST(postRequest());
    const body = await res.json();

    expect(res.status).toBe(200);
    expect(body).toEqual({ ok: true });
  });

  it("clears the obeya_session cookie", async () => {
    const res = await POST(postRequest());
    const cookie = res.headers.get("set-cookie");

    expect(cookie).toBeTruthy();
    expect(cookie).toContain("obeya_session=");
    expect(cookie).toContain("Max-Age=0");
    expect(cookie).toContain("Path=/");
  });
});
