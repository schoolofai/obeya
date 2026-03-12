import { NextRequest } from "next/server";
import { z } from "zod";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateParams, validateBody } from "@/lib/validation";
import { deserializeItem } from "@/lib/items/serialize";
import { deserializeColumns } from "@/lib/boards/serialize";
import { createHistoryEntry } from "@/lib/history";
import { AppError, ErrorCode } from "@/lib/errors";
import { Query } from "node-appwrite";

type RouteContext = { params: Promise<{ id: string }> };

const paramsSchema = z.object({ id: z.string().min(1) });
const moveBodySchema = z.object({ status: z.string().min(1) });

export async function POST(request: NextRequest, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const rawParams = await context.params;
    const { id } = validateParams(rawParams, paramsSchema);
    const { status: targetStatus } = await validateBody(request, moveBodySchema);

    const itemDoc = await getItemOrThrow(id);
    const boardDoc = await assertBoardAccess(itemDoc.board_id as string, user.id, "editor");

    const columns = deserializeColumns(boardDoc.columns as string);
    const targetColumn = columns.find((c) => c.name === targetStatus);
    if (!targetColumn) {
      throw new AppError(ErrorCode.VALIDATION_ERROR, `Column '${targetStatus}' does not exist`);
    }

    if (targetColumn.limit > 0) {
      await assertWipLimit(itemDoc.board_id as string, targetStatus, targetColumn.limit);
    }

    const oldStatus = itemDoc.status as string;
    const updatedDoc = await updateItemStatus(id, targetStatus);

    await createHistoryEntry({
      itemId: id,
      boardId: itemDoc.board_id as string,
      userId: user.id,
      action: "moved",
      detail: `status: ${oldStatus} -> ${targetStatus}`,
    });

    return ok(deserializeItem(updatedDoc as Record<string, unknown>));
  } catch (err) {
    return handleError(err);
  }
}

async function getItemOrThrow(itemId: string): Promise<Record<string, unknown>> {
  const db = getDatabases();
  const env = getEnv();
  try {
    return await db.getDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, itemId);
  } catch (err: unknown) {
    if (err instanceof Error && (err as any).code === 404) {
      throw new AppError(ErrorCode.ITEM_NOT_FOUND, `Item ${itemId} not found`);
    }
    throw err;
  }
}

async function assertWipLimit(boardId: string, status: string, limit: number): Promise<void> {
  const db = getDatabases();
  const env = getEnv();
  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, [
    Query.equal("board_id", boardId),
    Query.equal("status", status),
    Query.limit(limit + 1),
  ]);
  if (result.total >= limit) {
    throw new AppError(
      ErrorCode.VALIDATION_ERROR,
      `WIP limit of ${limit} reached for column '${status}'`
    );
  }
}

async function updateItemStatus(id: string, status: string): Promise<unknown> {
  const db = getDatabases();
  const env = getEnv();
  return await db.updateDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, id, {
    status,
    updated_at: new Date().toISOString(),
  });
}
