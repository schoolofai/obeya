import { NextRequest } from "next/server";
import { Client, Account, ID } from "node-appwrite";
import { handleError } from "@/lib/response";
import { getEnv } from "@/lib/env";
import { getDatabases } from "@/lib/appwrite/server";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { generateToken, hashToken } from "@/lib/auth/tokens";

export async function GET(request: NextRequest) {
  try {
    const { userId, secret, callback } = extractParams(request);
    const session = await createSession(userId, secret);

    if (callback) {
      return await handleCliCallback(callback, session.userId);
    }

    return handleWebCallback(session.secret);
  } catch (err) {
    return handleError(err);
  }
}

function extractParams(request: NextRequest) {
  const userId = request.nextUrl.searchParams.get("userId") ?? "";
  const secret = request.nextUrl.searchParams.get("secret") ?? "";
  const callback = request.nextUrl.searchParams.get("callback");
  return { userId, secret, callback };
}

async function createSession(userId: string, secret: string) {
  const env = getEnv();
  const client = new Client()
    .setEndpoint(env.APPWRITE_ENDPOINT)
    .setProject(env.APPWRITE_PROJECT_ID);

  const account = new Account(client);
  return await account.createSession(userId, secret);
}

async function handleCliCallback(callback: string, userId: string): Promise<Response> {
  const token = generateToken();
  const tokenHash = await hashToken(token);
  await storeApiToken(userId, tokenHash);

  const redirectUrl = new URL(callback);
  redirectUrl.searchParams.set("token", token);
  return Response.redirect(redirectUrl.toString());
}

function handleWebCallback(sessionSecret: string): Response {
  const env = getEnv();
  const headers = new Headers();
  headers.set("Location", `${env.NEXT_PUBLIC_APP_URL}/dashboard`);
  headers.set(
    "Set-Cookie",
    `a_session=${sessionSecret}; Path=/; HttpOnly; SameSite=Lax; Secure`,
  );
  return new Response(null, { status: 302, headers });
}

async function storeApiToken(userId: string, tokenHash: string): Promise<void> {
  const env = getEnv();
  const db = getDatabases();
  await db.createDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.API_TOKENS,
    ID.unique(),
    {
      user_id: userId,
      token_hash: tokenHash,
      created_at: new Date().toISOString(),
      last_used_at: null,
    },
  );
}
