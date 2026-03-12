import { z } from "zod";
import { Client, Account } from "node-appwrite";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { AppError, ErrorCode } from "@/lib/errors";
import { getEnv } from "@/lib/env";

const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
});

export async function POST(request: Request) {
  try {
    const body = await validateBody(request, loginSchema);
    const session = await createSession(body);
    const user = await getSessionUser(session.secret);
    return ok({
      user: { id: user.$id, email: user.email, name: user.name },
      session: { id: session.$id, secret: session.secret },
    });
  } catch (err) {
    return handleError(err);
  }
}

async function createSession(body: z.infer<typeof loginSchema>) {
  const env = getEnv();
  const client = new Client()
    .setEndpoint(env.APPWRITE_ENDPOINT)
    .setProject(env.APPWRITE_PROJECT_ID);

  const account = new Account(client);

  try {
    return await account.createEmailPasswordSession(body.email, body.password);
  } catch (err: unknown) {
    if (isAppwriteUnauthorized(err)) {
      throw new AppError(ErrorCode.INVALID_CREDENTIALS, "Invalid email or password");
    }
    throw err;
  }
}

async function getSessionUser(sessionSecret: string) {
  const env = getEnv();
  const client = new Client()
    .setEndpoint(env.APPWRITE_ENDPOINT)
    .setProject(env.APPWRITE_PROJECT_ID)
    .setSession(sessionSecret);

  const account = new Account(client);
  return await account.get();
}

function isAppwriteUnauthorized(err: unknown): boolean {
  return (
    typeof err === "object" &&
    err !== null &&
    "code" in err &&
    (err as { code: number }).code === 401
  );
}
