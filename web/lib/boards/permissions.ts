import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { AppError, ErrorCode } from "@/lib/errors";
import { Query } from "node-appwrite";

const ROLE_HIERARCHY: Record<string, number> = {
  viewer: 1,
  editor: 2,
  owner: 3,
};

export async function assertBoardAccess(
  boardId: string,
  userId: string,
  requiredRole: "viewer" | "editor" | "owner"
): Promise<Record<string, unknown>> {
  const db = getDatabases();
  const env = getEnv();

  const board = await getBoardOrThrow(db, env.APPWRITE_DATABASE_ID, boardId);

  if (board.owner_id === userId) {
    return board;
  }

  await assertMemberRole(db, env.APPWRITE_DATABASE_ID, boardId, userId, requiredRole);

  return board;
}

async function assertMemberRole(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  boardId: string,
  userId: string,
  requiredRole: "viewer" | "editor" | "owner"
): Promise<void> {
  const members = await db.listDocuments(databaseId, COLLECTIONS.BOARD_MEMBERS, [
    Query.equal("board_id", boardId),
    Query.equal("user_id", userId),
    Query.limit(1),
  ]);

  if (members.documents.length > 0) {
    const memberRole = members.documents[0].role as string;
    if (ROLE_HIERARCHY[memberRole] >= ROLE_HIERARCHY[requiredRole]) {
      return;
    }
  }

  throw new AppError(ErrorCode.FORBIDDEN, "You do not have access to this board");
}

async function getBoardOrThrow(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  boardId: string
): Promise<Record<string, unknown>> {
  try {
    return await db.getDocument(databaseId, COLLECTIONS.BOARDS, boardId);
  } catch (err: unknown) {
    if (err instanceof Error && (err as any).code === 404) {
      throw new AppError(ErrorCode.BOARD_NOT_FOUND, `Board ${boardId} not found`);
    }
    throw err;
  }
}
