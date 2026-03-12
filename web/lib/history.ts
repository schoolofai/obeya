import { ID } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";

type HistoryAction = "created" | "moved" | "edited" | "assigned" | "blocked" | "unblocked";

interface HistoryParams {
  itemId: string;
  boardId: string;
  userId: string;
  action: HistoryAction;
  detail: string;
}

export async function createHistoryEntry(params: HistoryParams): Promise<void> {
  const env = getEnv();
  const db = getDatabases();
  await db.createDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEM_HISTORY, ID.unique(), {
    item_id: params.itemId,
    board_id: params.boardId,
    user_id: params.userId,
    action: params.action,
    detail: params.detail,
    timestamp: new Date().toISOString(),
  });
}
