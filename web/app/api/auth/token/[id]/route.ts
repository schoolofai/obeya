import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { AppError, ErrorCode } from "@/lib/errors";

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ id: string }> },
) {
  try {
    const user = await authenticate(request);
    const { id } = await params;
    await revokeToken(user.id, id);
    return ok({ revoked: true });
  } catch (err) {
    return handleError(err);
  }
}

async function revokeToken(userId: string, tokenId: string): Promise<void> {
  const env = getEnv();
  const db = getDatabases();

  const doc = await db.getDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.API_TOKENS,
    tokenId,
  );

  if (doc.user_id !== userId) {
    throw new AppError(ErrorCode.FORBIDDEN, "Cannot revoke another user's token");
  }

  await db.deleteDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.API_TOKENS,
    tokenId,
  );
}
