import { NextRequest } from "next/server";
import { z } from "zod";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";

const paramsSchema = z.object({ id: z.string().min(1) });

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);

    const env = getEnv();
    const db = getDatabases();

    const plan = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      id
    );

    const linkedItemIds: string[] = JSON.parse(plan.linked_items || "[]");
    const linkedItems = await resolveLinkedItems(db, env.APPWRITE_DATABASE_ID, linkedItemIds);

    return ok({ plan, linked_items: linkedItems });
  } catch (err) {
    return handleError(err);
  }
}

async function resolveLinkedItems(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  itemIds: string[]
): Promise<unknown[]> {
  if (itemIds.length === 0) return [];

  const results = await Promise.all(
    itemIds.map((itemId) =>
      db.getDocument(databaseId, COLLECTIONS.ITEMS, itemId).catch(() => null)
    )
  );

  return results.filter(Boolean);
}
