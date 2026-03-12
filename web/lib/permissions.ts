import { AppError, ErrorCode } from "@/lib/errors";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { Query } from "node-appwrite";

export const ORG_ROLE_LEVEL: Record<string, number> = {
  member: 1,
  admin: 2,
  owner: 3,
};

export const BOARD_ROLE_LEVEL: Record<string, number> = {
  viewer: 1,
  editor: 2,
  owner: 3,
};

// org member role → equivalent board level mapping
const ORG_ROLE_TO_BOARD_LEVEL: Record<string, number> = {
  member: BOARD_ROLE_LEVEL.viewer,
  admin: BOARD_ROLE_LEVEL.editor,
  owner: BOARD_ROLE_LEVEL.owner,
};

export interface EffectivePermission {
  level: number;
  source: "org" | "board" | "none";
}

export async function resolvePermission(
  userId: string,
  boardId: string,
  orgId: string | null
): Promise<EffectivePermission> {
  const orgLevel = orgId ? await getOrgMemberLevel(userId, orgId) : null;
  const boardLevel = await getBoardMemberLevel(userId, boardId);

  if (orgLevel !== null && boardLevel !== null) {
    return orgLevel >= boardLevel
      ? { level: orgLevel, source: "org" }
      : { level: boardLevel, source: "board" };
  }

  if (boardLevel !== null) {
    return { level: boardLevel, source: "board" };
  }

  if (orgLevel !== null) {
    return { level: orgLevel, source: "org" };
  }

  return { level: 0, source: "none" };
}

async function getOrgMemberLevel(userId: string, orgId: string): Promise<number | null> {
  const db = getDatabases();
  const env = getEnv();

  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORG_MEMBERS, [
    Query.equal("user_id", userId),
    Query.equal("org_id", orgId),
    Query.limit(1),
  ]);

  if (result.documents.length === 0) return null;
  const role = result.documents[0].role as string;
  return ORG_ROLE_TO_BOARD_LEVEL[role] ?? BOARD_ROLE_LEVEL.viewer;
}

async function getBoardMemberLevel(userId: string, boardId: string): Promise<number | null> {
  const db = getDatabases();
  const env = getEnv();

  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.BOARD_MEMBERS, [
    Query.equal("user_id", userId),
    Query.equal("board_id", boardId),
    Query.limit(1),
  ]);

  if (result.documents.length === 0) return null;
  const role = result.documents[0].role as string;
  return BOARD_ROLE_LEVEL[role] ?? BOARD_ROLE_LEVEL.viewer;
}

export async function requireOrgRole(
  userId: string,
  orgId: string,
  minimumRole: "member" | "admin" | "owner"
): Promise<void> {
  const db = getDatabases();
  const env = getEnv();
  const minLevel = ORG_ROLE_LEVEL[minimumRole];

  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORG_MEMBERS, [
    Query.equal("user_id", userId),
    Query.equal("org_id", orgId),
    Query.limit(1),
  ]);

  if (result.documents.length === 0) {
    throw new AppError(ErrorCode.FORBIDDEN, "You are not a member of this organization");
  }

  const role = result.documents[0].role as string;
  const userLevel = ORG_ROLE_LEVEL[role] ?? 0;

  if (userLevel < minLevel) {
    throw new AppError(ErrorCode.FORBIDDEN, "Insufficient organization role");
  }
}

export async function requireBoardAccess(
  userId: string,
  boardId: string,
  orgId: string | null,
  minimumLevel: "viewer" | "editor" | "owner"
): Promise<void> {
  const minLevel = BOARD_ROLE_LEVEL[minimumLevel];
  const permission = await resolvePermission(userId, boardId, orgId);

  if (permission.level < minLevel) {
    throw new AppError(ErrorCode.FORBIDDEN, "You do not have sufficient access to this board");
  }
}
