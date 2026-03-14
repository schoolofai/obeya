import { Query } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import {
  buildBoardPermissions,
  buildItemPermissions,
  fetchBoardMemberList,
} from "./doc-permissions";

/** Update permissions on a board and all its items when members change. */
export async function syncBoardPermissions(boardId: string): Promise<number> {
  const db = getDatabases();
  const env = getEnv();

  const board = await db.getDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARDS,
    boardId,
  );
  const ownerId = board.owner_id as string;
  const members = await fetchBoardMemberList(boardId);

  const boardPerms = buildBoardPermissions(ownerId, members);
  await db.updateDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARDS,
    boardId,
    {},
    boardPerms,
  );

  const items = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.ITEMS,
    [Query.equal("board_id", boardId), Query.limit(5000)],
  );

  const itemPerms = buildItemPermissions(ownerId, members);
  let updated = 0;
  for (const item of items.documents) {
    await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      item.$id,
      {},
      itemPerms,
    );
    updated++;
  }

  return updated + 1; // items + board
}
