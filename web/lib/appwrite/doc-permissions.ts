import { Permission, Role } from "node-appwrite";

export interface BoardMember {
  userId: string;
  role: "viewer" | "editor" | "owner";
}

/** Build $permissions array for a board document. */
export function buildBoardPermissions(
  ownerId: string,
  members: BoardMember[],
): string[] {
  const perms: string[] = [
    Permission.read(Role.user(ownerId)),
    Permission.update(Role.user(ownerId)),
    Permission.delete(Role.user(ownerId)),
  ];
  for (const m of members) {
    perms.push(Permission.read(Role.user(m.userId)));
    if (m.role !== "viewer") {
      perms.push(Permission.update(Role.user(m.userId)));
    }
  }
  return [...new Set(perms)];
}

/** Build $permissions array for an item document. */
export function buildItemPermissions(
  ownerId: string,
  members: BoardMember[],
): string[] {
  const perms: string[] = [
    Permission.read(Role.user(ownerId)),
    Permission.update(Role.user(ownerId)),
    Permission.delete(Role.user(ownerId)),
  ];
  for (const m of members) {
    perms.push(Permission.read(Role.user(m.userId)));
    if (m.role !== "viewer") {
      perms.push(Permission.update(Role.user(m.userId)));
    }
  }
  return [...new Set(perms)];
}

/** Fetch board members from Appwrite and return as BoardMember[]. */
export async function fetchBoardMemberList(
  boardId: string,
): Promise<BoardMember[]> {
  const { getDatabases } = await import("@/lib/appwrite/server");
  const { getEnv } = await import("@/lib/env");
  const { COLLECTIONS } = await import("@/lib/appwrite/collections");
  const { Query } = await import("node-appwrite");

  const db = getDatabases();
  const env = getEnv();
  const result = await db.listDocuments(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.BOARD_MEMBERS,
    [Query.equal("board_id", boardId), Query.limit(500)],
  );
  return result.documents.map((doc: any) => ({
    userId: doc.user_id as string,
    role: doc.role as "viewer" | "editor" | "owner",
  }));
}
