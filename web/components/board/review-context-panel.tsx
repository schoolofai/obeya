"use client";

import { useState } from "react";
import type { ReviewContext } from "@/lib/types";
import { ReproduceCommands } from "./reproduce-commands";

interface ReviewContextPanelProps {
  context: ReviewContext;
  className?: string;
}

function ProofList({ proof }: { proof: ReviewContext["proof"] }) {
  if (!proof || proof.length === 0) return null;

  const statusIcon: Record<string, string> = {
    pass: "\u2713",
    fail: "\u2717",
    warn: "\u26A0",
  };

  const statusColor: Record<string, string> = {
    pass: "text-green-400",
    fail: "text-red-400",
    warn: "text-yellow-400",
  };

  return (
    <div data-testid="proof-list">
      <h4 className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider mb-2">
        Proof
      </h4>
      <ul className="space-y-1">
        {proof.map((item, i) => (
          <li key={i} className="flex items-start gap-2 text-xs">
            <span className={`font-mono ${statusColor[item.status]}`}>
              {statusIcon[item.status] ?? "?"}
            </span>
            <span className="text-[var(--text-primary)]">
              {item.check}
              {item.detail && (
                <span className="text-[var(--text-secondary)] ml-1">
                  — {item.detail}
                </span>
              )}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function FilesList({ files }: { files: ReviewContext["files_changed"] }) {
  if (!files || files.length === 0) return null;

  return (
    <div data-testid="files-list">
      <h4 className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider mb-2">
        Files Changed
      </h4>
      <ul className="space-y-1">
        {files.map((file, i) => (
          <li key={i} className="flex items-center gap-2 text-xs font-mono">
            <span className="text-[var(--text-primary)] truncate flex-1">
              {file.path}
            </span>
            <span className="text-green-400">+{file.added}</span>
            <span className="text-red-400">-{file.removed}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function TestsList({ tests }: { tests: ReviewContext["tests_written"] }) {
  if (!tests || tests.length === 0) return null;

  const passed = tests.filter((t) => t.passed).length;
  const total = tests.length;

  return (
    <div data-testid="tests-list">
      <h4 className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider mb-2">
        Tests
      </h4>
      <p className="text-xs text-[var(--text-primary)]">
        {passed}/{total} passing
      </p>
    </div>
  );
}

export function ReviewContextPanel({ context, className = "" }: ReviewContextPanelProps) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div data-testid="review-context-panel" className={className}>
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1.5 text-xs font-medium
          text-[var(--text-secondary)] hover:text-[var(--text-primary)]
          transition-colors w-full text-left"
      >
        <span className="font-mono">{expanded ? "\u25BC" : "\u25B6"}</span>
        review context
      </button>
      {expanded && (
        <div className="mt-2 pl-4 space-y-3 border-l border-[var(--border-default)]">
          {context.purpose && (
            <div>
              <h4 className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider mb-1">
                Purpose
              </h4>
              <p className="text-xs text-[var(--text-primary)] leading-relaxed">
                {context.purpose}
              </p>
            </div>
          )}
          <FilesList files={context.files_changed} />
          <TestsList tests={context.tests_written} />
          <ReproduceCommands commands={context.reproduce ?? []} />
          <ProofList proof={context.proof} />
          {context.reasoning && (
            <div>
              <h4 className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider mb-1">
                Reasoning
              </h4>
              <p className="text-xs text-[var(--text-primary)] leading-relaxed">
                {context.reasoning}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
