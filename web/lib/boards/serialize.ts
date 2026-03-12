export interface Column {
  name: string;
  limit: number;
}

export interface BoardDocument {
  $id: string;
  name: string;
  owner_id: string;
  org_id: string | null;
  display_counter: number;
  columns: string;
  display_map: string;
  users: string;
  projects: string;
  agent_role: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface Board {
  id: string;
  name: string;
  owner_id: string;
  org_id: string | null;
  display_counter: number;
  columns: Column[];
  display_map: Record<string, string>;
  users: Record<string, unknown>;
  projects: Record<string, unknown>;
  agent_role: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export function serializeColumns(columns: Column[]): string {
  return JSON.stringify(columns);
}

export function deserializeColumns(json: string): Column[] {
  if (!json) return [];
  return JSON.parse(json) as Column[];
}

export function serializeBoard(
  board: Partial<Board> & {
    columns?: Column[];
    display_map?: Record<string, string>;
    users?: Record<string, unknown>;
    projects?: Record<string, unknown>;
  }
): Record<string, unknown> {
  const result: Record<string, unknown> = { ...board };
  if (board.columns !== undefined) result.columns = JSON.stringify(board.columns);
  if (board.display_map !== undefined) result.display_map = JSON.stringify(board.display_map);
  if (board.users !== undefined) result.users = JSON.stringify(board.users);
  if (board.projects !== undefined) result.projects = JSON.stringify(board.projects);
  delete result.id;
  return result;
}

export function deserializeBoard(doc: Record<string, unknown>): Board {
  return {
    id: doc.$id as string,
    name: doc.name as string,
    owner_id: doc.owner_id as string,
    org_id: (doc.org_id as string) || null,
    display_counter: doc.display_counter as number,
    columns: deserializeColumns(doc.columns as string),
    display_map: parseJsonField(doc.display_map as string, {}) as Record<string, string>,
    users: parseJsonField(doc.users as string, {}) as Record<string, unknown>,
    projects: parseJsonField(doc.projects as string, {}) as Record<string, unknown>,
    agent_role: doc.agent_role as string,
    version: doc.version as number,
    created_at: doc.created_at as string,
    updated_at: doc.updated_at as string,
  };
}

function parseJsonField(json: string, fallback: unknown = {}): unknown {
  if (!json) return fallback;
  return JSON.parse(json);
}
