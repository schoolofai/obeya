import type { ApiResult, FileChange, TestResult, ProofItem, ReviewContext, HumanReview } from "./types";
export type { FileChange, TestResult, ProofItem, ReviewContext, HumanReview };

// Appwrite document types for server-side data
export interface Board {
  $id: string;
  name: string;
  owner_id: string;
  org_id: string | null;
  display_counter: number;
  columns: string; // JSON string of BoardColumn[]
  created_at: string;
  updated_at: string;
}

export interface BoardColumn {
  name: string;
  limit: number;
}

export interface BoardItem {
  $id: string;
  board_id: string;
  display_num: number;
  type: "epic" | "story" | "task";
  title: string;
  description: string;
  status: string;
  priority: "low" | "medium" | "high" | "critical";
  parent_id: string | null;
  assignee_id: string | null;
  blocked_by: string[];
  tags: string[];
  project: string | null;
  sponsor?: string;
  confidence?: number | null;
  review_context?: ReviewContext | null;
  human_review?: HumanReview | null;
  created_at: string;
  updated_at: string;
}

export interface HistoryEntry {
  $id: string;
  item_id: string;
  board_id: string;
  user_id: string;
  action: string;
  detail: string;
  timestamp: string;
}

export interface Org {
  $id: string;
  name: string;
  slug: string;
  owner_id: string;
  plan: "free" | "pro" | "enterprise";
  created_at: string;
}

export interface OrgMember {
  $id: string;
  org_id: string;
  user_id: string;
  role: "owner" | "admin" | "member";
  invited_at: string;
  accepted_at: string | null;
}

export interface BoardMember {
  $id: string;
  board_id: string;
  user_id: string;
  role: "owner" | "editor" | "viewer";
  invited_at: string;
}

export interface ApiToken {
  $id: string;
  name: string;
  scopes: string[];
  last_used_at: string | null;
  expires_at: string | null;
}

export class ApiClientError extends Error {
  readonly code: string;
  readonly statusCode: number;

  constructor(code: string, message: string, statusCode: number) {
    super(message);
    this.name = "ApiClientError";
    this.code = code;
    this.statusCode = statusCode;
  }
}

async function request<T>(
  url: string,
  method: string,
  body?: unknown
): Promise<T> {
  const init: RequestInit = {
    method,
    headers: { "Content-Type": "application/json" },
  };

  if (body !== undefined) {
    init.body = JSON.stringify(body);
  }

  const response = await fetch(url, init);
  const json: ApiResult<T> = await response.json();

  if (!json.ok) {
    throw new ApiClientError(
      json.error.code,
      json.error.message,
      response.status
    );
  }

  return json.data;
}

export const apiClient = {
  get<T>(url: string): Promise<T> {
    return request<T>(url, "GET");
  },

  post<T>(url: string, body: unknown): Promise<T> {
    return request<T>(url, "POST", body);
  },

  patch<T>(url: string, body: unknown): Promise<T> {
    return request<T>(url, "PATCH", body);
  },

  delete<T>(url: string): Promise<T> {
    return request<T>(url, "DELETE");
  },
};
