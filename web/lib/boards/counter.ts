import { getDatabases } from "@/lib/appwrite/server";
import { getEnv } from "@/lib/env";
import { COLLECTIONS } from "@/lib/appwrite/collections";
import { AppError, ErrorCode } from "@/lib/errors";

const MAX_RETRIES = 3;

export async function incrementDisplayCounter(boardId: string): Promise<number> {
  const db = getDatabases();
  const env = getEnv();

  for (let attempt = 0; attempt < MAX_RETRIES; attempt++) {
    const nextCounter = await tryIncrement(db, env.APPWRITE_DATABASE_ID, boardId);
    if (nextCounter !== null) return nextCounter;
  }

  throw new AppError(
    ErrorCode.COUNTER_CONFLICT,
    `Failed to increment display counter after ${MAX_RETRIES} retries`
  );
}

async function tryIncrement(
  db: ReturnType<typeof getDatabases>,
  databaseId: string,
  boardId: string
): Promise<number | null> {
  const board = await db.getDocument(databaseId, COLLECTIONS.BOARDS, boardId);
  const currentCounter = board.display_counter as number;
  const nextCounter = currentCounter + 1;

  try {
    await db.updateDocument(databaseId, COLLECTIONS.BOARDS, boardId, {
      display_counter: nextCounter,
    });
    return nextCounter;
  } catch (err: unknown) {
    if (isConflictError(err)) return null;
    throw err;
  }
}

function isConflictError(err: unknown): boolean {
  return err instanceof Error && (err as any).code === 409;
}
