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
      <label htmlFor={name} className="text-sm font-medium text-gray-700">
        {label}
      </label>
      <input
        id={name}
        name={name}
        type={type}
        className={[
          "rounded-md border px-3 py-2 text-sm text-gray-900 bg-white placeholder:text-gray-400",
          "focus:outline-none focus:ring-2 focus:ring-blue-500",
          error ? "border-red-500" : "border-gray-300",
          className,
        ]
          .filter(Boolean)
          .join(" ")}
        {...props}
      />
      {error && <p className="text-xs text-red-600">{error}</p>}
    </div>
  );
}
