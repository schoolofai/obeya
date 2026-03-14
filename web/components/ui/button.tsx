"use client";

import React from "react";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type ButtonSize = "sm" | "md" | "lg";

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  fullWidth?: boolean;
}

const variantClasses: Record<ButtonVariant, string> = {
  primary: "bg-[#7aa2f7] text-[#0d1117] hover:bg-[#6b93e8] font-semibold",
  secondary:
    "bg-[#21262d] text-[#c9d1d9] border border-[#30363d] hover:bg-[#30363d]",
  ghost: "bg-transparent text-[#8b949e] hover:bg-[#21262d] hover:text-[#c9d1d9]",
  danger: "bg-[#f85149] text-[#0d1117] hover:bg-[#e5443c] font-semibold",
};

const sizeClasses: Record<ButtonSize, string> = {
  sm: "px-3 py-1.5 text-sm",
  md: "px-4 py-2 text-sm",
  lg: "px-5 py-2.5 text-base",
};

export function Button({
  variant = "primary",
  size = "md",
  fullWidth = false,
  disabled = false,
  className = "",
  children,
  ...props
}: ButtonProps) {
  const classes = [
    "inline-flex items-center justify-center rounded-md font-mono font-medium transition-colors",
    "focus:outline-none focus:ring-2 focus:ring-[#58a6ff] focus:ring-offset-1 focus:ring-offset-[#0d1117]",
    "disabled:opacity-50 disabled:cursor-not-allowed",
    variantClasses[variant],
    sizeClasses[size],
    fullWidth ? "w-full" : "",
    className,
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <button className={classes} disabled={disabled} {...props}>
      {children}
    </button>
  );
}
