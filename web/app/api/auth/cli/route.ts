import { NextRequest } from "next/server";
import { getEnv } from "@/lib/env";
import { handleError } from "@/lib/response";

export async function GET(request: NextRequest) {
  try {
    const provider = request.nextUrl.searchParams.get("provider") ?? "github";
    const callback = request.nextUrl.searchParams.get("callback") ?? "";

    const env = getEnv();
    const oauthUrl = new URL(`${env.NEXT_PUBLIC_APP_URL}/api/auth/oauth/${provider}`);
    if (callback) {
      oauthUrl.searchParams.set("callback", callback);
    }

    return Response.redirect(oauthUrl.toString());
  } catch (err) {
    return handleError(err);
  }
}
