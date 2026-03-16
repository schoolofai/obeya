import { Query } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { AppError, ErrorCode } from "@/lib/errors";

export async function findItemByRef(boardId: string, ref: string): Promise<Record<string, unknown>> {
  const db = getDatabases();
  const env = getEnv();

  const displayNum = parseInt(ref, 10);
  if (!isNaN(displayNum)) {
    const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, [
      Query.equal("board_id", boardId),
      Query.equal("display_num", displayNum),
      Query.limit(1),
    ]);
    if (result.documents.length > 0) return result.documents[0] as Record<string, unknown>;
  }

  try {
    const doc = await db.getDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, ref);
    if (doc.board_id === boardId) return doc as Record<string, unknown>;
  } catch {
    // Not found by ID, fall through
  }

  throw new AppError(ErrorCode.ITEM_NOT_FOUND, `Item '${ref}' not found on board ${boardId}`);
}
