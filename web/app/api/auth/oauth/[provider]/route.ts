import { NextRequest } from "next/server";
import { handleError } from "@/lib/response";
import { AppError, ErrorCode } from "@/lib/errors";
import { getEnv } from "@/lib/env";

const SUPPORTED_PROVIDERS = ["github", "google"] as const;

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ provider: string }> },
) {
  try {
    const { provider } = await params;
    validateProvider(provider);
    const url = buildOAuthUrl(provider, request);
    return Response.redirect(url);
  } catch (err) {
    return handleError(err);
  }
}

function validateProvider(provider: string): asserts provider is (typeof SUPPORTED_PROVIDERS)[number] {
  if (!SUPPORTED_PROVIDERS.includes(provider as (typeof SUPPORTED_PROVIDERS)[number])) {
    throw new AppError(
      ErrorCode.VALIDATION_ERROR,
      `Unsupported OAuth provider: ${provider}. Supported: ${SUPPORTED_PROVIDERS.join(", ")}`,
    );
  }
}

function buildOAuthUrl(provider: string, request: NextRequest): string {
  const env = getEnv();
  const callbackParam = request.nextUrl.searchParams.get("callback");

  const successUrl = buildSuccessUrl(env.NEXT_PUBLIC_APP_URL, callbackParam);
  const failureUrl = `${env.NEXT_PUBLIC_APP_URL}/auth/error`;

  const url = new URL(
    `${env.APPWRITE_ENDPOINT}/account/sessions/oauth2/${provider}`,
  );
  url.searchParams.set("project", env.APPWRITE_PROJECT_ID);
  url.searchParams.set("success", successUrl);
  url.searchParams.set("failure", failureUrl);

  return url.toString();
}

function buildSuccessUrl(appUrl: string, callback: string | null): string {
  const successUrl = new URL(`${appUrl}/api/auth/callback`);
  if (callback) {
    successUrl.searchParams.set("callback", callback);
  }
  return successUrl.toString();
}
