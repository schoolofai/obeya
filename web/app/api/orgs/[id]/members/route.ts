import { ID, Query } from "node-appwrite";
import { z } from "zod";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody, validateParams } from "@/lib/validation";
import { requireOrgRole } from "@/lib/permissions";
import { enforceOrgMemberLimit } from "@/lib/limits";
import { AppError, ErrorCode } from "@/lib/errors";

const paramsSchema = z.object({ id: z.string().min(1) });

const addMemberSchema = z.object({
  user_id: z.string().min(1),
  role: z.enum(["admin", "member"]),
});

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    await requireOrgRole(user.id, id, "member");

    const members = await listOrgMembers(id);
    return ok(members);
  } catch (err) {
    return handleError(err);
  }
}

export async function POST(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    await requireOrgRole(user.id, id, "admin");

    const input = await validateBody(request, addMemberSchema);
    const db = getDatabases();
    const env = getEnv();

    const org = await db.getDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORGS, id);
    await enforceOrgMemberLimit(id, org.plan as string);

    await checkDuplicateMember(db, env.APPWRITE_DATABASE_ID, id, input.user_id);

    const member = await db.createDocument(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORG_MEMBERS,
      ID.unique(),
      { user_id: input.user_id, org_id: id, role: input.role }
    );

    return ok(member, { status: 201 });
  } catch (err) {
    return handleError(err);
  }
}

async function listOrgMembers(orgId: string): Promise<unknown[]> {
  const db = getDatabases();
  const env = getEnv();

  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORG_MEMBERS, [
    Query.equal("org_id", orgId),
    Query.limit(500),
  ]);

  return result.documents;
}

async function checkDuplicateMember(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  orgId: string,
  userId: string
): Promise<void> {
  const existing = await db.listDocuments(databaseId, COLLECTIONS.ORG_MEMBERS, [
    Query.equal("org_id", orgId),
    Query.equal("user_id", userId),
    Query.limit(1),
  ]);

  if (existing.documents.length > 0) {
    throw new AppError(ErrorCode.SLUG_ALREADY_EXISTS, "User is already a member of this org");
  }
}
