import React from "react";
import { PixelLogo } from "@/components/ui/pixel-logo";

interface AuthLayoutProps {
  children: React.ReactNode;
}

export default function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-[#0d1117] px-4">
      <div className="w-full max-w-md">
        <AuthHeader />
        <div className="mt-8 rounded-xl border border-[#30363d] bg-[#161b22] p-8">
          {children}
        </div>
      </div>
    </div>
  );
}

function AuthHeader() {
  return (
    <div className="flex flex-col items-center gap-3">
      <PixelLogo size="lg" />
      <h1 className="font-mono text-2xl font-bold text-[#c9d1d9]">obeya</h1>
      <p className="font-mono text-sm text-[#8b949e]">
        Collaborative task and project management
      </p>
    </div>
  );
}
