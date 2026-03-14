import { getUsers } from "@/lib/appwrite/server";
import { AppError, ErrorCode } from "@/lib/errors";

export interface AuthUser {
  id: string;
  email: string;
  name: string;
}

export async function getUserFromSession(cookieHeader: string): Promise<AuthUser> {
  const userId = parseCookie(cookieHeader, "obeya_session");
  if (!userId) {
    throw new AppError(ErrorCode.UNAUTHORIZED, "No session cookie");
  }

  try {
    const user = await getUsers().get(userId);
    return { id: user.$id, email: user.email, name: user.name };
  } catch {
    throw new AppError(ErrorCode.UNAUTHORIZED, "Invalid session");
  }
}

function parseCookie(header: string, name: string): string | null {
  const match = header.match(new RegExp(`(?:^|;\\s*)${name}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
}
