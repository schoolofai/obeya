"use client";
import { useState } from "react";
import { apiClient } from "@/lib/api-client";
import type { ApiToken } from "@/lib/api-client";

interface TokenRowProps {
  token: ApiToken;
  onRevoke: (id: string) => void;
}

function formatLastUsed(ts: string | null): string {
  if (!ts) return "Never used";
  return new Date(ts).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function TokenRow({ token, onRevoke }: TokenRowProps) {
  return (
    <li className="flex items-center justify-between py-3 border-b border-gray-100 last:border-0">
      <div>
        <p className="text-sm font-medium text-gray-900">{token.name}</p>
        <p className="text-xs text-gray-500">Last used: {formatLastUsed(token.last_used_at)}</p>
      </div>
      <button
        onClick={() => onRevoke(token.$id)}
        className="text-sm text-red-600 hover:text-red-800 font-medium"
      >
        Revoke
      </button>
    </li>
  );
}

interface NewTokenBannerProps {
  rawToken: string;
}

function NewTokenBanner({ rawToken }: NewTokenBannerProps) {
  async function handleCopy() {
    await navigator.clipboard.writeText(rawToken);
  }

  return (
    <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-4">
      <p className="text-sm font-medium text-green-800 mb-2">
        Token created — copy it now, it won&apos;t be shown again:
      </p>
      <div className="flex items-center gap-2">
        <code className="flex-1 text-xs bg-white border border-green-200 rounded px-3 py-2 font-mono break-all">
          {rawToken}
        </code>
        <button
          onClick={handleCopy}
          className="text-sm text-green-700 hover:text-green-900 font-medium whitespace-nowrap"
        >
          Copy
        </button>
      </div>
    </div>
  );
}

interface CreateTokenFormProps {
  name: string;
  creating: boolean;
  onNameChange: (v: string) => void;
  onCreate: () => void;
}

function CreateTokenForm({ name, creating, onNameChange, onCreate }: CreateTokenFormProps) {
  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    onCreate();
  }

  return (
    <form onSubmit={handleSubmit} className="flex gap-2 mt-4">
      <input
        type="text"
        placeholder="Token name"
        value={name}
        onChange={(e) => onNameChange(e.target.value)}
        className="flex-1 border border-gray-300 rounded-lg px-4 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
      />
      <button
        type="submit"
        disabled={creating || !name.trim()}
        className="bg-indigo-600 text-white py-2 px-4 rounded-lg text-sm font-medium hover:bg-indigo-700 disabled:opacity-50"
      >
        {creating ? "Creating..." : "Create Token"}
      </button>
    </form>
  );
}

interface ApiTokenManagerProps {
  tokens: ApiToken[];
}

export function ApiTokenManager({ tokens: initialTokens }: ApiTokenManagerProps) {
  const [tokens, setTokens] = useState<ApiToken[]>(initialTokens);
  const [newTokenName, setNewTokenName] = useState("");
  const [creating, setCreating] = useState(false);
  const [newRawToken, setNewRawToken] = useState<string | null>(null);

  async function handleCreate() {
    if (!newTokenName.trim()) return;
    setCreating(true);
    const result = await apiClient.post<{ token: ApiToken; raw: string }>(
      "/api/auth/token",
      { name: newTokenName.trim() }
    );
    setTokens((prev) => [...prev, result.token]);
    setNewRawToken(result.raw);
    setNewTokenName("");
    setCreating(false);
  }

  async function handleRevoke(tokenId: string) {
    await apiClient.delete<void>(`/api/auth/token/${tokenId}`);
    setTokens((prev) => prev.filter((t) => t.$id !== tokenId));
  }

  return (
    <div>
      <h2 className="text-lg font-medium text-gray-900 mb-4">API Tokens</h2>
      {newRawToken && <NewTokenBanner rawToken={newRawToken} />}
      {tokens.length > 0 ? (
        <ul>
          {tokens.map((token) => (
            <TokenRow key={token.$id} token={token} onRevoke={handleRevoke} />
          ))}
        </ul>
      ) : (
        <p className="text-sm text-gray-500">No tokens yet.</p>
      )}
      <CreateTokenForm
        name={newTokenName}
        creating={creating}
        onNameChange={setNewTokenName}
        onCreate={handleCreate}
      />
    </div>
  );
}
