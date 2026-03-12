/**
 * Database setup script — creates the Appwrite database and all collections
 * with their attributes. Safe to re-run: handles 409 (already exists) gracefully.
 *
 * Usage: npx tsx scripts/setup-db.ts
 * Requires: APPWRITE_ENDPOINT, APPWRITE_PROJECT_ID, APPWRITE_API_KEY, APPWRITE_DATABASE_ID
 */

import { Client, Databases } from "node-appwrite";
import { SCHEMAS, type Attribute, type CollectionSchema } from "./schema";

const ENDPOINT = requireEnv("APPWRITE_ENDPOINT");
const PROJECT_ID = requireEnv("APPWRITE_PROJECT_ID");
const API_KEY = requireEnv("APPWRITE_API_KEY");
const DATABASE_ID = requireEnv("APPWRITE_DATABASE_ID");

function requireEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Missing required environment variable: ${name}`);
  }
  return value;
}

function isAlreadyExists(err: unknown): boolean {
  if (err && typeof err === "object" && "code" in err) {
    return (err as { code: number }).code === 409;
  }
  return false;
}

function buildClient(): Databases {
  const client = new Client()
    .setEndpoint(ENDPOINT)
    .setProject(PROJECT_ID)
    .setKey(API_KEY);
  return new Databases(client);
}

async function createDatabase(db: Databases): Promise<void> {
  try {
    await db.create({ databaseId: DATABASE_ID, name: "obeya" });
    console.log(`[db] Created database "${DATABASE_ID}"`);
  } catch (err) {
    if (isAlreadyExists(err)) {
      console.log(`[db] Database "${DATABASE_ID}" already exists`);
      return;
    }
    throw err;
  }
}

async function createCollection(db: Databases, schema: CollectionSchema): Promise<void> {
  try {
    await db.createCollection({
      databaseId: DATABASE_ID,
      collectionId: schema.id,
      name: schema.name,
    });
    console.log(`[collection] Created "${schema.id}"`);
  } catch (err) {
    if (isAlreadyExists(err)) {
      console.log(`[collection] "${schema.id}" already exists`);
    } else {
      throw err;
    }
  }
}

async function createAttribute(db: Databases, collectionId: string, attr: Attribute): Promise<void> {
  const base = { databaseId: DATABASE_ID, collectionId };
  const label = `${collectionId}.${attr.key}`;

  try {
    switch (attr.type) {
      case "string":
        await db.createStringAttribute({ ...base, key: attr.key, size: attr.size, required: attr.required });
        break;
      case "integer":
        await db.createIntegerAttribute({ ...base, key: attr.key, required: attr.required, min: attr.min, max: attr.max });
        break;
      case "enum":
        await db.createEnumAttribute({ ...base, key: attr.key, elements: attr.elements, required: attr.required });
        break;
      case "datetime":
        await db.createDatetimeAttribute({ ...base, key: attr.key, required: attr.required });
        break;
    }
    console.log(`  [attr] Created "${label}" (${attr.type})`);
  } catch (err) {
    if (isAlreadyExists(err)) {
      console.log(`  [attr] "${label}" already exists`);
      return;
    }
    throw err;
  }
}

async function setupCollection(db: Databases, schema: CollectionSchema): Promise<void> {
  await createCollection(db, schema);
  for (const attr of schema.attributes) {
    await createAttribute(db, schema.id, attr);
  }
}

async function main(): Promise<void> {
  console.log("Setting up Obeya Cloud database...\n");
  const db = buildClient();

  await createDatabase(db);

  for (const schema of SCHEMAS) {
    await setupCollection(db, schema);
  }

  console.log("\nDatabase setup complete.");
}

main().catch((err) => {
  console.error("Database setup failed:", err);
  process.exit(1);
});
