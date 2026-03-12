import { NextRequest } from "next/server";
import { z } from "zod";
import { Query } from "node-appwrite";
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

    const url = new URL(request.url);
    const limit = Math.min(parseInt(url.searchParams.get("limit") || "25"), 100);
    const offset = parseInt(url.searchParams.get("offset") || "0");

    const result = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ITEM_HISTORY,
      [
        Query.equal("board_id", id),
        Query.orderDesc("timestamp"),
        Query.limit(limit),
        Query.offset(offset),
      ]
    );

    return ok(result.documents, {
      meta: { total: result.total, limit, offset },
    });
  } catch (err) {
    return handleError(err);
  }
}
