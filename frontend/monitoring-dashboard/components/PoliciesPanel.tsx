"use client";

import { useState } from "react";
import type { EmissionPolicy, Site, Asset } from "@/lib/api";

interface Props {
  policies: EmissionPolicy[];
  orgId: string;
  sites: Site[];
  assets: Asset[];
  onCreatePolicy: (data: {
    entity_type: string;
    entity_id: string;
    emission_limit: number;
    period: string;
    unit?: string;
    effective_from?: string;
  }) => void;
  onRetire: (policyId: string) => void;
}

export default function PoliciesPanel({
  policies,
  orgId,
  sites,
  assets,
  onCreatePolicy,
  onRetire,
}: Props) {
  const [entityType, setEntityType] = useState("org");
  const [entityId, setEntityId] = useState("");
  const [emissionLimit, setEmissionLimit] = useState("");
  const [period, setPeriod] = useState("day");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const eId = entityType === "org" ? orgId : entityId;
    if (!eId || !emissionLimit) return;
    onCreatePolicy({
      entity_type: entityType,
      entity_id: eId,
      emission_limit: parseFloat(emissionLimit),
      period,
    });
    setEmissionLimit("");
    setEntityId("");
  }

  const entityOptions =
    entityType === "site"
      ? sites.map((s) => ({ id: s.id, label: s.name }))
      : entityType === "asset"
      ? assets.map((a) => ({ id: a.id, label: a.name }))
      : [];

  return (
    <div className="space-y-6">
      {/* Policy table */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        {policies.length === 0 ? (
          <p className="text-sm text-gray-400 text-center py-8">
            No active policies.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 bg-gray-50">
                  <th className="text-left px-4 py-3 font-medium text-gray-500">
                    Entity
                  </th>
                  <th className="text-right px-4 py-3 font-medium text-gray-500">
                    Limit
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-gray-500">
                    Period
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-gray-500">
                    Effective From
                  </th>
                  <th className="text-left px-4 py-3 font-medium text-gray-500">
                    Status
                  </th>
                  <th className="px-4 py-3"></th>
                </tr>
              </thead>
              <tbody>
                {policies.map((p) => (
                  <tr
                    key={p.id}
                    className="border-b border-gray-100 hover:bg-gray-50"
                  >
                    <td className="px-4 py-3 text-gray-700">
                      <span className="inline-block bg-gray-100 text-gray-600 text-xs px-1.5 py-0.5 rounded mr-1">
                        {p.entity_type}
                      </span>
                      <span className="text-xs text-gray-400 font-mono">
                        {p.entity_id.slice(0, 8)}...
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right text-gray-700 font-medium">
                      {p.emission_limit} {p.unit}
                    </td>
                    <td className="px-4 py-3 text-gray-700">{p.period}</td>
                    <td className="px-4 py-3 text-gray-500 text-xs">
                      {new Date(p.effective_from).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3">
                      {p.effective_to ? (
                        <span className="text-xs bg-gray-100 text-gray-500 px-2 py-0.5 rounded">
                          Retired
                        </span>
                      ) : (
                        <span className="text-xs bg-green-50 text-green-700 px-2 py-0.5 rounded">
                          Active
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-right">
                      {!p.effective_to && (
                        <button
                          onClick={() => onRetire(p.id)}
                          className="text-xs bg-red-50 text-red-600 px-2.5 py-1 rounded hover:bg-red-100 transition-colors font-medium"
                        >
                          Retire
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* New policy form */}
      <div className="bg-white rounded-lg border border-gray-200 p-5">
        <h3 className="text-sm font-semibold text-gray-700 mb-3">
          Create New Policy
        </h3>
        <form onSubmit={handleSubmit} className="flex flex-wrap gap-3 items-end">
          <div>
            <label className="block text-xs text-gray-500 mb-1">
              Entity Type
            </label>
            <select
              value={entityType}
              onChange={(e) => {
                setEntityType(e.target.value);
                setEntityId("");
              }}
              className="border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
            >
              <option value="org">Organization</option>
              <option value="site">Site</option>
              <option value="asset">Asset</option>
            </select>
          </div>

          {entityType !== "org" && (
            <div>
              <label className="block text-xs text-gray-500 mb-1">
                Entity
              </label>
              <select
                value={entityId}
                onChange={(e) => setEntityId(e.target.value)}
                className="border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
                required
              >
                <option value="">Select...</option>
                {entityOptions.map((o) => (
                  <option key={o.id} value={o.id}>
                    {o.label}
                  </option>
                ))}
              </select>
            </div>
          )}

          <div>
            <label className="block text-xs text-gray-500 mb-1">Period</label>
            <select
              value={period}
              onChange={(e) => setPeriod(e.target.value)}
              className="border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
            >
              <option value="hour">Hour</option>
              <option value="day">Day</option>
              <option value="month">Month</option>
            </select>
          </div>

          <div>
            <label className="block text-xs text-gray-500 mb-1">
              Emission Limit (kg CO2e)
            </label>
            <input
              type="number"
              step="any"
              value={emissionLimit}
              onChange={(e) => setEmissionLimit(e.target.value)}
              placeholder="e.g. 500"
              className="border border-gray-300 rounded px-2 py-1.5 text-sm w-32"
              required
            />
          </div>

          <button
            type="submit"
            className="bg-blue-600 text-white text-sm px-4 py-1.5 rounded-md hover:bg-blue-700 transition-colors"
          >
            Create Policy
          </button>
        </form>
      </div>
    </div>
  );
}
