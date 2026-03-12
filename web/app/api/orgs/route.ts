import { ID, Query } from "node-appwrite";
import { z } from "zod";
import { authenticate } from "@/lib/auth/middleware";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { generateSlug, ensureUniqueSlug } from "@/lib/slugs";
import { enforceOrgLimit } from "@/lib/limits";

const createOrgSchema = z.object({
  name: z.string().min(1, "Name is required"),
  slug: z.string().min(1).optional(),
});

export async function GET(request: Request) {
  try {
    const user = await authenticate(request);
    const orgs = await listUserOrgs(user.id);
    return ok(orgs);
  } catch (err) {
    return handleError(err);
  }
}

export async function POST(request: Request) {
  try {
    const user = await authenticate(request);
    const input = await validateBody(request, createOrgSchema);
    await enforceOrgLimit(user.id);

    const baseSlug = input.slug ? generateSlug(input.slug) : generateSlug(input.name);
    const slug = await ensureUniqueSlug(baseSlug);

    const org = await createOrg(user.id, input.name, slug);
    return ok(org, { status: 201 });
  } catch (err) {
    return handleError(err);
  }
}

async function listUserOrgs(userId: string): Promise<unknown[]> {
  const db = getDatabases();
  const env = getEnv();

  const memberships = await db.listDocuments(env.APPWRITE_DATABASE_ID, COLLECTIONS.ORG_MEMBERS, [
    Query.equal("user_id", userId),
    Query.limit(100),
  ]);

  if (memberships.documents.length === 0) return [];

  return Promise.all(
    memberships.documents.map((mem) => fetchOrgWithRole(db, env.APPWRITE_DATABASE_ID, mem))
  );
}

async function fetchOrgWithRole(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  membership: Record<string, unknown>
): Promise<unknown> {
  const org = await db.getDocument(databaseId, COLLECTIONS.ORGS, membership.org_id as string);
  return serializeOrg(org, membership.role as string);
}

async function createOrg(
  userId: string,
  name: string,
  slug: string
): Promise<unknown> {
  const db = getDatabases();
  const env = getEnv();
  const now = new Date().toISOString();

  const org = await db.createDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.ORGS,
    ID.unique(),
    { name, slug, plan: "free", owner_id: userId, created_at: now }
  );

  await db.createDocument(
    env.APPWRITE_DATABASE_ID,
    COLLECTIONS.ORG_MEMBERS,
    ID.unique(),
    { user_id: userId, org_id: org.$id, role: "owner" }
  );

  return serializeOrg(org, "owner");
}

function serializeOrg(doc: Record<string, unknown>, role: string): Record<string, unknown> {
  return {
    id: doc.$id,
    name: doc.name,
    slug: doc.slug,
    plan: doc.plan,
    owner_id: doc.owner_id,
    created_at: doc.created_at,
    role,
  };
}
