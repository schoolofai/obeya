import { NextRequest } from "next/server";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { completeItemSchema } from "@/lib/items/review-schemas";
import { deserializeItem, serializeItem } from "@/lib/items/serialize";
import { createHistoryEntry } from "@/lib/history";
import { findItemByRef } from "@/lib/items/find-by-ref";

type RouteContext = { params: Promise<{ id: string; ref: string }> };

export async function POST(request: NextRequest, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id: boardId, ref } = await context.params;
    await assertBoardAccess(boardId, user.id, "editor");

    const input = await validateBody(request, completeItemSchema);
    const itemDoc = await findItemByRef(boardId, ref);

    const now = new Date().toISOString();
    const updatePayload = serializeItem({
      status: "done",
      confidence: input.confidence,
      review_context: input.review_context,
      human_review: { status: "pending" },
      updated_at: now,
    });

    const db = getDatabases();
    const env = getEnv();
    const updatedDoc = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEMS,
      itemDoc.$id as string,
      updatePayload
    );

    await createHistoryEntry({
      itemId: itemDoc.$id as string,
      boardId,
      userId: user.id,
      action: "complete-with-context",
      detail: input.review_context.purpose,
    });

    return ok(deserializeItem(updatedDoc as Record<string, unknown>));
  } catch (err) {
    return handleError(err);
  }
}
