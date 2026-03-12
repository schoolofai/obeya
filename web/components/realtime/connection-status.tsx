"use client";

import type { ConnectionStatus } from "@/hooks/use-board-subscription";

interface ConnectionStatusIndicatorProps {
  status: ConnectionStatus;
  compact?: boolean;
}

const STATUS_CONFIG: Record<
  ConnectionStatus,
  { label: string; dotClass: string; textClass: string }
> = {
  connected: {
    label: "Connected",
    dotClass: "bg-green-500",
    textClass: "text-green-700 dark:text-green-400",
  },
  connecting: {
    label: "Connecting...",
    dotClass: "bg-yellow-500 animate-pulse",
    textClass: "text-yellow-700 dark:text-yellow-400",
  },
  disconnected: {
    label: "Offline",
    dotClass: "bg-gray-400",
    textClass: "text-gray-500 dark:text-gray-400",
  },
  error: {
    label: "Error",
    dotClass: "bg-red-500",
    textClass: "text-red-700 dark:text-red-400",
  },
};

export function ConnectionStatusIndicator({
  status,
  compact = false,
}: ConnectionStatusIndicatorProps) {
  const config = STATUS_CONFIG[status];

  return (
    <div
      className="flex items-center gap-2"
      data-status={status}
      role="status"
      aria-label={`Realtime connection: ${config.label}`}
    >
      <span
        className={`inline-block h-2 w-2 rounded-full ${config.dotClass}`}
        data-status={status}
      />
      {!compact && (
        <span className={`text-xs font-medium ${config.textClass}`}>
          {config.label}
        </span>
      )}
    </div>
  );
}
