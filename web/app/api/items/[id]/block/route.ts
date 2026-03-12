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
const blockBodySchema = z.object({ blocked_by_id: z.string().min(1) });

export async function POST(request: NextRequest, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const rawParams = await context.params;
    const { id } = validateParams(rawParams, paramsSchema);
    const { blocked_by_id } = await validateBody(request, blockBodySchema);

    const itemDoc = await getItemOrThrow(id);
    await assertBoardAccess(itemDoc.board_id as string, user.id, "editor");

    const currentBlockers = parseBlockedBy(itemDoc.blocked_by as string);
    assertNotDuplicate(currentBlockers, blocked_by_id);

    const updatedBlockers = [...currentBlockers, blocked_by_id];
    const updatedDoc = await persistBlockedBy(id, updatedBlockers);

    await createHistoryEntry({
      itemId: id,
      boardId: itemDoc.board_id as string,
      userId: user.id,
      action: "blocked",
      detail: `blocked by ${blocked_by_id}`,
    });

    return ok(deserializeItem(updatedDoc as Record<string, unknown>));
  } catch (err) {
    return handleError(err);
  }
}

function parseBlockedBy(json: string): string[] {
  if (!json) return [];
  return JSON.parse(json) as string[];
}

function assertNotDuplicate(blockers: string[], newBlockerId: string): void {
  if (blockers.includes(newBlockerId)) {
    throw new AppError(
      ErrorCode.VALIDATION_ERROR,
      `Item is already blocked by ${newBlockerId}`
    );
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

async function persistBlockedBy(id: string, blockers: string[]): Promise<unknown> {
  const db = getDatabases();
  const env = getEnv();
  return await db.updateDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, id, {
    blocked_by: JSON.stringify(blockers),
    updated_at: new Date().toISOString(),
  });
}
