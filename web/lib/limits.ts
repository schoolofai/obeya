import { AppError, ErrorCode } from "@/lib/errors";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { Query } from "node-appwrite";

export const FREE_TIER_LIMITS = {
  PERSONAL_BOARDS: 3,
  ORGS: 1,
  MEMBERS_PER_ORG: 3,
  ITEMS_PER_BOARD: 100,
} as const;

const PAID_PLANS = new Set(["pro", "enterprise"]);

export async function enforcePersonalBoardLimit(userId: string): Promise<void> {
  const db = getDatabases();
  const env = getEnv();

  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.BOARDS, [
    Query.equal("owner_id", userId),
    Query.isNull("org_id"),
    Query.limit(1),
  ]);

  if (result.total >= FREE_TIER_LIMITS.PERSONAL_BOARDS) {
    throw new AppError(
      ErrorCode.PLAN_LIMIT_REACHED,
      `Personal board limit of ${FREE_TIER_LIMITS.PERSONAL_BOARDS} reached. Upgrade to pro for unlimited boards.`
    );
  }
}

export async function enforceOrgLimit(userId: string): Promise<void> {
  const db = getDatabases();
  const env = getEnv();

  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORG_MEMBERS, [
    Query.equal("user_id", userId),
    Query.equal("role", "owner"),
    Query.limit(1),
  ]);

  if (result.total >= FREE_TIER_LIMITS.ORGS) {
    throw new AppError(
      ErrorCode.PLAN_LIMIT_REACHED,
      `Organization limit of ${FREE_TIER_LIMITS.ORGS} reached on free plan. Upgrade to pro for more organizations.`
    );
  }
}

export async function enforceOrgMemberLimit(orgId: string, orgPlan: string): Promise<void> {
  if (PAID_PLANS.has(orgPlan)) return;

  const db = getDatabases();
  const env = getEnv();

  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORG_MEMBERS, [
    Query.equal("org_id", orgId),
    Query.limit(1),
  ]);

  if (result.total >= FREE_TIER_LIMITS.MEMBERS_PER_ORG) {
    throw new AppError(
      ErrorCode.PLAN_LIMIT_REACHED,
      `Member limit of ${FREE_TIER_LIMITS.MEMBERS_PER_ORG} reached for free organizations. Upgrade to pro for more members.`
    );
  }
}

export async function enforceBoardItemLimit(boardId: string): Promise<void> {
  const db = getDatabases();
  const env = getEnv();

  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ITEMS, [
    Query.equal("board_id", boardId),
    Query.limit(1),
  ]);

  if (result.total >= FREE_TIER_LIMITS.ITEMS_PER_BOARD) {
    throw new AppError(
      ErrorCode.PLAN_LIMIT_REACHED,
      `Item limit of ${FREE_TIER_LIMITS.ITEMS_PER_BOARD} reached for this board. Upgrade to pro for unlimited items.`
    );
  }
}
