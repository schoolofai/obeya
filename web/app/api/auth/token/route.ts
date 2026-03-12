import { z } from "zod";
import { Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { generateToken, hashToken } from "@/lib/auth/tokens";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { ID } from "node-appwrite";

export async function GET(request: Request) {
  try {
    const user = await authenticate(request);
    const tokens = await listUserTokens(user.id);
    return ok(tokens);
  } catch (err) {
    return handleError(err);
  }
}

async function listUserTokens(userId: string) {
  const env = getEnv();
  const db = getDatabases();
  const result = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.API_TOKENS,
    [Query.equal("user_id", userId), Query.limit(100)],
  );
  return result.documents.map((doc) => ({
    id: doc.$id,
    name: doc.name,
    scopes: doc.scopes,
    last_used_at: doc.last_used_at ?? null,
    created_at: doc.created_at,
  }));
}

const createTokenSchema = z.object({
  name: z.string().min(1).max(255),
  scopes: z.array(z.string()).default(["*"]),
});

export async function POST(request: Request) {
  try {
    const user = await authenticate(request);
    const body = await validateBody(request, createTokenSchema);
    const { id, token } = await createApiToken(user.id, body);
    return ok({ id, token, name: body.name, scopes: body.scopes }, { status: 201 });
  } catch (err) {
    return handleError(err);
  }
}

async function createApiToken(
  userId: string,
  body: z.infer<typeof createTokenSchema>,
): Promise<{ id: string; token: string }> {
  const env = getEnv();
  const db = getDatabases();
  const token = generateToken();
  const tokenHash = await hashToken(token);

  const doc = await db.createDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.API_TOKENS,
    ID.unique(),
    {
      user_id: userId,
      token_hash: tokenHash,
      name: body.name,
      scopes: body.scopes,
      created_at: new Date().toISOString(),
    },
  );

  return { id: doc.$id, token };
}
