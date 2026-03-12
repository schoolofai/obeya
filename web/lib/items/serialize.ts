export interface Item {
  id: string;
  board_id: string;
  display_num: number;
  type: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  parent_id: string | null;
  assignee_id: string | null;
  blocked_by: string[];
  tags: string[];
  project: string | null;
  created_at: string;
  updated_at: string;
}

export function serializeItem(
  item: Partial<{ blocked_by: string[]; tags: string[] }> & Record<string, unknown>
): Record<string, unknown> {
  const result: Record<string, unknown> = { ...item };
  if (item.blocked_by !== undefined) result.blocked_by = JSON.stringify(item.blocked_by);
  if (item.tags !== undefined) result.tags = JSON.stringify(item.tags);
  delete result.id;
  return result;
}

export function deserializeItem(doc: Record<string, unknown>): Item {
  return {
    id: doc.$id as string,
    board_id: doc.board_id as string,
    display_num: doc.display_num as number,
    type: doc.type as string,
    title: doc.title as string,
    description: (doc.description as string) || "",
    status: doc.status as string,
    priority: doc.priority as string,
    parent_id: (doc.parent_id as string) || null,
    assignee_id: (doc.assignee_id as string) || null,
    blocked_by: parseJsonArray(doc.blocked_by as string),
    tags: parseJsonArray(doc.tags as string),
    project: (doc.project as string) || null,
    created_at: doc.created_at as string,
    updated_at: doc.updated_at as string,
  };
}

function parseJsonArray(json: string): string[] {
  if (!json) return [];
  return JSON.parse(json) as string[];
}
