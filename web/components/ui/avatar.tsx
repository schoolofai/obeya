"use client";

import React from "react";

type AvatarSize = "sm" | "md" | "lg";

interface AvatarProps {
  name: string;
  src?: string;
  size?: AvatarSize;
  className?: string;
}

const sizeClasses: Record<AvatarSize, string> = {
  sm: "h-8 w-8 text-xs",
  md: "h-10 w-10 text-sm",
  lg: "h-12 w-12 text-base",
};

export function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 1) return parts[0][0].toUpperCase();
  const first = parts[0][0].toUpperCase();
  const last = parts[parts.length - 1][0].toUpperCase();
  return `${first}${last}`;
}

export function Avatar({ name, src, size = "md", className = "" }: AvatarProps) {
  const classes = [
    "inline-flex items-center justify-center rounded-full overflow-hidden",
    "bg-blue-100 font-medium text-blue-700",
    sizeClasses[size],
    className,
  ]
    .filter(Boolean)
    .join(" ");

  if (src) {
    return (
      <div className={classes}>
        <img src={src} alt={name} className="h-full w-full object-cover" />
      </div>
    );
  }

  return <div className={classes}>{getInitials(name)}</div>;
}
