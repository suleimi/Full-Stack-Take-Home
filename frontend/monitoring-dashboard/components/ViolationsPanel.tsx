"use client";

import type { EmissionViolation } from "@/lib/api";

interface Props {
  violations: EmissionViolation[];
  onAcknowledge: (violationId: string) => void;
}

export default function ViolationsPanel({ violations, onAcknowledge }: Props) {
  if (violations.length === 0) {
    return (
      <div className="bg-white rounded-lg border border-gray-200 p-8 text-center">
        <p className="text-sm text-gray-400">No open violations.</p>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-200 bg-gray-50">
              <th className="text-left px-4 py-3 font-medium text-gray-500">
                Period
              </th>
              <th className="text-left px-4 py-3 font-medium text-gray-500">
                Entity
              </th>
              <th className="text-right px-4 py-3 font-medium text-gray-500">
                Measured
              </th>
              <th className="text-right px-4 py-3 font-medium text-gray-500">
                Limit
              </th>
              <th className="text-right px-4 py-3 font-medium text-gray-500">
                Overage
              </th>
              <th className="text-left px-4 py-3 font-medium text-gray-500">
                Detected At
              </th>
              <th className="px-4 py-3"></th>
            </tr>
          </thead>
          <tbody>
            {violations.map((v) => (
              <tr
                key={v.id}
                className="border-b border-gray-100 hover:bg-gray-50"
              >
                <td className="px-4 py-3 text-gray-700">{v.period}</td>
                <td className="px-4 py-3 text-gray-700">
                  <span className="inline-block bg-gray-100 text-gray-600 text-xs px-1.5 py-0.5 rounded mr-1">
                    {v.entity_type}
                  </span>
                  <span className="text-xs text-gray-400 font-mono">
                    {v.entity_id.slice(0, 8)}...
                  </span>
                </td>
                <td className="px-4 py-3 text-right text-red-600 font-medium">
                  {v.measured_value.toFixed(2)}
                </td>
                <td className="px-4 py-3 text-right text-gray-700">
                  {v.limit_value.toFixed(2)}
                </td>
                <td className="px-4 py-3 text-right text-red-600 font-medium">
                  +{v.overage.toFixed(2)}
                </td>
                <td className="px-4 py-3 text-gray-500 text-xs">
                  {new Date(v.detected_at).toLocaleString()}
                </td>
                <td className="px-4 py-3 text-right">
                  {v.acknowledged_at ? (
                    <span className="text-xs text-gray-400">Acknowledged</span>
                  ) : (
                    <button
                      onClick={() => onAcknowledge(v.id)}
                      className="text-xs bg-red-50 text-red-600 px-2.5 py-1 rounded hover:bg-red-100 transition-colors font-medium"
                    >
                      Acknowledge
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
