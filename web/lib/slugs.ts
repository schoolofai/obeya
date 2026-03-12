import { AppError, ErrorCode } from "@/lib/errors";
import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { Query } from "node-appwrite";

const TRANSLITERATION: Record<string, string> = {
  ä: "ae",
  ö: "oe",
  ü: "ue",
  Ä: "Ae",
  Ö: "Oe",
  Ü: "Ue",
  ß: "ss",
  à: "a",
  á: "a",
  â: "a",
  ã: "a",
  å: "a",
  æ: "ae",
  ç: "c",
  è: "e",
  é: "e",
  ê: "e",
  ë: "e",
  ì: "i",
  í: "i",
  î: "i",
  ï: "i",
  ð: "d",
  ñ: "n",
  ò: "o",
  ó: "o",
  ô: "o",
  õ: "o",
  ø: "o",
  ù: "u",
  ú: "u",
  û: "u",
  ý: "y",
  þ: "th",
  ÿ: "y",
};

function transliterate(input: string): string {
  return input
    .split("")
    .map((char) => TRANSLITERATION[char] ?? char)
    .join("");
}

export function generateSlug(name: string): string {
  const transliterated = transliterate(name);
  const slug = transliterated
    .toLowerCase()
    .replace(/[^a-z0-9\s-]/g, "")
    .replace(/[\s]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-+|-+$/g, "");
  return slug || "org";
}

export async function ensureUniqueSlug(baseSlug: string): Promise<string> {
  const db = getDatabases();
  const env = getEnv();
  const maxAttempts = 20;

  const isSlugTaken = async (slug: string): Promise<boolean> => {
    const result = await db.listDocuments(
      env.APPWRITE_DATABASE_ID,
      COLLECTIONS.ORGS,
      [Query.equal("slug", slug), Query.limit(1)]
    );
    return result.total > 0;
  };

  if (!(await isSlugTaken(baseSlug))) {
    return baseSlug;
  }

  for (let i = 1; i <= maxAttempts; i++) {
    const candidate = `${baseSlug}-${i}`;
    if (!(await isSlugTaken(candidate))) {
      return candidate;
    }
  }

  throw new AppError(
    ErrorCode.SLUG_ALREADY_EXISTS,
    `Could not generate unique slug after ${maxAttempts} attempts`
  );
}
