import type { BoardItem } from "./api-client";

export function breadcrumbPath(
  allItems: Record<string, BoardItem>,
  item: BoardItem
): string {
  if (!item.parent_id) return "";
  const titles: string[] = [];
  const visited = new Set([item.$id]);
  let cur = item;
  while (cur.parent_id) {
    if (visited.has(cur.parent_id)) break;
    visited.add(cur.parent_id);
    const parent = allItems[cur.parent_id];
    if (!parent) break;
    titles.unshift(parent.title);
    cur = parent;
  }
  return titles.join(" › ");
}

export function childCount(
  allItems: Record<string, BoardItem>,
  itemId: string
): number {
  let count = 0;
  for (const item of Object.values(allItems)) {
    if (isDescendantOf(allItems, item, itemId)) count++;
  }
  return count;
}

export function doneCount(
  allItems: Record<string, BoardItem>,
  itemId: string,
  doneStatus = "done"
): number {
  let count = 0;
  for (const item of Object.values(allItems)) {
    if (item.status === doneStatus && isDescendantOf(allItems, item, itemId))
      count++;
  }
  return count;
}

export function hasChildren(
  allItems: Record<string, BoardItem>,
  itemId: string
): boolean {
  return Object.values(allItems).some((item) => item.parent_id === itemId);
}

function isDescendantOf(
  allItems: Record<string, BoardItem>,
  item: BoardItem,
  ancestorId: string
): boolean {
  const visited = new Set([item.$id]);
  let cur = item;
  while (cur.parent_id) {
    if (cur.parent_id === ancestorId) return true;
    if (visited.has(cur.parent_id)) return false;
    visited.add(cur.parent_id);
    const parent = allItems[cur.parent_id];
    if (!parent) return false;
    cur = parent;
  }
  return false;
}

export function isHiddenByCollapse(
  allItems: Record<string, BoardItem>,
  item: BoardItem,
  collapsed: Record<string, boolean>
): boolean {
  const visited = new Set([item.$id]);
  let cur = item;
  while (cur.parent_id) {
    if (visited.has(cur.parent_id)) return false;
    visited.add(cur.parent_id);
    const parent = allItems[cur.parent_id];
    if (!parent) return false;
    if (collapsed[parent.$id] && parent.status === item.status) return true;
    cur = parent;
  }
  return false;
}

export function orderItemsHierarchically(
  allItems: Record<string, BoardItem>,
  columnItems: BoardItem[]
): BoardItem[] {
  const idSet = new Set(columnItems.map((i) => i.$id));
  const roots: BoardItem[] = [];
  const childrenOf: Record<string, BoardItem[]> = {};

  for (const item of columnItems) {
    if (!item.parent_id || !idSet.has(item.parent_id)) {
      roots.push(item);
    } else {
      (childrenOf[item.parent_id] ||= []).push(item);
    }
  }

  const byNum = (a: BoardItem, b: BoardItem) =>
    a.display_num - b.display_num;
  roots.sort(byNum);
  for (const children of Object.values(childrenOf)) children.sort(byNum);

  const ordered: BoardItem[] = [];
  const walk = (item: BoardItem) => {
    ordered.push(item);
    for (const child of childrenOf[item.$id] || []) walk(child);
  };
  for (const root of roots) walk(root);
  return ordered;
}
