"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiClient, ApiClientError } from "@/lib/api-client";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      await apiClient.post("/api/auth/login", { email, password });
      router.replace("/dashboard");
    } catch (err) {
      const message =
        err instanceof ApiClientError ? err.message : "Login failed";
      setError(message);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageTitle />
      <form onSubmit={handleSubmit} className="space-y-4">
        <Input
          label="Email"
          name="email"
          type="email"
          placeholder="you@example.com"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
        />
        <Input
          label="Password"
          name="password"
          type="password"
          placeholder="••••••••"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
        />
        {error && (
          <p className="font-mono text-sm text-[#f85149]">{error}</p>
        )}
        <Button
          type="submit"
          variant="primary"
          fullWidth
          disabled={loading}
        >
          {loading ? "Signing in…" : "Sign in"}
        </Button>
      </form>
      <OAuthButtons />
      <p className="text-center font-mono text-sm text-[#8b949e]">
        {"Don't have an account? "}
        <Link
          href="/auth/signup"
          className="font-medium text-[#7aa2f7] hover:underline"
        >
          Sign up
        </Link>
      </p>
    </div>
  );
}

function PageTitle() {
  return (
    <div className="text-center">
      <h2 className="font-mono text-xl font-semibold text-[#c9d1d9]">
        Welcome back
      </h2>
      <p className="mt-1 font-mono text-sm text-[#8b949e]">
        Sign in to your account
      </p>
    </div>
  );
}

function OAuthButtons() {
  return (
    <div className="space-y-3">
      <div className="relative">
        <div className="absolute inset-0 flex items-center">
          <div className="w-full border-t border-[#30363d]" />
        </div>
        <div className="relative flex justify-center text-xs">
          <span className="bg-[#161b22] px-2 font-mono text-[#484f58]">
            or continue with
          </span>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <Button
          variant="secondary"
          onClick={() => {
            window.location.href = "/api/auth/oauth/github";
          }}
          type="button"
        >
          GitHub
        </Button>
        <Button
          variant="secondary"
          onClick={() => {
            window.location.href = "/api/auth/oauth/google";
          }}
          type="button"
        >
          Google
        </Button>
      </div>
    </div>
  );
}
