import { Query } from "node-appwrite";
import { z } from "zod";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody, validateParams } from "@/lib/validation";
import { requireOrgRole } from "@/lib/permissions";

const paramsSchema = z.object({ id: z.string().min(1) });

const updateOrgSchema = z
  .object({ name: z.string().min(1).optional() })
  .refine((data) => Object.keys(data).some((k) => data[k as keyof typeof data] !== undefined), {
    message: "At least one field must be provided",
  });

type RouteContext = { params: Promise<{ id: string }> };

export async function GET(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    await requireOrgRole(user.id, id, "member");

    const db = getDatabases();
    const env = getEnv();

    const [org, members] = await Promise.all([
      db.getDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORGS, id),
      db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORG_MEMBERS, [
        Query.equal("org_id", id),
        Query.limit(1),
      ]),
    ]);

    return ok(serializeOrgWithMemberCount(org, members.total));
  } catch (err) {
    return handleError(err);
  }
}

export async function PATCH(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    await requireOrgRole(user.id, id, "admin");

    const input = await validateBody(request, updateOrgSchema);
    const updated = await updateOrg(id, input);
    return ok(serializeOrg(updated));
  } catch (err) {
    return handleError(err);
  }
}

export async function DELETE(request: Request, context: RouteContext) {
  try {
    const user = await authenticate(request);
    const { id } = validateParams(await context.params, paramsSchema);
    await requireOrgRole(user.id, id, "owner");

    await deleteOrgAndMembers(id);
    return ok({ deleted: true, id });
  } catch (err) {
    return handleError(err);
  }
}

async function updateOrg(id: string, input: { name?: string }): Promise<Record<string, unknown>> {
  const db = getDatabases();
  const env = getEnv();
  const payload: Record<string, unknown> = {};
  if (input.name !== undefined) payload.name = input.name;

  return await db.updateDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORGS, id, payload);
}

async function deleteOrgAndMembers(orgId: string): Promise<void> {
  const db = getDatabases();
  const env = getEnv();

  const members = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORG_MEMBERS, [
    Query.equal("org_id", orgId),
    Query.limit(500),
  ]);

  await Promise.all(
    members.documents.map((m) =>
      db.deleteDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORG_MEMBERS, m.$id)
    )
  );

  await db.deleteDocument(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORGS, orgId);
}

function serializeOrg(doc: Record<string, unknown>): Record<string, unknown> {
  return {
    id: doc.$id,
    name: doc.name,
    slug: doc.slug,
    plan: doc.plan,
    owner_id: doc.owner_id,
    created_at: doc.created_at,
  };
}

function serializeOrgWithMemberCount(
  doc: Record<string, unknown>,
  memberCount: number
): Record<string, unknown> {
  return { ...serializeOrg(doc), member_count: memberCount };
}
