import { Query } from "node-appwrite";
import { z } from "zod";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody, validateParams } from "@/lib/validation";
import { requireOrgRole } from "@/lib/permissions";
import { AppError, ErrorCode } from "@/lib/errors";

const paramsSchema = z.object({
  id: z.string().min(1),
  uid: z.string().min(1),
});

const updateMemberSchema = z.object({
  role: z.enum(["admin", "member"]),
});

type RouteContext = { params: Promise<{ id: string; uid: string }> };

export async function PATCH(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id, uid } = validateParams(await context.params, paramsSchema);
    await requireOrgRole(user.id, id, "owner");

    const input = await validateBody(request, updateMemberSchema);
    const membership = await findMembership(id, uid);

    if (membership.role === "owner") {
      throw new AppError(ErrorCode.FORBIDDEN, "Cannot change the owner's role");
    }

    const updated = await updateMemberRole(membership.$id as string, input.role);
    return ok(updated);
  } catch (err) {
    return handleError(err);
  }
}

export async function DELETE(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id, uid } = validateParams(await context.params, paramsSchema);

    const isSelf = user.id === uid;
    await requireOrgRole(user.id, id, isSelf ? "member" : "admin");

    const membership = await findMembership(id, uid);

    if (membership.role === "owner") {
      throw new AppError(ErrorCode.FORBIDDEN, "Cannot remove the org owner");
    }

    const db = getDatabases();
    const env = getEnv();
    await db.deleteDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORG_MEMBERS, membership.$id as string);

    return ok({ deleted: true, user_id: uid });
  } catch (err) {
    return handleError(err);
  }
}

async function findMembership(orgId: string, userId: string): Promise<Record<string, unknown>> {
  const db = getDatabases();
  const env = getEnv();

  const result = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORG_MEMBERS, [
    Query.equal("org_id", orgId),
    Query.equal("user_id", userId),
    Query.limit(1),
  ]);

  if (result.documents.length === 0) {
    throw new AppError(ErrorCode.USER_NOT_FOUND, `User ${userId} is not a member of this org`);
  }

  return result.documents[0];
}

async function updateMemberRole(
  membershipId: string,
  role: string
): Promise<Record<string, unknown>> {
  const db = getDatabases();
  const env = getEnv();

  return await db.updateDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.ORG_MEMBERS,
    membershipId,
    { role }
  );
}
