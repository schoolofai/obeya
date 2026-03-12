import { Query } from "node-appwrite";
import { authenticate } from "@/lib/auth/middleware";
import { assertBoardAccess } from "@/lib/boards/permissions";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { deserializeColumns } from "@/lib/boards/serialize";

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = await context.params;
    const boardDoc = await assertBoardAccess(id, user.id, "viewer");
    const items = await fetchBoardItems(id);
    const exportData = buildExportPayload(boardDoc, items);
    return ok(exportData);
  } catch (err) {
    return handleError(err);
  }
}

async function fetchBoardItems(boardId: string) {
  const db = getDatabases();
  const env = getEnv();
  const result = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.ITEMS,
    [Query.equal("board_id", boardId), Query.limit(5000)]
  );
  return result.documents.map(formatItemForExport);
}

function buildExportPayload(
  boardDoc: Record<string, unknown>,
  items: Record<string, unknown>[]
) {
  return {
    name: boardDoc.name,
    display_counter: boardDoc.display_counter,
    columns: deserializeColumns(boardDoc.columns as string),
    display_map: safeParseJson(boardDoc.display_map as string, {}),
    users: safeParseJson(boardDoc.users as string, {}),
    projects: safeParseJson(boardDoc.projects as string, {}),
    agent_role: boardDoc.agent_role,
    version: boardDoc.version,
    items,
  };
}

function formatItemForExport(doc: Record<string, unknown>): Record<string, unknown> {
  return {
    id: doc.$id,
    display_num: doc.display_num,
    type: doc.type,
    title: doc.title,
    description: doc.description || "",
    status: doc.status,
    priority: doc.priority,
    parent_id: doc.parent_id || null,
    assignee_id: doc.assignee_id || null,
    blocked_by: safeParseJson(doc.blocked_by as string, []),
    tags: safeParseJson(doc.tags as string, []),
    project: doc.project || null,
  };
}

function safeParseJson(json: string, fallback: unknown): unknown {
  if (!json) return fallback;
  try {
    return JSON.parse(json);
  } catch {
    return fallback;
  }
}
