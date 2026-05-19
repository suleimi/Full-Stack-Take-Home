"use client";

import { useState, useEffect } from "react";
import type {
  Site,
  Asset,
  AssetType,
  EmissionRollup,
  EmissionTotalToDate,
} from "@/lib/api";
import * as api from "@/lib/api";
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
  orgId: string;
  sites: Site[];
  selectedSite: Site | null;
  onSelectSite: (site: Site) => void;
  onCreateSite: (data: { name: string; location?: string }) => void;
  assets: Asset[];
  assetTypes: AssetType[];
  onCreateAsset: (data: { name: string; asset_type_id?: string }) => void;
  refreshKey: number;
}

function EmissionMiniChart({
  orgId,
  entityType,
  entityId,
  label,
  refreshKey,
}: {
  orgId: string;
  entityType: string;
  entityId: string;
  label: string;
  refreshKey: number;
}) {
  const [trend, setTrend] = useState<EmissionRollup[]>([]);
  const [total, setTotal] = useState<EmissionTotalToDate | null>(null);
  const [grain, setGrain] = useState("day");

  useEffect(() => {
    const to = new Date();
    const from = new Date();
    if (grain === "hour") from.setHours(from.getHours() - 48);
    else if (grain === "day") from.setDate(from.getDate() - 30);
    else from.setMonth(from.getMonth() - 12);

    api
      .getEmissionTrend(orgId, {
        entity_type: entityType,
        entity_id: entityId,
        grain,
        from: from.toISOString(),
        to: to.toISOString(),
      })
      .then((res) => setTrend(res.data ?? []))
      .catch(() => setTrend([]));

    api
      .getEmissionTotal(orgId, entityType, entityId)
      .then(setTotal)
      .catch(() => setTotal(null));
  }, [orgId, entityType, entityId, grain, refreshKey]);

  const chartData = trend.map((r) => ({
    time: new Date(r.bucket_start).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      ...(grain === "hour" ? { hour: "2-digit" } : {}),
    }),
    emissions: Number(r.total_emission.toFixed(2)),
  }));

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4">
      <div className="flex items-center justify-between mb-3">
        <div>
          <h4 className="text-sm font-semibold text-gray-900">
            {label}
            <span
              className={`ml-2 text-xs font-medium px-2 py-0.5 rounded ${
                entityType === "site"
                  ? "text-blue-600 bg-blue-50"
                  : "text-purple-600 bg-purple-50"
              }`}
            >
              {entityType === "site" ? "Site" : "Asset"}
            </span>
          </h4>
          <p className="text-xs text-gray-500">
            Total Till Date:{" "}
            <span className="font-medium text-gray-700">
              {total
                ? total.total_to_date.toLocaleString(undefined, {
                    maximumFractionDigits: 2,
                  })
                : "0"}{" "}
              kg CO2e
            </span>
            {total && (
              <span className="ml-2">
                ({total.reading_count} reading{total.reading_count !== 1 && "s"})
              </span>
            )}
          </p>
        </div>
        <div className="flex gap-1">
          {(["hour", "day", "month"] as const).map((g) => (
            <button
              key={g}
              onClick={() => setGrain(g)}
              className={`px-2 py-0.5 text-xs rounded font-medium transition-colors ${
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
        <p className="text-xs text-gray-400 text-center py-8">
          No emission data yet.
        </p>
      ) : (
        <ResponsiveContainer width="100%" height={180}>
          <LineChart data={chartData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" />
            <XAxis dataKey="time" tick={{ fontSize: 10 }} tickLine={false} />
            <YAxis tick={{ fontSize: 10 }} tickLine={false} />
            <Tooltip />
            <Line
              type="monotone"
              dataKey="emissions"
              stroke={entityType === "site" ? "#3b82f6" : "#8b5cf6"}
              strokeWidth={2}
              dot={{ r: 3 }}
              activeDot={{ r: 5 }}
              name="kg CO2e"
            />
          </LineChart>
        </ResponsiveContainer>
      )}
    </div>
  );
}

export default function SitesAssets({
  orgId,
  sites,
  selectedSite,
  onSelectSite,
  onCreateSite,
  assets,
  assetTypes,
  onCreateAsset,
  refreshKey,
}: Props) {
  const [siteName, setSiteName] = useState("");
  const [siteLocation, setSiteLocation] = useState("");
  const [assetName, setAssetName] = useState("");
  const [assetTypeId, setAssetTypeId] = useState("");
  const [selectedAsset, setSelectedAsset] = useState<Asset | null>(null);

  // Emission totals per site
  const [siteTotals, setSiteTotals] = useState<Record<string, EmissionTotalToDate>>({});
  // Emission totals per asset
  const [assetTotals, setAssetTotals] = useState<Record<string, EmissionTotalToDate>>({});

  // Fetch emission totals for all sites
  useEffect(() => {
    if (!orgId || sites.length === 0) {
      setSiteTotals({});
      return;
    }
    Promise.all(
      sites.map((s) =>
        api.getEmissionTotal(orgId, "site", s.id).then((t) => [s.id, t] as const)
      )
    )
      .then((results) => {
        const map: Record<string, EmissionTotalToDate> = {};
        for (const [id, t] of results) map[id] = t;
        setSiteTotals(map);
      })
      .catch(() => setSiteTotals({}));
  }, [orgId, sites, refreshKey]);

  // Fetch emission totals for assets of the selected site
  useEffect(() => {
    if (!orgId || assets.length === 0) {
      setAssetTotals({});
      return;
    }
    Promise.all(
      assets.map((a) =>
        api.getEmissionTotal(orgId, "asset", a.id).then((t) => [a.id, t] as const)
      )
    )
      .then((results) => {
        const map: Record<string, EmissionTotalToDate> = {};
        for (const [id, t] of results) map[id] = t;
        setAssetTotals(map);
      })
      .catch(() => setAssetTotals({}));
  }, [orgId, assets, refreshKey]);

  // Auto-select first asset when assets list changes (new site selected or data loaded)
  useEffect(() => {
    if (assets.length > 0) {
      setSelectedAsset(assets[0]);
    } else {
      setSelectedAsset(null);
    }
  }, [assets]);

  function handleCreateSite(e: React.FormEvent) {
    e.preventDefault();
    if (!siteName.trim()) return;
    onCreateSite({
      name: siteName.trim(),
      location: siteLocation.trim() || undefined,
    });
    setSiteName("");
    setSiteLocation("");
  }

  function handleCreateAsset(e: React.FormEvent) {
    e.preventDefault();
    if (!assetName.trim()) return;
    onCreateAsset({
      name: assetName.trim(),
      asset_type_id: assetTypeId || undefined,
    });
    setAssetName("");
    setAssetTypeId("");
  }

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Sites column */}
        <div className="bg-white rounded-lg border border-gray-200 p-5">
          <h3 className="text-base font-semibold text-gray-900 mb-3">Sites</h3>
          <div className="space-y-1 mb-4 max-h-64 overflow-y-auto">
            {sites.length === 0 && (
              <p className="text-sm text-gray-400">No sites yet.</p>
            )}
            {sites.map((s) => {
              const total = siteTotals[s.id];
              return (
                <button
                  key={s.id}
                  onClick={() => onSelectSite(s)}
                  className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${
                    selectedSite?.id === s.id
                      ? "bg-blue-50 text-blue-700 font-medium"
                      : "hover:bg-gray-50 text-gray-700"
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-medium">{s.name}</span>
                    {total && (
                      <span className="text-xs font-medium text-gray-500">
                        {total.total_to_date.toLocaleString(undefined, {
                          maximumFractionDigits: 1,
                        })}{" "}
                        kg
                      </span>
                    )}
                  </div>
                  {s.location && (
                    <div className="text-xs text-gray-400">{s.location}</div>
                  )}
                </button>
              );
            })}
          </div>

          <form
            onSubmit={handleCreateSite}
            className="border-t border-gray-100 pt-3 space-y-2"
          >
            <p className="text-xs font-medium text-gray-500 uppercase">
              New Site
            </p>
            <input
              placeholder="Site name *"
              value={siteName}
              onChange={(e) => setSiteName(e.target.value)}
              className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm"
              required
            />
            <input
              placeholder="Location"
              value={siteLocation}
              onChange={(e) => setSiteLocation(e.target.value)}
              className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm"
            />
            <button
              type="submit"
              className="w-full bg-blue-600 text-white text-sm py-1.5 rounded-md hover:bg-blue-700 transition-colors"
            >
              Create Site
            </button>
          </form>
        </div>

        {/* Assets column */}
        <div className="bg-white rounded-lg border border-gray-200 p-5">
          <h3 className="text-base font-semibold text-gray-900 mb-3">
            Assets
            {selectedSite && (
              <span className="text-sm font-normal text-gray-400 ml-2">
                at {selectedSite.name}
              </span>
            )}
          </h3>

          {!selectedSite ? (
            <p className="text-sm text-gray-400">
              Select a site to view its assets.
            </p>
          ) : (
            <>
              <div className="space-y-1 mb-4 max-h-64 overflow-y-auto">
                {assets.length === 0 && (
                  <p className="text-sm text-gray-400">No assets yet.</p>
                )}
                {assets.map((a) => {
                  const total = assetTotals[a.id];
                  return (
                    <button
                      key={a.id}
                      onClick={() => setSelectedAsset(selectedAsset?.id === a.id ? null : a)}
                      className={`w-full text-left px-3 py-2 rounded-md text-sm transition-colors ${
                        selectedAsset?.id === a.id
                          ? "bg-purple-50 text-purple-700 font-medium"
                          : "bg-gray-50 hover:bg-gray-100 text-gray-700"
                      }`}
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-medium">{a.name}</span>
                        {total && (
                          <span className="text-xs font-medium text-gray-500">
                            {total.total_to_date.toLocaleString(undefined, {
                              maximumFractionDigits: 1,
                            })}{" "}
                            kg
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-gray-400">
                        {a.asset_type_id
                          ? assetTypes.find((t) => t.id === a.asset_type_id)
                              ?.name ?? a.asset_type_id
                          : "No type"}
                      </div>
                    </button>
                  );
                })}
              </div>

              <form
                onSubmit={handleCreateAsset}
                className="border-t border-gray-100 pt-3 space-y-2"
              >
                <p className="text-xs font-medium text-gray-500 uppercase">
                  New Asset
                </p>
                <input
                  placeholder="Asset name *"
                  value={assetName}
                  onChange={(e) => setAssetName(e.target.value)}
                  className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm"
                  required
                />
                <select
                  value={assetTypeId}
                  onChange={(e) => setAssetTypeId(e.target.value)}
                  className="w-full border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
                >
                  <option value="">Select type (optional)</option>
                  {assetTypes.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.name}
                    </option>
                  ))}
                </select>
                <button
                  type="submit"
                  className="w-full bg-blue-600 text-white text-sm py-1.5 rounded-md hover:bg-blue-700 transition-colors"
                >
                  Create Asset
                </button>
              </form>
            </>
          )}
        </div>
      </div>

      {/* Emission charts for selected site / asset */}
      {selectedSite && (
        <EmissionMiniChart
          orgId={orgId}
          entityType="site"
          entityId={selectedSite.id}
          label={`Emissions — ${selectedSite.name}`}
          refreshKey={refreshKey}
        />
      )}

      {selectedAsset && (
        <EmissionMiniChart
          orgId={orgId}
          entityType="asset"
          entityId={selectedAsset.id}
          label={`Emissions — ${selectedAsset.name}`}
          refreshKey={refreshKey}
        />
      )}
    </div>
  );
}
