"use client";

interface AgentBadgeProps {
  className?: string;
}

export function AgentBadge({ className = "" }: AgentBadgeProps) {
  return (
    <span
      data-testid="agent-badge"
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5
        text-xs font-semibold bg-purple-900/40 text-purple-300
        border border-purple-500/30 ${className}`}
    >
      <span aria-hidden="true">&#x1F916;</span>
      Agent
    </span>
  );
}
