"use client";

import { useState } from "react";
import type { Site, Asset, RecorderDevice } from "@/lib/api";

interface Props {
  sites: Site[];
  assets: Asset[];
  devices: RecorderDevice[];
  onSelectSite: (siteId: string) => void;
  onSubmitReading: (data: {
    siteId: string;
    assetId: string;
    deviceId: string;
    reading: number;
    recordedAt: string;
    batchId: string;
  }) => void;
  onRefreshRollups: () => void;
  lastBatchId: string | null;
  error: string | null;
}

function generateUUID(): string {
  return crypto.randomUUID();
}

export default function EmissionSimulator({
  sites,
  assets,
  devices,
  onSelectSite,
  onSubmitReading,
  onRefreshRollups,
  lastBatchId,
  error,
}: Props) {
  const [open, setOpen] = useState(false);
  const [siteId, setSiteId] = useState("");
  const [assetId, setAssetId] = useState("");
  const [deviceId, setDeviceId] = useState("");
  const [reading, setReading] = useState(
    () => `${Math.floor(Math.random() * 100) + 1}`
  );
  const [recordedAt, setRecordedAt] = useState(() => {
    const now = new Date();
    now.setMinutes(now.getMinutes() - now.getTimezoneOffset());
    return now.toISOString().slice(0, 16);
  });
  const [batchId, setBatchId] = useState(generateUUID);

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!siteId || !assetId || !deviceId || !reading || !batchId.trim()) return;
    onSubmitReading({
      siteId,
      assetId,
      deviceId,
      reading: parseFloat(reading),
      recordedAt: new Date(recordedAt).toISOString(),
      batchId: batchId.trim(),
    });
    setReading(`${Math.floor(Math.random() * 100) + 1}`);
    setBatchId(generateUUID());
  }

  return (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <button
        onClick={() => setOpen(!open)}
        className="w-full flex items-center justify-between px-5 py-3 text-left hover:bg-gray-50 transition-colors"
      >
        <span className="text-sm font-semibold text-gray-700">
          Emission Simulator
        </span>
        <span className="text-gray-400 text-lg">{open ? "\u25B2" : "\u25BC"}</span>
      </button>

      {open && (
        <div className="border-t border-gray-200 px-5 py-4 space-y-4">
          <form
            onSubmit={handleSubmit}
            className="flex flex-wrap gap-3 items-end"
          >
            <div>
              <label className="block text-xs text-gray-500 mb-1">Site</label>
              <select
                value={siteId}
                onChange={(e) => {
                  setSiteId(e.target.value);
                  setAssetId("");
                  onSelectSite(e.target.value);
                }}
                className="border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
                required
              >
                <option value="">Select site...</option>
                {sites.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs text-gray-500 mb-1">Asset</label>
              <select
                value={assetId}
                onChange={(e) => setAssetId(e.target.value)}
                className="border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
                required
              >
                <option value="">Select asset...</option>
                {assets.map((a) => (
                  <option key={a.id} value={a.id}>
                    {a.name}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs text-gray-500 mb-1">
                Recorder Device
              </label>
              <select
                value={deviceId}
                onChange={(e) => setDeviceId(e.target.value)}
                className="border border-gray-300 rounded px-2 py-1.5 text-sm bg-white"
                required
              >
                <option value="">Select device...</option>
                {devices.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.type} ({d.id.slice(0, 8)}...)
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-xs text-gray-500 mb-1">
                Reading (kg CO2e)
              </label>
              <input
                type="number"
                step="any"
                value={reading}
                onChange={(e) => setReading(e.target.value)}
                className="border border-gray-300 rounded px-2 py-1.5 text-sm w-24"
                required
              />
            </div>

            <div>
              <label className="block text-xs text-gray-500 mb-1">
                Recorded At
              </label>
              <input
                type="datetime-local"
                value={recordedAt}
                onChange={(e) => setRecordedAt(e.target.value)}
                className="border border-gray-300 rounded px-2 py-1.5 text-sm"
                required
              />
            </div>

            <div className="w-full">
              <label className="block text-xs text-gray-500 mb-1">
                Batch ID (edit to reuse an old one for duplicate testing)
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={batchId}
                  onChange={(e) => setBatchId(e.target.value)}
                  className="border border-gray-300 rounded px-2 py-1.5 text-sm font-mono flex-1"
                  placeholder="UUID"
                  required
                />
                <button
                  type="button"
                  onClick={() => setBatchId(generateUUID())}
                  className="text-xs text-blue-600 hover:text-blue-800 whitespace-nowrap"
                >
                  New UUID
                </button>
              </div>
            </div>

            <div className="flex gap-2 items-end">
              <button
                type="submit"
                className="bg-blue-600 text-white text-sm px-4 py-1.5 rounded-md hover:bg-blue-700 transition-colors"
              >
                Simulate
              </button>

              <button
                type="button"
                onClick={onRefreshRollups}
                className="bg-gray-600 text-white text-sm px-4 py-1.5 rounded-md hover:bg-gray-700 transition-colors"
              >
                Refresh Rollups
              </button>
            </div>
          </form>

          {error && (
            <div className="text-sm text-red-700 bg-red-50 border border-red-200 rounded px-3 py-2">
              {error}
            </div>
          )}

          {lastBatchId && (
            <div className="text-xs text-gray-500 bg-gray-50 rounded px-3 py-2">
              Last batch ID:{" "}
              <button
                onClick={() => setBatchId(lastBatchId)}
                className="font-mono text-gray-700 hover:text-blue-600 underline decoration-dotted"
                title="Click to reuse this batch ID (will trigger duplicate rejection)"
              >
                {lastBatchId}
              </button>
              <span className="ml-2 text-gray-400">(click to reuse)</span>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
