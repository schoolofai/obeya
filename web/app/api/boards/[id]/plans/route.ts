import { NextRequest } from "next/server";
import { z } from "zod";
import { ID, Query } from "node-appwrite";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { authenticate } from "@/lib/auth/middleware";
import { validateBody, validateParams } from "@/lib/validation";
import { ok, handleError } from "@/lib/response";

const paramsSchema = z.object({ id: z.string().min(1) });

const createPlanSchema = z.object({
  title: z.string().min(1),
  source_path: z.string().default(""),
  content: z.string().default(""),
});

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
      COLLECTIONS.PLANS,
      [
        Query.equal("board_id", id),
        Query.orderDesc("created_at"),
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

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ id: string }> }
) {
  try {
    await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    const body = await validateBody(request, createPlanSchema);

    const env = getEnv();
    const db = getDatabases();

    const board = await db.getDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      id
    );

    const displayNum = board.display_counter + 1;

    await db.updateDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.BOARDS,
      id,
      { display_counter: displayNum }
    );

    const plan = await db.createDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.PLANS,
      ID.unique(),
      {
        board_id: id,
        display_num: displayNum,
        title: body.title,
        source_path: body.source_path,
        content: body.content,
        linked_items: "[]",
        created_at: new Date().toISOString(),
      }
    );

    return ok(plan, { status: 201 });
  } catch (err) {
    return handleError(err);
  }
}
