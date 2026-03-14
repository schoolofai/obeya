import { ID, Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { incrementDisplayCounter } from "@/lib/boards/counter";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { createItemSchema } from "@/lib/items/schemas";
import { deserializeItem, serializeItem } from "@/lib/items/serialize";
import { buildItemPermissions, fetchBoardMemberList } from "@/lib/appwrite/doc-permissions";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;
    await assertBoardAccess(id, user.id, "viewer");

    const db = getDatabases();
    const env = getEnv();
    const url = new URL(request.url);
    const queries = buildItemFilters(id, url.searchParams);
    const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, queries);
    const items = result.documents.map(deserializeItem);
    return ok(items, { meta: { total: result.total } });
  } catch (err) {
    return handleError(err);
  }
}

export async function POST(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id: boardId } = await context.params;
    await assertBoardAccess(boardId, user.id, "editor");

    const input = await validateBody(request, createItemSchema);
    const displayNum = await incrementDisplayCounter(boardId);
    const item = await createItem(boardId, displayNum, input);
    await updateDisplayMap(boardId, displayNum, item.id);
    return ok(item, { status: 201 });
  } catch (err) {
    return handleError(err);
  }
}

function buildItemFilters(boardId: string, searchParams: URLSearchParams): string[] {
  const queries = [Query.equal("board_id", boardId), Query.limit(5000)];
  const status = searchParams.get("status");
  if (status) queries.push(Query.equal("status", status));
  const type = searchParams.get("type");
  if (type) queries.push(Query.equal("type", type));
  const assignee = searchParams.get("assignee");
  if (assignee) queries.push(Query.equal("assignee_id", assignee));
  return queries;
}

async function createItem(boardId: string, displayNum: number, input: ReturnType<typeof createItemSchema.parse>) {
  const db = getDatabases();
  const env = getEnv();
  const now = new Date().toISOString();

  const board = await db.getDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.BOARDS, boardId);
  const members = await fetchBoardMemberList(boardId);
  const permissions = buildItemPermissions(board.owner_id as string, members);

  const serialized = serializeItem({
    board_id: boardId,
    display_num: displayNum,
    type: input.type,
    title: input.title,
    description: input.description,
    status: input.status,
    priority: input.priority,
    parent_id: input.parent_id || null,
    assignee_id: input.assignee_id || null,
    blocked_by: input.blocked_by,
    tags: input.tags,
    project: input.project || null,
    created_at: now,
    updated_at: now,
  });

  const itemDoc = await db.createDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, ID.unique(), serialized, permissions);
  return deserializeItem(itemDoc);
}

async function updateDisplayMap(boardId: string, displayNum: number, itemId: string): Promise<void> {
  const db = getDatabases();
  const env = getEnv();
  const board = await db.getDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.BOARDS, boardId);
  const displayMap: Record<string, string> = board.display_map
    ? JSON.parse(board.display_map as string)
    : {};
  displayMap[String(displayNum)] = itemId;
  await db.updateDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.BOARDS, boardId, {
    display_map: JSON.stringify(displayMap),
  });
}
