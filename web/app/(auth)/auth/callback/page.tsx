"use client";

import { useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";

export default function CallbackPage() {
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

  return (
    <div className="flex h-screen items-center justify-center">
      <p className="text-sm text-gray-500">Completing sign in…</p>
    </div>
  );
}
