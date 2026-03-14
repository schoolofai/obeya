"use client";

import React from "react";

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  name: string;
  error?: string;
}

export function Input({
  label,
  name,
  error,
  type = "text",
  className = "",
  ...props
}: InputProps) {
  return (
    <div className="flex flex-col gap-1">
      <label
        htmlFor={name}
        className="font-mono text-sm font-medium text-[#c9d1d9]"
      >
        {label}
      </label>
      <input
        id={name}
        name={name}
        type={type}
        className={[
          "rounded-md border bg-[#0d1117] px-3 py-2 font-mono text-sm text-[#c9d1d9] placeholder:text-[#484f58]",
          "focus:outline-none focus:ring-2 focus:ring-[#58a6ff] focus:border-[#58a6ff]",
          error ? "border-[#f85149]" : "border-[#30363d]",
          className,
        ]
          .filter(Boolean)
          .join(" ")}
        {...props}
      />
      {error && (
        <p className="font-mono text-xs text-[#f85149]">{error}</p>
      )}
    </div>
  );
}
