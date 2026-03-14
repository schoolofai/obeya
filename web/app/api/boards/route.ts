import { ID, Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { createBoardSchema } from "@/lib/boards/schemas";
import { deserializeBoard, serializeColumns } from "@/lib/boards/serialize";
import { buildBoardPermissions } from "@/lib/appwrite/doc-permissions";

export async function GET(request: Request) {
  try {
    const user = await authenticate(request);
    const boards = await listUserBoards(user.id);
    return ok(boards, { meta: { total: boards.length } });
  } catch (err) {
    return handleError(err);
  }
}

export async function POST(request: Request) {
  try {
    const user = await authenticate(request);
    const input = await validateBody(request, createBoardSchema);
    const board = await createBoard(user.id, input);
    return ok(board, { status: 201 });
  } catch (err) {
    return handleError(err);
  }
}

async function listUserBoards(userId: string) {
  const db = getDatabases();
  const env = getEnv();

  const owned = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARDS,
    [Query.equal("owner_id", userId), Query.limit(100)]
  );

  const memberships = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARD_MEMBERS,
    [Query.equal("user_id", userId), Query.limit(100)]
  );

  const memberBoardIds = memberships.documents.map((m) => m.board_id as string);
  const memberBoards = await fetchBoardsByIds(db, env.APPWRITE_DATABASE_ID, memberBoardIds);

  const boardMap = new Map<string, Record<string, unknown>>();
  for (const doc of owned.documents) boardMap.set(doc.$id, doc);
  for (const doc of memberBoards) boardMap.set(doc.$id as string, doc);

  return Array.from(boardMap.values()).map(deserializeBoard);
}

async function fetchBoardsByIds(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  ids: string[]
): Promise<Record<string, unknown>[]> {
  const results: Record<string, unknown>[] = [];
  for (const id of ids) {
    try {
      const doc = await db.getDocument(databaseId, COLLECTIONS.BOARDS, id);
      results.push(doc);
    } catch (err: unknown) {
      if (err instanceof Error && (err as any).code === 404) continue;
      throw err;
    }
  }
  return results;
}

async function createBoard(userId: string, input: any) {
  const db = getDatabases();
  const env = getEnv();
  const now = new Date().toISOString();

  const permissions = buildBoardPermissions(userId, []);

  const doc = await db.createDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARDS,
    ID.unique(),
    {
      name: input.name,
      owner_id: userId,
      org_id: input.org_id || null,
      display_counter: 0,
      columns: serializeColumns(input.columns),
      display_map: "{}",
      users: "{}",
      projects: "{}",
      agent_role: input.agent_role,
      version: 1,
      created_at: now,
      updated_at: now,
    },
    permissions,
  );

  return deserializeBoard(doc);
}
