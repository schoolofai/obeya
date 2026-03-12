"use client";

import { useState, useEffect } from "react";
import { apiClient, ApiClientError } from "@/lib/api-client";
import type { User } from "@/lib/types";

interface AuthState {
  user: User | null;
  loading: boolean;
  error: ApiClientError | Error | null;
}

export function useAuth(): AuthState {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiClientError | Error | null>(null);

  useEffect(() => {
    fetchCurrentUser(setUser, setError, setLoading);
  }, []);

  return { user, loading, error };
}

async function fetchCurrentUser(
  setUser: (u: User | null) => void,
  setError: (e: ApiClientError | Error | null) => void,
  setLoading: (l: boolean) => void
): Promise<void> {
  try {
    const user = await apiClient.get<User>("/api/auth/me");
    setUser(user);
  } catch (err) {
    setError(err instanceof Error ? err : new Error(String(err)));
  } finally {
    setLoading(false);
  }
}
