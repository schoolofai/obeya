import { randomBytes } from "crypto";
import bcrypt from "bcryptjs";

const TOKEN_PREFIX = "ob_tok_";
const SALT_ROUNDS = 10;

export function generateToken(): string {
  const bytes = randomBytes(32);
  return TOKEN_PREFIX + bytes.toString("hex");
}

export async function hashToken(token: string): Promise<string> {
  return bcrypt.hash(token, SALT_ROUNDS);
}

export async function verifyToken(token: string, hash: string): Promise<boolean> {
  return bcrypt.compare(token, hash);
}
