"use client";

import { useState, useCallback } from "react";
import type { FileChange } from "@/lib/types";

interface DiffsViewerProps {
  files: FileChange[];
  className?: string;
}

interface DiffLineProps {
  line: string;
  lineNum: number;
}

function DiffLine({ line, lineNum }: DiffLineProps) {
  const color = getDiffLineColor(line);
  const bg = getDiffLineBg(line);

  return (
    <div className={`flex font-mono text-xs ${bg}`}>
      <span className="w-10 shrink-0 text-right pr-2 select-none text-[var(--text-faint)]">
        {lineNum}
      </span>
      <pre className={`flex-1 overflow-x-auto whitespace-pre ${color}`}>
        {line}
      </pre>
    </div>
  );
}

function getDiffLineColor(line: string): string {
  if (line.startsWith("@@")) return "text-cyan-400/70";
  if (line.startsWith("+")) return "text-green-400";
  if (line.startsWith("-")) return "text-red-400";
  return "text-[var(--text-primary)]";
}

function getDiffLineBg(line: string): string {
  if (line.startsWith("+")) return "bg-green-900/10";
  if (line.startsWith("-")) return "bg-red-900/10";
  return "";
}

function FileDiff({ file }: { file: FileChange }) {
  const lines = file.diff?.split("\n") ?? [];

  return (
    <div data-testid={`file-diff-${file.path}`}>
      <div className="flex items-center justify-between px-3 py-2
        bg-[var(--bg-tertiary)] border-b border-[var(--border-default)]">
        <span className="text-xs font-mono font-medium text-[var(--text-primary)]">
          {file.path}
        </span>
        <div className="flex items-center gap-2 text-xs font-mono">
          <span className="text-green-400">+{file.added}</span>
          <span className="text-red-400">-{file.removed}</span>
        </div>
      </div>
      {lines.length > 0 && (
        <div className="py-1 overflow-x-auto">
          {lines.map((line, i) => (
            <DiffLine key={i} line={line} lineNum={i + 1} />
          ))}
        </div>
      )}
    </div>
  );
}

function FileSidebar({ files, selected, onSelect }: {
  files: FileChange[];
  selected: number;
  onSelect: (i: number) => void;
}) {
  return (
    <div data-testid="diff-file-sidebar" className="w-48 shrink-0 border-r border-[var(--border-default)]
      overflow-y-auto">
      <h4 className="text-xs font-semibold text-[var(--text-secondary)] uppercase
        tracking-wider px-3 py-2">
        Files
      </h4>
      <ul>
        {files.map((file, i) => (
          <li key={i}>
            <button
              onClick={() => onSelect(i)}
              className={`w-full text-left px-3 py-1.5 text-xs font-mono truncate
                transition-colors ${
                  i === selected
                    ? "bg-[var(--bg-tertiary)] text-[var(--text-primary)]"
                    : "text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-secondary)]"
                }`}
            >
              {file.path}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function DiffsViewer({ files, className = "" }: DiffsViewerProps) {
  const filesWithDiffs = files.filter((f) => f.diff);
  const [selectedIdx, setSelectedIdx] = useState(0);

  const handleSelect = useCallback((i: number) => {
    setSelectedIdx(i);
    document.getElementById(`diff-section-${i}`)?.scrollIntoView({ behavior: "smooth" });
  }, []);

  if (filesWithDiffs.length === 0) {
    return (
      <div className={`text-xs text-[var(--text-secondary)] p-4 ${className}`}>
        No diffs available
      </div>
    );
  }

  return (
    <div data-testid="diffs-viewer" className={`flex ${className}`}>
      {filesWithDiffs.length > 1 && (
        <FileSidebar
          files={filesWithDiffs}
          selected={selectedIdx}
          onSelect={handleSelect}
        />
      )}
      <div className="flex-1 overflow-y-auto">
        {filesWithDiffs.map((file, i) => (
          <div
            key={i}
            id={`diff-section-${i}`}
            className="border-b border-[var(--border-default)] last:border-b-0"
          >
            <FileDiff file={file} />
          </div>
        ))}
      </div>
    </div>
  );
}
