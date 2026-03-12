import { Client } from "appwrite";

let browserClient: Client | null = null;

export function getBrowserClient(): Client {
  if (browserClient) return browserClient;

  const endpoint = process.env.NEXT_PUBLIC_APPWRITE_ENDPOINT;
  const projectId = process.env.NEXT_PUBLIC_APPWRITE_PROJECT_ID;

  if (!endpoint || !projectId) {
    throw new Error(
      "Missing NEXT_PUBLIC_APPWRITE_ENDPOINT or NEXT_PUBLIC_APPWRITE_PROJECT_ID. " +
        "These must be set for realtime subscriptions to work."
    );
  }

  browserClient = new Client()
    .setEndpoint(endpoint)
    .setProject(projectId);

  return browserClient;
}

export function resetBrowserClient(): void {
  browserClient = null;
}
