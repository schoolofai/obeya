"use client";

import React from "react";
import { BoardCard } from "./board-card";
import type { Board, Org } from "@/lib/types";

interface BoardListProps {
  boards: Board[];
  orgs: Org[];
}

export function BoardList({ boards, orgs }: BoardListProps) {
  if (boards.length === 0) {
    return <EmptyState />;
  }

  const personalBoards = boards.filter((b) => b.org_id === null);
  const orgBoardsMap = buildOrgBoardsMap(boards, orgs);

  return (
    <div className="space-y-8">
      {personalBoards.length > 0 && (
        <BoardSection title="Personal" boards={personalBoards} />
      )}
      {orgBoardsMap.map(({ org, boards: orgBoards }) => (
        <BoardSection key={org.id} title={org.name} boards={orgBoards} />
      ))}
    </div>
  );
}

interface OrgBoardGroup {
  org: Org;
  boards: Board[];
}

function buildOrgBoardsMap(boards: Board[], orgs: Org[]): OrgBoardGroup[] {
  return orgs
    .map((org) => ({
      org,
      boards: boards.filter((b) => b.org_id === org.id),
    }))
    .filter(({ boards: orgBoards }) => orgBoards.length > 0);
}

interface BoardSectionProps {
  title: string;
  boards: Board[];
}

function BoardSection({ title, boards }: BoardSectionProps) {
  return (
    <section>
      <h2 className="mb-4 text-lg font-semibold text-gray-900">{title}</h2>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {boards.map((board) => (
          <BoardCard key={board.id} board={board} />
        ))}
      </div>
    </section>
  );
}

function EmptyState() {
  return (
    <div className="rounded-lg border border-dashed border-gray-300 p-12 text-center">
      <p className="text-sm text-gray-500">No boards yet.</p>
    </div>
  );
}
