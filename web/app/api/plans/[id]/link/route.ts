import { NextRequest } from "next/server";
import { z } from "zod";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateBody, validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";

const paramsSchema = z.object({ id: z.string().min(1) });
const bodySchema = z.object({
  item_ids: z.array(z.string().min(1)).min(1),
});

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    const { item_ids } = await validateBody(request, bodySchema);

    const env = getEnv();
    const db = getDatabases();

    const plan = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      id
    );

    const existing: string[] = JSON.parse(plan.linked_items || "[]");
    const merged = [...new Set([...existing, ...item_ids])];

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      id,
      { linked_items: JSON.stringify(merged) }
    );

    return ok(updated);
  } catch (err) {
    return handleError(err);
  }
}
