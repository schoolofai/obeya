"use client";

import { Suspense, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";

function CallbackHandler() {
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const error = searchParams.get("error");
    if (error) {
      router.replace(`/auth/error?message=${encodeURIComponent(error)}`);
      return;
    }
    router.replace("/dashboard");
  }, [router, searchParams]);

  return <p className="text-sm text-gray-500">Completing sign in…</p>;
}

export default function CallbackPage() {
  return (
    <div className="flex h-screen items-center justify-center">
      <Suspense fallback={<p className="text-sm text-gray-500">Loading…</p>}>
        <CallbackHandler />
      </Suspense>
    </div>
  );
}
