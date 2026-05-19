"use client";

import { useState } from "react";
import type { Organization } from "@/lib/api";

interface Props {
  orgs: Organization[];
  selectedOrg: Organization | null;
  onSelect: (org: Organization) => void;
  onCreate: (data: {
    name: string;
    email: string;
    description?: string;
    office_location?: string;
  }) => void;
}

export default function OrgSelector({
  orgs,
  selectedOrg,
  onSelect,
  onCreate,
}: Props) {
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [description, setDescription] = useState("");
  const [officeLocation, setOfficeLocation] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim() || !email.trim()) return;
    onCreate({
      name: name.trim(),
      email: email.trim(),
      description: description.trim() || undefined,
      office_location: officeLocation.trim() || undefined,
    });
    setName("");
    setEmail("");
    setDescription("");
    setOfficeLocation("");
    setShowForm(false);
  }

  return (
    <div className="flex items-center gap-4 bg-white border-b border-gray-200 px-6 py-3">
      <span className="text-sm font-medium text-gray-500">Organization:</span>
      <select
        className="border border-gray-300 rounded-md px-3 py-1.5 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
        value={selectedOrg?.id ?? ""}
        onChange={(e) => {
          const org = orgs.find((o) => o.id === e.target.value);
          if (org) onSelect(org);
        }}
      >
        <option value="" disabled>
          Select org...
        </option>
        {orgs.map((o) => (
          <option key={o.id} value={o.id}>
            {o.name}
          </option>
        ))}
      </select>

      {selectedOrg && (
        <span className="text-lg font-semibold text-gray-900">
          {selectedOrg.name}
        </span>
      )}

      <div className="ml-auto">
        <button
          onClick={() => setShowForm(!showForm)}
          className="text-sm bg-blue-600 text-white px-3 py-1.5 rounded-md hover:bg-blue-700 transition-colors"
        >
          {showForm ? "Cancel" : "+ New Org"}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleSubmit} className="flex items-center gap-2">
          <input
            placeholder="Name *"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="border border-gray-300 rounded px-2 py-1 text-sm w-32"
            required
          />
          <input
            placeholder="Email *"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="border border-gray-300 rounded px-2 py-1 text-sm w-40"
            required
          />
          <input
            placeholder="Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="border border-gray-300 rounded px-2 py-1 text-sm w-32"
          />
          <input
            placeholder="Location"
            value={officeLocation}
            onChange={(e) => setOfficeLocation(e.target.value)}
            className="border border-gray-300 rounded px-2 py-1 text-sm w-28"
          />
          <button
            type="submit"
            className="bg-green-600 text-white px-3 py-1 rounded text-sm hover:bg-green-700 transition-colors"
          >
            Create
          </button>
        </form>
      )}
    </div>
  );
}
