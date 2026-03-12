"use client";

import React, { useEffect, useState } from "react";
import { BoardList } from "@/components/dashboard/board-list";
import { NewBoardDialog } from "@/components/dashboard/new-board-dialog";
import { Button } from "@/components/ui/button";
import { apiClient, ApiClientError } from "@/lib/api-client";
import type { Board, Org } from "@/lib/types";

export default function DashboardPage() {
  const [boards, setBoards] = useState<Board[]>([]);
  const [orgs, setOrgs] = useState<Org[]>([]);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);

  useEffect(() => {
    loadDashboardData(setBoards, setOrgs, setLoading);
  }, []);

  async function handleCreateBoard(name: string) {
    const board = await apiClient.post<Board>("/api/boards", { name });
    setBoards((prev) => [board, ...prev]);
    setDialogOpen(false);
  }

  if (loading) {
    return <LoadingState />;
  }

  return (
    <div className="space-y-6">
      <PageHeader onNewBoard={() => setDialogOpen(true)} />
      <BoardList boards={boards} orgs={orgs} />
      <NewBoardDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        onCreate={handleCreateBoard}
      />
    </div>
  );
}

async function loadDashboardData(
  setBoards: (boards: Board[]) => void,
  setOrgs: (orgs: Org[]) => void,
  setLoading: (loading: boolean) => void
): Promise<void> {
  try {
    const [boardsResult, orgsResult] = await Promise.all([
      apiClient.get<Board[]>("/api/boards"),
      apiClient.get<Org[]>("/api/orgs"),
    ]);
    setBoards(boardsResult);
    setOrgs(orgsResult);
  } catch (err) {
    if (err instanceof ApiClientError) {
      throw err;
    }
    throw new Error(
      `Failed to load dashboard data: ${err instanceof Error ? err.message : String(err)}`
    );
  } finally {
    setLoading(false);
  }
}

interface PageHeaderProps {
  onNewBoard: () => void;
}

function PageHeader({ onNewBoard }: PageHeaderProps) {
  return (
    <div className="flex items-center justify-between">
      <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
      <Button variant="primary" onClick={onNewBoard}>
        New Board
      </Button>
    </div>
  );
}

function LoadingState() {
  return (
    <div className="flex items-center justify-center py-12">
      <p className="text-sm text-gray-500">Loading boards…</p>
    </div>
  );
}
