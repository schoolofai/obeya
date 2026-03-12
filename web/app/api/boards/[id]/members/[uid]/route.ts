import { Query } from "node-appwrite";
import { z } from "zod";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody, validateParams } from "@/lib/validation";
import { requireBoardAccess } from "@/lib/permissions";
import { AppError, ErrorCode } from "@/lib/errors";

const paramsSchema = z.object({
  id: z.string().min(1),
  uid: z.string().min(1),
});

const updateMemberSchema = z.object({
  role: z.enum(["editor", "viewer"]),
});

type RouteContext = { params: Promise<{ id: string; uid: string }> };

export async function PATCH(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id, uid } = validateParams(await context.params, paramsSchema);

    const board = await fetchBoard(id);
    await requireBoardAccess(user.id, id, board.org_id as string | null, "owner");

    const input = await validateBody(request, updateMemberSchema);
    const membership = await findBoardMembership(id, uid);

    if (membership.role === "owner") {
      throw new AppError(ErrorCode.FORBIDDEN, "Cannot change the board owner's role");
    }

    const updated = await updateBoardMemberRole(membership.$id as string, input.role);
    return ok(updated);
  } catch (err) {
    return handleError(err);
  }
}

export async function DELETE(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id, uid } = validateParams(await context.params, paramsSchema);

    const board = await fetchBoard(id);
    const isSelf = user.id === uid;
    const requiredLevel = isSelf ? "viewer" : "owner";
    await requireBoardAccess(user.id, id, board.org_id as string | null, requiredLevel);

    const membership = await findBoardMembership(id, uid);

    if (membership.role === "owner") {
      throw new AppError(ErrorCode.FORBIDDEN, "Cannot remove the board owner");
    }

    const db = getDatabases();
    const env = getEnv();
    await db.deleteDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.BOARD_MEMBERS, membership.$id as string);

    return ok({ deleted: true, user_id: uid });
  } catch (err) {
    return handleError(err);
  }
}

async function fetchBoard(boardId: string): Promise<Record<string, unknown>> {
  const db = getDatabases();
  const env = getEnv();
  return await db.getDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.BOARDS, boardId);
}

async function findBoardMembership(
  boardId: string,
  userId: string
): Promise<Record<string, unknown>> {
  const db = getDatabases();
  const env = getEnv();

  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.BOARD_MEMBERS, [
    Query.equal("board_id", boardId),
    Query.equal("user_id", userId),
    Query.limit(1),
  ]);

  if (result.documents.length === 0) {
    throw new AppError(ErrorCode.USER_NOT_FOUND, `User ${userId} is not a member of this board`);
  }

  return result.documents[0];
}

async function updateBoardMemberRole(
  membershipId: string,
  role: string
): Promise<Record<string, unknown>> {
  const db = getDatabases();
  const env = getEnv();

  return await db.updateDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARD_MEMBERS,
    membershipId,
    { role } as unknown as Record<string, unknown>
  );
}
