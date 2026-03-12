import { z } from "zod";
import { ID } from "node-appwrite";
import { ok, handleError } from "@/lib/response";
import { validateBody } from "@/lib/validation";
import { AppError, ErrorCode } from "@/lib/errors";
import { getUsers } from "@/lib/appwrite/server";

const signupSchema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
  name: z.string().min(1),
});

export async function POST(request: Request) {
  try {
    const body = await validateBody(request, signupSchema);
    const user = await createUser(body);
    return ok({ id: user.$id, email: user.email, name: user.name }, { status: 201 });
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

function isAppwriteConflict(err: unknown): boolean {
  return (
    typeof err === "object" &&
    err !== null &&
    "code" in err &&
    (err as { code: number }).code === 409
  );
}
