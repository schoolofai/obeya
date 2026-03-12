import { Client, Account } from "node-appwrite";
import { getEnv } from "@/lib/env";

export interface AuthUser {
  id: string;
  email: string;
  name: string;
}

export async function getUserFromSession(sessionCookie: string): Promise<AuthUser> {
  const env = getEnv();
  const client = new Client()
    .setEndpoint(env.APPWRITE_ENDPOINT)
    .setProject(env.APPWRITE_PROJECT_ID)
    .setSession(sessionCookie);
  const account = new Account(client);
  const user = await account.get();
  return { id: user.$id, email: user.email, name: user.name };
}
