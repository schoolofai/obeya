import React from "react";

interface AuthLayoutProps {
  children: React.ReactNode;
}

export default function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-md">
        <AuthHeader />
        <div className="mt-8 rounded-xl bg-white p-8 shadow-sm ring-1 ring-gray-200">
          {children}
        </div>
      </div>
    </div>
  );
}

function AuthHeader() {
  return (
    <div className="text-center">
      <h1 className="text-3xl font-bold text-blue-600">Obeya</h1>
      <p className="mt-2 text-sm text-gray-600">
        Collaborative task and project management
      </p>
    </div>
  );
}
