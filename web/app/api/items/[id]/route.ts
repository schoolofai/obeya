import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { updateItemSchema } from "@/lib/items/schemas";
import { deserializeItem } from "@/lib/items/serialize";
import { AppError, ErrorCode } from "@/lib/errors";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;
    const itemDoc = await getItemOrThrow(id);
    await assertBoardAccess(itemDoc.board_id as string, user.id, "viewer");
    return ok(deserializeItem(itemDoc));
  } catch (err) {
    return handleError(err);
  }
}

export async function PATCH(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;
    const itemDoc = await getItemOrThrow(id);
    await assertBoardAccess(itemDoc.board_id as string, user.id, "editor");
    const input = await validateBody(request, updateItemSchema);
    const updated = await updateItem(id, input);
    return ok(deserializeItem(updated));
  } catch (err) {
    return handleError(err);
  }
}

export async function DELETE(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;
    const itemDoc = await getItemOrThrow(id);
    await assertBoardAccess(itemDoc.board_id as string, user.id, "editor");
    const db = getDatabases();
    const env = getEnv();
    await db.deleteDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, id);
    return ok({ deleted: true, id });
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

async function updateItem(id: string, input: ReturnType<typeof updateItemSchema.parse>) {
  const db = getDatabases();
  const env = getEnv();
  const updatePayload: Record<string, unknown> = { updated_at: new Date().toISOString() };
  if (input.title !== undefined) updatePayload.title = input.title;
  if (input.description !== undefined) updatePayload.description = input.description;
  if (input.priority !== undefined) updatePayload.priority = input.priority;
  if (input.parent_id !== undefined) updatePayload.parent_id = input.parent_id;
  if (input.project !== undefined) updatePayload.project = input.project;
  if (input.tags !== undefined) updatePayload.tags = JSON.stringify(input.tags);
  return await db.updateDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, id, updatePayload);
}
