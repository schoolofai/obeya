import { z } from "zod";
import { NextResponse } from "next/server";
import { handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { AppError, ErrorCode } from "@/lib/errors";
import { getEnv } from "@/lib/env";
import { getUsers } from "@/lib/appwrite/server";

const loginSchema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
});

interface AppwriteSession {
  $id: string;
  userId: string;
  secret: string;
}

export async function POST(request: Request) {
  try {
    const body = await validateBody(request, loginSchema);
    const session = await createSession(body);
    const user = await getUsers().get(session.userId);

    const res = NextResponse.json({
      ok: true,
      data: {
        user: { id: user.$id, email: user.email, name: user.name },
        session: { id: session.$id },
      },
    });

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

async function createSession(body: z.infer<typeof loginSchema>): Promise<AppwriteSession> {
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
    if (res.status === 401) {
      throw new AppError(ErrorCode.INVALID_CREDENTIALS, "Invalid email or password");
    }
    throw new AppError(
      ErrorCode.INTERNAL_ERROR,
      errBody?.message ?? `Appwrite auth failed (${res.status})`
    );
  }

  return await res.json() as AppwriteSession;
}
