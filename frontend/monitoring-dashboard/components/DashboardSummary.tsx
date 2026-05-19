"use client";

import type { OrgSummary, EmissionRollup, EmissionTotalToDate } from "@/lib/api";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

interface Props {
  summary: OrgSummary | null;
  totalEmissions: EmissionTotalToDate | null;
  violationCount: number;
  trendData: EmissionRollup[];
  grain: string;
  onGrainChange: (grain: string) => void;
}

function SummaryCard({
  label,
  value,
  color,
}: {
  label: string;
  value: string | number;
  color?: string;
}) {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-5 flex flex-col">
      <span className="text-sm text-gray-500 font-medium">{label}</span>
      <span className={`text-2xl font-bold mt-1 ${color ?? "text-gray-900"}`}>
        {value}
      </span>
    </div>
  );
}

export default function DashboardSummary({
  summary,
  totalEmissions,
  violationCount,
  trendData,
  grain,
  onGrainChange,
}: Props) {
  const chartData = trendData.map((r) => ({
    time: new Date(r.bucket_start).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      ...(grain === "hour" ? { hour: "2-digit" } : {}),
    }),
    emissions: Number(r.total_emission.toFixed(2)),
  }));

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <SummaryCard
          label="Sites"
          value={summary?.site_count ?? 0}
        />
        <SummaryCard
          label="Assets"
          value={summary?.asset_count ?? 0}
        />
        <SummaryCard
          label="Open Violations"
          value={violationCount}
          color={violationCount > 0 ? "text-red-600" : "text-gray-900"}
        />
        <SummaryCard
          label="Total Emissions Till Date (kg CO2e)"
          value={
            totalEmissions
              ? totalEmissions.total_to_date.toLocaleString(undefined, {
                  maximumFractionDigits: 2,
                })
              : "0"
          }
        />
      </div>

      <div className="bg-white rounded-lg border border-gray-200 p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-base font-semibold text-gray-900">
            Emission Trend
            <span className="ml-2 text-xs font-medium text-blue-600 bg-blue-50 px-2 py-0.5 rounded">
              Organization
            </span>
          </h3>
          <div className="flex gap-1">
            {(["hour", "day", "month"] as const).map((g) => (
              <button
                key={g}
                onClick={() => onGrainChange(g)}
                className={`px-3 py-1 text-xs rounded-md font-medium transition-colors ${
                  grain === g
                    ? "bg-blue-600 text-white"
                    : "bg-gray-100 text-gray-600 hover:bg-gray-200"
                }`}
              >
                {g.charAt(0).toUpperCase() + g.slice(1)}
              </button>
            ))}
          </div>
        </div>

        {chartData.length === 0 ? (
          <p className="text-sm text-gray-400 text-center py-12">
            No emission data available for this period.
          </p>
        ) : (
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
              <XAxis
                dataKey="time"
                tick={{ fontSize: 11 }}
                tickLine={false}
              />
              <YAxis tick={{ fontSize: 11 }} tickLine={false} />
              <Tooltip />
              <Line
                type="monotone"
                dataKey="emissions"
                stroke="#3b82f6"
                strokeWidth={2}
                dot={{ r: 3 }}
                activeDot={{ r: 5 }}
                name="Emissions (kg CO2e)"
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}
