import { z } from "zod";

const envSchema = z.object({
  APPWRITE_ENDPOINT: z.string().url("APPWRITE_ENDPOINT must be a valid URL"),
  APPWRITE_PROJECT_ID: z.string().min(1, "APPWRITE_PROJECT_ID is required"),
  APPWRITE_API_KEY: z.string().min(1, "APPWRITE_API_KEY is required"),
  APPWRITE_DATABASE_ID: z.string().min(1, "APPWRITE_DATABASE_ID is required"),
  NEXT_PUBLIC_APP_URL: z.string().url().default("http://localhost:3000"),
});

export type Env = z.infer<typeof envSchema>;

let cached: Env | null = null;

export function getEnv(): Env {
  if (cached) return cached;
  const result = envSchema.safeParse(process.env);
  if (!result.success) {
    const missing = result.error.issues
      .map((i) => `  ${i.path.join(".")}: ${i.message}`)
      .join("\n");
    throw new Error(
      `Missing or invalid environment variables:\n${missing}`,
    );
  }
  cached = result.data;
  return cached;
}
