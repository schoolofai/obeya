import { authenticate } from "@/lib/auth/middleware";
import { ok, handleError } from "@/lib/response";

export async function GET(request: Request) {
  try {
    const user = await authenticate(request);
    return ok(user);
  } catch (err) {
    return handleError(err);
  }
}
