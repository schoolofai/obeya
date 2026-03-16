"use client";

import { useState, useCallback } from "react";

interface ReproduceCommandsProps {
  commands: string[];
  className?: string;
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [text]);

  return (
    <button
      onClick={handleCopy}
      aria-label={`Copy command: ${text}`}
      className="shrink-0 px-2 py-1 text-xs rounded
        bg-[var(--bg-tertiary)] text-[var(--text-secondary)]
        hover:text-[var(--text-primary)] hover:bg-[var(--border-default)]
        transition-colors"
    >
      {copied ? "Copied!" : "Copy"}
    </button>
  );
}

export function ReproduceCommands({ commands, className = "" }: ReproduceCommandsProps) {
  if (commands.length === 0) return null;

  return (
    <div data-testid="reproduce-commands" className={className}>
      <h4 className="text-xs font-semibold text-[var(--text-secondary)] uppercase tracking-wider mb-2">
        Reproduce
      </h4>
      <ul className="space-y-1.5">
        {commands.map((cmd, i) => (
          <li key={i} className="flex items-center gap-2">
            <code className="flex-1 text-xs font-mono px-2 py-1.5 rounded
              bg-[var(--bg-primary)] text-[var(--text-primary)]
              border border-[var(--border-default)] truncate">
              $ {cmd}
            </code>
            <CopyButton text={cmd} />
          </li>
        ))}
      </ul>
    </div>
  );
}
