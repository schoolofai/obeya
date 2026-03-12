import { NextRequest } from "next/server";
import { z } from "zod";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";
import { AppError, ErrorCode } from "@/lib/errors";

const paramsSchema = z.object({
  id: z.string().min(1),
  iid: z.string().min(1),
});

export async function DELETE(
  request: NextRequest,
  context: { params: Promise<{ id: string; iid: string }> }
) {
  try {
    await authenticate(request);
    const { id, iid } = validateParams(await context.params, paramsSchema);

    const env = getEnv();
    const db = getDatabases();

    const plan = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      id
    );

    const linkedItems: string[] = JSON.parse(plan.linked_items || "[]");
    const index = linkedItems.indexOf(iid);

    if (index === -1) {
      throw new AppError(
        ErrorCode.ITEM_NOT_FOUND,
        `Item "${iid}" is not linked to this plan`
      );
    }

    linkedItems.splice(index, 1);

    const updated = await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      id,
      { linked_items: JSON.stringify(linkedItems) }
    );

    return ok(updated);
  } catch (err) {
    return handleError(err);
  }
}
