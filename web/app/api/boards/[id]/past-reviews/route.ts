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
      Query.limit(5000),
    ]);

    const allItems = result.documents.map((doc) =>
      deserializeItem(doc as Record<string, unknown>)
    );

    const reviewedItems = allItems.filter(
      (item) => item.human_review !== null && item.human_review !== undefined
    );

    const ancestorIds = collectAncestorIds(reviewedItems, allItems);
    const relevantItems = allItems.filter(
      (item) =>
        (item.human_review !== null && item.human_review !== undefined) ||
        ancestorIds.has(item.id)
    );

    return ok(relevantItems, { meta: { total: relevantItems.length } });
  } catch (err) {
    return handleError(err);
  }
}

function collectAncestorIds(reviewed: Item[], allItems: Item[]): Set<string> {
  const itemMap = new Map(allItems.map((i) => [i.id, i]));
  const ancestors = new Set<string>();

  for (const item of reviewed) {
    let current = item;
    while (current.parent_id && itemMap.has(current.parent_id)) {
      ancestors.add(current.parent_id);
      current = itemMap.get(current.parent_id)!;
    }
  }

  return ancestors;
}
