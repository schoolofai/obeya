import { z } from "zod";
import { ID } from "node-appwrite";
import { NextResponse } from "next/server";
import { handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { AppError, ErrorCode } from "@/lib/errors";
import { getEnv } from "@/lib/env";
import { getUsers } from "@/lib/appwrite/server";

const signupSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
  name: z.string().min(1),
});

interface AppwriteSession {
  $id: string;
  userId: string;
  secret: string;
}

export async function POST(request: Request) {
  try {
    const body = await validateBody(request, signupSchema);
    const user = await createUser(body);
    const session = await createSession(body);

    const res = NextResponse.json(
      {
        ok: true,
        data: {
          user: { id: user.$id, email: user.email, name: user.name },
          session: { id: session.$id },
        },
      },
      { status: 201 },
    );

    res.cookies.set("obeya_session", session.userId, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 60 * 60 * 24 * 30, // 30 days
    });

    return res;
  } catch (err) {
    return handleError(err);
  }
}

async function createUser(body: z.infer<typeof signupSchema>) {
  try {
    return await getUsers().create(
      ID.unique(),
      body.email,
      undefined,
      body.password,
      body.name,
    );
  } catch (err: unknown) {
    if (isAppwriteConflict(err)) {
      throw new AppError(ErrorCode.EMAIL_ALREADY_EXISTS, "Email already exists");
    }
    throw err;
  }
}

async function createSession(body: z.infer<typeof signupSchema>): Promise<AppwriteSession> {
  const env = getEnv();
  const url = `${env.APPWRITE_ENDPOINT}/account/sessions/email`;

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Appwrite-Project": env.APPWRITE_PROJECT_ID,
    },
    body: JSON.stringify({ email: body.email, password: body.password }),
  });

  if (!res.ok) {
    const errBody = await res.json().catch(() => null);
    throw new AppError(
      ErrorCode.INTERNAL_ERROR,
      errBody?.message ?? `Session creation failed (${res.status})`,
    );
  }

  return await res.json() as AppwriteSession;
}

function isAppwriteConflict(err: unknown): boolean {
  return (
    typeof err === "object" &&
    err !== null &&
    "code" in err &&
    (err as { code: number }).code === 409
  );
}
