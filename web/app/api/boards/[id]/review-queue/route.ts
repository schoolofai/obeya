import { NextRequest } from "next/server";
import { Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { deserializeItem, type Item } from "@/lib/items/serialize";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: NextRequest, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id: boardId } = await context.params;
    await assertBoardAccess(boardId, user.id, "viewer");

    const db = getDatabases();
    const env = getEnv();

    const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, [
      Query.equal("board_id", boardId),
      Query.equal("status", "done"),
      Query.limit(5000),
    ]);

    const allItems = result.documents.map((doc) =>
      deserializeItem(doc as Record<string, unknown>)
    );

    const queueItems = filterReviewQueue(allItems);
    const sorted = sortByConfidence(queueItems);

    return ok(sorted, { meta: { total: sorted.length } });
  } catch (err) {
    return handleError(err);
  }
}

function filterReviewQueue(items: Item[]): Item[] {
  return items.filter(
    (item) =>
      item.review_context !== null &&
      item.review_context !== undefined &&
      (item.human_review === null ||
        item.human_review === undefined ||
        item.human_review.status !== "hidden")
  );
}

function sortByConfidence(items: Item[]): Item[] {
  return [...items].sort((a, b) => {
    const aConf = a.confidence ?? -1;
    const bConf = b.confidence ?? -1;
    if (aConf !== bConf) return aConf - bConf;
    return new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime();
  });
}
