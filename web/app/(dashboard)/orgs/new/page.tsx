import { CreateOrgForm } from "@/components/org/create-org-form";

export default function CreateOrgPage() {
  return (
    <div className="max-w-md mx-auto p-6">
      <h1 className="text-xl font-semibold text-gray-900 mb-6">Create Organization</h1>
      <div className="bg-white border border-gray-200 rounded-lg p-6">
        <CreateOrgForm />
      </div>
    </div>
  );
}
