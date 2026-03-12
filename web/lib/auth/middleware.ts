import { AppError, ErrorCode } from "@/lib/errors";
import { getUserFromSession, type AuthUser } from "@/lib/auth/session";
import { verifyToken } from "@/lib/auth/tokens";
import { getDatabases, getUsers } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { Query } from "node-appwrite";

export async function authenticate(request: Request): Promise<AuthUser> {
  const authHeader = request.headers.get("authorization");
  if (authHeader?.startsWith("Bearer ")) {
    const token = authHeader.slice(7);
    return await authenticateWithToken(token);
  }

  const cookie = request.headers.get("cookie");
  if (cookie) {
    return await getUserFromSession(cookie);
  }

  throw new AppError(ErrorCode.UNAUTHORIZED, "No authentication provided");
}

async function authenticateWithToken(token: string): Promise<AuthUser> {
  const env = getEnv();
  const db = getDatabases();

  let offset = 0;
  const batchSize = 100;

  while (true) {
    const result = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.API_TOKENS,
      [Query.limit(batchSize), Query.offset(offset)],
    );

    for (const doc of result.documents) {
      const match = await verifyToken(token, doc.token_hash);
      if (match) {
        updateLastUsed(db, env.APPWRITE_DATABASE_ID, doc.$id);
        return await lookupUser(doc.user_id);
      }
    }

    if (result.documents.length < batchSize) {
      break;
    }
    offset += batchSize;
  }

  throw new AppError(ErrorCode.UNAUTHORIZED, "Invalid API token");
}

async function lookupUser(userId: string): Promise<AuthUser> {
  const users = getUsers();
  const user = await users.get(userId);
  return { id: user.$id, email: user.email, name: user.name };
}

function updateLastUsed(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  documentId: string,
): void {
  db.updateDocument(databaseId, COLLECTIONS.API_TOKENS, documentId, {
    last_used_at: new Date().toISOString(),
  }).catch((err: unknown) => {
    console.error("Failed to update token last_used_at:", err);
  });
}
