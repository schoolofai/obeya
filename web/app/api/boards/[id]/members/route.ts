import { ID, Query } from "node-appwrite";
import { z } from "zod";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody, validateParams } from "@/lib/validation";
import { requireBoardAccess } from "@/lib/permissions";
import { AppError, ErrorCode } from "@/lib/errors";
import { syncBoardPermissions } from "@/lib/appwrite/sync-permissions";

const paramsSchema = z.object({ id: z.string().min(1) });

const addMemberSchema = z.object({
  user_id: z.string().min(1),
  role: z.enum(["editor", "viewer"]),
});

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);

    const board = await fetchBoard(id);
    await requireBoardAccess(user.id, id, board.org_id as string | null, "viewer");

    const members = await listBoardMembers(id);
    return ok(members);
  } catch (err) {
    return handleError(err);
  }
}

export async function POST(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);

    const board = await fetchBoard(id);
    await requireBoardAccess(user.id, id, board.org_id as string | null, "owner");

    const input = await validateBody(request, addMemberSchema);
    const db = getDatabases();
    const env = getEnv();

    await checkDuplicateBoardMember(db, env.APPWRITE_DATABASE_ID, id, input.user_id);

    const member = await db.createDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARD_MEMBERS,
      ID.unique(),
      { user_id: input.user_id, board_id: id, role: input.role }
    );

    await syncBoardPermissions(id);

    return ok(member, { status: 201 });
  } catch (err) {
    return handleError(err);
  }
}

async function fetchBoard(boardId: string): Promise<Record<string, unknown>> {
  const db = getDatabases();
  const env = getEnv();
  return await db.getDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.BOARDS, boardId);
}

async function listBoardMembers(boardId: string): Promise<unknown[]> {
  const db = getDatabases();
  const env = getEnv();

  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.BOARD_MEMBERS, [
    Query.equal("board_id", boardId),
    Query.limit(500),
  ]);

  return result.documents;
}

async function checkDuplicateBoardMember(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  boardId: string,
  userId: string
): Promise<void> {
  const existing = await db.listDocuments(databaseId, COLLECTIONS.BOARD_MEMBERS, [
    Query.equal("board_id", boardId),
    Query.equal("user_id", userId),
    Query.limit(1),
  ]);

  if (existing.documents.length > 0) {
    throw new AppError(ErrorCode.SLUG_ALREADY_EXISTS, "User is already a member of this board");
  }
}
