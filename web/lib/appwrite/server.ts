import { Client, Databases, Users } from "node-appwrite";
import { getEnv } from "@/lib/env";

let client: Client | null = null;

function getClient(): Client {
  if (client) return client;
  const env = getEnv();
  client = new Client()
    .setEndpoint(env.APPWRITE_ENDPOINT)
    .setProject(env.APPWRITE_PROJECT_ID)
    .setKey(env.APPWRITE_API_KEY);
  return client;
}

export function getDatabases(): Databases {
  return new Databases(getClient());
}

export function getUsers(): Users {
  return new Users(getClient());
}

export function getServerClient(): Client {
  return getClient();
}
