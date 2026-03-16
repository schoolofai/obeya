export interface Board {
  id: string;
  name: string;
  owner_id: string;
  org_id: string | null;
  display_counter: number;
  columns: Column[];
  item_count: number;
  created_at: string;
  updated_at: string;
}

export interface Column {
  name: string;
  limit: number;
}

export interface FileChange {
  path: string;
  added: number;
  removed: number;
  diff?: string;
}

export interface TestResult {
  name: string;
  passed: boolean;
}

export interface ProofItem {
  check: string;
  status: "pass" | "fail" | "warn";
  detail?: string;
}

export interface ReviewContext {
  purpose: string;
  files_changed?: FileChange[];
  tests_written?: TestResult[];
  proof?: ProofItem[];
  reasoning?: string;
  reproduce?: string[];
}

export interface HumanReview {
  status: "pending" | "reviewed" | "hidden";
  reviewed_by?: string;
  reviewed_at?: string;
}

export interface Item {
  id: string;
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

export interface User {
  id: string;
  email: string;
  name: string;
}

export interface Org {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  plan: "free" | "pro" | "enterprise";
  created_at: string;
}

export interface ApiResponse<T> {
  ok: true;
  data: T;
  meta?: { total?: number; page?: number };
}

export interface ApiError {
  ok: false;
  error: { code: string; message: string };
}

export type ApiResult<T> = ApiResponse<T> | ApiError;
