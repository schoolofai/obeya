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
import { createHistoryEntry } from "@/lib/history";
import { AppError, ErrorCode } from "@/lib/errors";

type RouteContext = { params: Promise<{ id: string }> };

const paramsSchema = z.object({ id: z.string().min(1) });
const assignBodySchema = z.object({ assignee_id: z.string().nullable() });

export async function POST(request: NextRequest, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const rawParams = await context.params;
    const { id } = validateParams(rawParams, paramsSchema);
    const { assignee_id } = await validateBody(request, assignBodySchema);

    const itemDoc = await getItemOrThrow(id);
    await assertBoardAccess(itemDoc.board_id as string, user.id, "editor");

    const updatedDoc = await updateAssignee(id, assignee_id);

    const detail = buildAssignDetail(assignee_id);
    await createHistoryEntry({
      itemId: id,
      boardId: itemDoc.board_id as string,
      userId: user.id,
      action: "assigned",
      detail,
    });

    return ok(deserializeItem(updatedDoc as Record<string, unknown>));
  } catch (err) {
    return handleError(err);
  }
}

function buildAssignDetail(assigneeId: string | null): string {
  if (assigneeId === null) return "unassigned";
  return `assigned to ${assigneeId}`;
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

async function updateAssignee(id: string, assigneeId: string | null): Promise<unknown> {
  const db = getDatabases();
  const env = getEnv();
  return await db.updateDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, id, {
    assignee_id: assigneeId,
    updated_at: new Date().toISOString(),
  });
}
