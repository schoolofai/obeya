"use client";

import { Suspense } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";

function ErrorContent() {
  const searchParams = useSearchParams();
  const message =
    searchParams.get("message") ?? "An authentication error occurred";

  return (
    <div className="space-y-6 text-center">
      <ErrorIcon />
      <div>
        <h2 className="text-xl font-semibold text-gray-900">
          Authentication Error
        </h2>
        <p className="mt-2 text-sm text-gray-600">{message}</p>
      </div>
      <Link href="/auth/login">
        <Button variant="primary" fullWidth>
          Back to Login
        </Button>
      </Link>
    </div>
  );
}

export default function AuthErrorPage() {
  return (
    <Suspense fallback={<p className="text-sm text-gray-500">Loading…</p>}>
      <ErrorContent />
    </Suspense>
  );
}

function ErrorIcon() {
  return (
    <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-red-100">
      <span className="text-xl text-red-600" aria-hidden>
        !
      </span>
    </div>
  );
}
