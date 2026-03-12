import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { updateBoardSchema } from "@/lib/boards/schemas";
import { deserializeBoard, serializeColumns } from "@/lib/boards/serialize";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;
    const boardDoc = await assertBoardAccess(id, user.id, "viewer");
    return ok(deserializeBoard(boardDoc));
  } catch (err) {
    return handleError(err);
  }
}

export async function PATCH(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;
    await assertBoardAccess(id, user.id, "editor");

    const input = await validateBody(request, updateBoardSchema);
    const updated = await updateBoard(id, input);
    return ok(deserializeBoard(updated));
  } catch (err) {
    return handleError(err);
  }
}

export async function DELETE(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;
    await assertBoardAccess(id, user.id, "owner");

    const db = getDatabases();
    const env = getEnv();
    await db.deleteDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.BOARDS, id);
    return ok({ deleted: true, id });
  } catch (err) {
    return handleError(err);
  }
}

async function updateBoard(id: string, input: any) {
  const db = getDatabases();
  const env = getEnv();
  const updatePayload: Record<string, unknown> = {
    updated_at: new Date().toISOString(),
  };

  if (input.name !== undefined) updatePayload.name = input.name;
  if (input.columns !== undefined) updatePayload.columns = serializeColumns(input.columns);
  if (input.agent_role !== undefined) updatePayload.agent_role = input.agent_role;

  return await db.updateDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARDS,
    id,
    updatePayload
  );
}
