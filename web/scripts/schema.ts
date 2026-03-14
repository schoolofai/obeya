/**
 * Database schema definitions for all Obeya Cloud collections.
 * Used by setup-db.ts to create collections and attributes.
 */

export type AttrType = "string" | "integer" | "enum" | "datetime";

export interface StringAttr {
  type: "string";
  key: string;
  size: number;
  required: boolean;
}

export interface IntegerAttr {
  type: "integer";
  key: string;
  required: boolean;
  min?: number;
  max?: number;
}

export interface EnumAttr {
  type: "enum";
  key: string;
  elements: string[];
  required: boolean;
}

export interface DatetimeAttr {
  type: "datetime";
  key: string;
  required: boolean;
}

export type Attribute = StringAttr | IntegerAttr | EnumAttr | DatetimeAttr;

export interface CollectionSchema {
  id: string;
  name: string;
  attributes: Attribute[];
}

function str(key: string, size: number, required = true): StringAttr {
  return { type: "string", key, size, required };
}

function int(key: string, required = true, min?: number, max?: number): IntegerAttr {
  return { type: "integer", key, required, min, max };
}

function enm(key: string, elements: string[], required = true): EnumAttr {
  return { type: "enum", key, elements, required };
}

function dt(key: string, required = true): DatetimeAttr {
  return { type: "datetime", key, required };
}

export const SCHEMAS: CollectionSchema[] = [
  {
    id: "boards",
    name: "Boards",
    attributes: [
      str("name", 128),
      str("owner_id", 36),
      str("org_id", 36, false),
      int("display_counter", true, 0),
      str("columns", 10000),
      str("display_map", 2000),
      str("users", 2000),
      str("projects", 1000),
      str("agent_role", 64),
      int("version", true, 0),
      dt("created_at"),
      dt("updated_at"),
    ],
  },
  {
    id: "items",
    name: "Items",
    attributes: [
      str("board_id", 36),
      int("display_num", true, 0),
      enm("type", ["epic", "story", "task"]),
      str("title", 256),
      str("description", 10000),
      str("status", 64),
      enm("priority", ["low", "medium", "high", "critical"]),
      str("parent_id", 36, false),
      str("assignee_id", 36, false),
      str("blocked_by", 2000),
      str("tags", 2000),
      str("project", 128, false),
      dt("created_at"),
      dt("updated_at"),
    ],
  },
  {
    id: "item_history",
    name: "Item History",
    attributes: [
      str("item_id", 36),
      str("board_id", 36),
      str("user_id", 36),
      str("session_id", 64),
      enm("action", ["created", "moved", "edited", "assigned", "blocked", "unblocked"]),
      str("detail", 10000),
      dt("timestamp"),
    ],
  },
  {
    id: "plans",
    name: "Plans",
    attributes: [
      str("board_id", 36),
      int("display_num", true, 0),
      str("title", 256),
      str("source_path", 512),
      str("content", 10000),
      str("linked_items", 2000),
      dt("created_at"),
    ],
  },
  {
    id: "orgs",
    name: "Organizations",
    attributes: [
      str("name", 128),
      str("slug", 128),
      str("owner_id", 36),
      enm("plan", ["free", "pro", "enterprise"]),
      dt("created_at"),
    ],
  },
  {
    id: "org_members",
    name: "Organization Members",
    attributes: [
      str("org_id", 36),
      str("user_id", 36),
      enm("role", ["owner", "admin", "member"]),
      dt("invited_at"),
      dt("accepted_at", false),
    ],
  },
  {
    id: "board_members",
    name: "Board Members",
    attributes: [
      str("board_id", 36),
      str("user_id", 36),
      enm("role", ["owner", "editor", "viewer"]),
      dt("invited_at"),
    ],
  },
  {
    id: "api_tokens",
    name: "API Tokens",
    attributes: [
      str("user_id", 36),
      str("name", 128),
      str("token_hash", 256),
      str("scopes", 2000),
      dt("last_used_at", false),
      dt("expires_at", false),
    ],
  },
];
