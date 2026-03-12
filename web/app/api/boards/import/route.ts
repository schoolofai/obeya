import { z } from "zod";
import { ID } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { serializeColumns } from "@/lib/boards/serialize";

const importItemSchema = z.object({
  id: z.string(),
  display_num: z.number().int(),
  type: z.enum(["epic", "story", "task"]),
  title: z.string(),
  description: z.string().default(""),
  status: z.string(),
  priority: z.enum(["low", "medium", "high", "critical"]),
  parent_id: z.string().nullable().default(null),
  assignee_id: z.string().nullable().default(null),
  blocked_by: z.array(z.string()).default([]),
  tags: z.array(z.string()).default([]),
  project: z.string().nullable().default(null),
});

const importBoardSchema = z.object({
  name: z.string().min(1),
  display_counter: z.number().int(),
  columns: z.array(z.object({ name: z.string(), limit: z.number().int().default(0) })),
  display_map: z.record(z.string(), z.string()).default({}),
  users: z.record(z.string(), z.unknown()).default({}),
  projects: z.record(z.string(), z.unknown()).default({}),
  agent_role: z.string().default("worker"),
  version: z.number().int().default(1),
  items: z.array(importItemSchema).default([]),
});

export async function POST(request: Request) {
  try {
    const user = await authenticate(request);
    const input = await validateBody(request, importBoardSchema);
    const result = await importBoard(user.id, input);
    return ok(result, { status: 201 });
  } catch (err) {
    return handleError(err);
  }
}

async function importBoard(userId: string, input: z.infer<typeof importBoardSchema>) {
  const db = getDatabases();
  const env = getEnv();
  const now = new Date().toISOString();

  const boardDoc = await createBoardDocument(db, env.APPWRITE_DATABASE_ID, userId, input, now);
  const boardId = boardDoc.$id;
  const idMap = await createItemDocuments(db, env.APPWRITE_DATABASE_ID, boardId, input.items, now);
  await resolveReferences(db, env.APPWRITE_DATABASE_ID, input.items, idMap);
  await updateDisplayMap(db, env.APPWRITE_DATABASE_ID, boardId, input.display_map, idMap);

  return { board_id: boardId, id_map: idMap, items_imported: input.items.length };
}

async function createBoardDocument(db: any, dbId: string, userId: string, input: any, now: string) {
  return await db.createDocument(dbId, COLLECTIONS.BOARDS, ID.unique(), {
    name: input.name, owner_id: userId, org_id: null,
    display_counter: input.display_counter,
    columns: serializeColumns(input.columns),
    display_map: "{}", users: JSON.stringify(input.users),
    projects: JSON.stringify(input.projects),
    agent_role: input.agent_role, version: input.version,
    created_at: now, updated_at: now,
  });
}

async function createItemDocuments(
  db: any,
  dbId: string,
  boardId: string,
  items: any[],
  now: string
): Promise<Record<string, string>> {
  const idMap: Record<string, string> = {};
  for (const item of items) {
    const itemDoc = await db.createDocument(dbId, COLLECTIONS.ITEMS, ID.unique(), {
      board_id: boardId, display_num: item.display_num,
      type: item.type, title: item.title, description: item.description,
      status: item.status, priority: item.priority,
      parent_id: null, assignee_id: item.assignee_id,
      blocked_by: "[]", tags: JSON.stringify(item.tags),
      project: item.project, created_at: now, updated_at: now,
    });
    idMap[item.id] = itemDoc.$id;
  }
  return idMap;
}

async function resolveReferences(
  db: any,
  dbId: string,
  items: any[],
  idMap: Record<string, string>
) {
  for (const item of items) {
    const cloudId = idMap[item.id];
    const updates: Record<string, unknown> = {};
    let needsUpdate = false;

    if (item.parent_id && idMap[item.parent_id]) {
      updates.parent_id = idMap[item.parent_id];
      needsUpdate = true;
    }
    if (item.blocked_by.length > 0) {
      updates.blocked_by = JSON.stringify(
        item.blocked_by.map((lid: string) => idMap[lid]).filter(Boolean)
      );
      needsUpdate = true;
    }
    if (needsUpdate) {
      await db.updateDocument(dbId, COLLECTIONS.ITEMS, cloudId, updates);
    }
  }
}

async function updateDisplayMap(
  db: any,
  dbId: string,
  boardId: string,
  displayMap: Record<string, string>,
  idMap: Record<string, string>
) {
  const cloudMap: Record<string, string> = {};
  for (const [num, localId] of Object.entries(displayMap)) {
    if (idMap[localId]) cloudMap[num] = idMap[localId];
  }
  await db.updateDocument(dbId, COLLECTIONS.BOARDS, boardId, {
    display_map: JSON.stringify(cloudMap),
  });
}
