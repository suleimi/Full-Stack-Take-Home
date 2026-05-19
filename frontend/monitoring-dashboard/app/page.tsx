"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import * as api from "@/lib/api";
import { DuplicateBatchError } from "@/lib/api";
import type {
  Organization,
  OrgSummary,
  Site,
  Asset,
  AssetType,
  RecorderDevice,
  EmissionRollup,
  EmissionTotalToDate,
  EmissionPolicy,
  EmissionViolation,
} from "@/lib/api";
import OrgSelector from "@/components/OrgSelector";
import DashboardSummary from "@/components/DashboardSummary";
import SitesAssets from "@/components/SitesAssets";
import ViolationsPanel from "@/components/ViolationsPanel";
import PoliciesPanel from "@/components/PoliciesPanel";
import EmissionSimulator from "@/components/EmissionSimulator";

const POLL_INTERVAL_MS = 30_000; // 30 seconds

type Tab = "dashboard" | "violations" | "policies";

export default function Home() {
  // ── Org state ──────────────────────────────────────────────────────────
  const [orgs, setOrgs] = useState<Organization[]>([]);
  const [selectedOrg, setSelectedOrg] = useState<Organization | null>(null);
  const [tab, setTab] = useState<Tab>("dashboard");

  // ── Dashboard state ────────────────────────────────────────────────────
  const [summary, setSummary] = useState<OrgSummary | null>(null);
  const [totalEmissions, setTotalEmissions] =
    useState<EmissionTotalToDate | null>(null);
  const [trendData, setTrendData] = useState<EmissionRollup[]>([]);
  const [grain, setGrain] = useState("day");

  // ── Sites / Assets state ───────────────────────────────────────────────
  const [sites, setSites] = useState<Site[]>([]);
  const [selectedSite, setSelectedSite] = useState<Site | null>(null);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [assetTypes, setAssetTypes] = useState<AssetType[]>([]);
  const [allAssets, setAllAssets] = useState<Asset[]>([]);

  // ── Violations state ───────────────────────────────────────────────────
  const [violations, setViolations] = useState<EmissionViolation[]>([]);

  // ── Policies state ─────────────────────────────────────────────────────
  const [policies, setPolicies] = useState<EmissionPolicy[]>([]);

  // ── Devices state ──────────────────────────────────────────────────────
  const [devices, setDevices] = useState<RecorderDevice[]>([]);

  // ── Simulator state ────────────────────────────────────────────────────
  const [simAssets, setSimAssets] = useState<Asset[]>([]);
  const [lastBatchId, setLastBatchId] = useState<string | null>(null);
  const [simulatorError, setSimulatorError] = useState<string | null>(null);
  const [refreshKey, setRefreshKey] = useState(0);

  // Track whether this is the initial site load (to auto-select first site)
  const initialSiteLoadRef = useRef(true);

  // ── Initial load ───────────────────────────────────────────────────────
  useEffect(() => {
    api.listOrgs().then((res) => {
      const data = res.data ?? [];
      setOrgs(data);
      if (data.length > 0) setSelectedOrg(data[0]);
    }).catch(console.error);

    api.listAssetTypes().then((res) => setAssetTypes(res.data ?? [])).catch(console.error);
  }, []);

  // ── Fetch data when org changes ────────────────────────────────────────
  const orgId = selectedOrg?.id;

  const fetchOrgData = useCallback(() => {
    if (!orgId) return;

    // Summary
    api.getOrgSummary(orgId).then(setSummary).catch(console.error);

    // Total emissions
    api
      .getEmissionTotal(orgId, "org", orgId)
      .then(setTotalEmissions)
      .catch(() => setTotalEmissions(null));

    // Sites -- only reset selection on initial load, not on poll refreshes
    api
      .listSites(orgId)
      .then((res) => {
        const data = res.data ?? [];
        setSites(data);
        if (initialSiteLoadRef.current) {
          initialSiteLoadRef.current = false;
          if (data.length > 0) {
            setSelectedSite(data[0]);
          } else {
            setSelectedSite(null);
            setAssets([]);
          }
        }
      })
      .catch(console.error);

    // Violations
    api
      .listViolations(orgId, true)
      .then((res) => setViolations(res.data ?? []))
      .catch(console.error);

    // Policies
    api
      .listPolicies(orgId, true)
      .then((res) => setPolicies(res.data ?? []))
      .catch(console.error);

    // Devices (org-scoped + field devices)
    Promise.all([api.listOrgDevices(orgId), api.listFieldDevices()])
      .then(([orgDevices, fieldDevices]) => {
        const map = new Map<string, RecorderDevice>();
        for (const d of [...(orgDevices.data ?? []), ...(fieldDevices.data ?? [])]) {
          map.set(d.id, d);
        }
        setDevices(Array.from(map.values()));
      })
      .catch(console.error);
  }, [orgId]);

  // Reset initial-load flag when org changes
  useEffect(() => {
    initialSiteLoadRef.current = true;
  }, [orgId]);

  useEffect(() => {
    fetchOrgData();
  }, [fetchOrgData]);

  // ── Fetch trend data ───────────────────────────────────────────────────
  const fetchTrend = useCallback(() => {
    if (!orgId) return;
    const to = new Date();
    const from = new Date();
    if (grain === "hour") from.setHours(from.getHours() - 48);
    else if (grain === "day") from.setDate(from.getDate() - 30);
    else from.setMonth(from.getMonth() - 12);

    api
      .getEmissionTrend(orgId, {
        entity_type: "org",
        entity_id: orgId,
        grain,
        from: from.toISOString(),
        to: to.toISOString(),
      })
      .then((res) => setTrendData(res.data ?? []))
      .catch(() => setTrendData([]));
  }, [orgId, grain]);

  useEffect(() => {
    fetchTrend();
  }, [fetchTrend, refreshKey]);

  // ── Polling: refresh all data every 30 seconds ─────────────────────────
  useEffect(() => {
    if (!orgId) return;
    const id = setInterval(() => {
      fetchOrgData();
      fetchTrend();
      setRefreshKey((k) => k + 1);
    }, POLL_INTERVAL_MS);
    return () => clearInterval(id);
  }, [orgId, fetchOrgData, fetchTrend]);

  // ── Fetch assets when site is selected ─────────────────────────────────
  useEffect(() => {
    if (!orgId || !selectedSite) {
      setAssets([]);
      return;
    }
    api
      .listAssets(orgId, selectedSite.id)
      .then((res) => setAssets(res.data ?? []))
      .catch(console.error);
  }, [orgId, selectedSite, refreshKey]);

  // ── Collect all assets for policy entity dropdown ──────────────────────
  useEffect(() => {
    if (!orgId || sites.length === 0) {
      setAllAssets([]);
      return;
    }
    Promise.all(sites.map((s) => api.listAssets(orgId, s.id)))
      .then((results) => setAllAssets(results.flatMap((r) => r.data ?? [])))
      .catch(console.error);
  }, [orgId, sites]);

  // ── Handlers ───────────────────────────────────────────────────────────

  async function handleCreateOrg(data: {
    name: string;
    email: string;
    description?: string;
    office_location?: string;
  }) {
    try {
      const org = await api.createOrg(data);
      setOrgs((prev) => [...prev, org]);
      setSelectedOrg(org);
    } catch (e) {
      console.error(e);
    }
  }

  async function handleCreateSite(data: { name: string; location?: string }) {
    if (!orgId) return;
    try {
      const site = await api.createSite(orgId, data);
      setSites((prev) => [...prev, site]);
      setSelectedSite(site);
    } catch (e) {
      console.error(e);
    }
  }

  async function handleCreateAsset(data: {
    name: string;
    asset_type_id?: string;
  }) {
    if (!orgId || !selectedSite) return;
    try {
      const asset = await api.createAsset(orgId, selectedSite.id, data);
      setAssets((prev) => [...prev, asset]);
      setAllAssets((prev) => [...prev, asset]);
    } catch (e) {
      console.error(e);
    }
  }

  async function handleAcknowledge(violationId: string) {
    try {
      await api.acknowledgeViolation(violationId);
      setViolations((prev) => prev.filter((v) => v.id !== violationId));
    } catch (e) {
      console.error(e);
    }
  }

  async function handleCreatePolicy(data: {
    entity_type: string;
    entity_id: string;
    emission_limit: number;
    period: string;
    unit?: string;
    effective_from?: string;
  }) {
    if (!orgId) return;
    try {
      const policy = await api.createPolicy(orgId, data);
      setPolicies((prev) => [...prev, policy]);
    } catch (e) {
      console.error(e);
    }
  }

  async function handleRetirePolicy(policyId: string) {
    try {
      await api.retirePolicy(policyId);
      setPolicies((prev) =>
        prev.map((p) =>
          p.id === policyId
            ? { ...p, effective_to: new Date().toISOString() }
            : p
        )
      );
    } catch (e) {
      console.error(e);
    }
  }

  async function handleSimulatorSiteChange(siteId: string) {
    if (!orgId || !siteId) {
      setSimAssets([]);
      return;
    }
    try {
      const res = await api.listAssets(orgId, siteId);
      setSimAssets(res.data ?? []);
    } catch (e) {
      console.error(e);
    }
  }

  async function handleSubmitReading(data: {
    siteId: string;
    assetId: string;
    deviceId: string;
    reading: number;
    recordedAt: string;
    batchId: string;
  }) {
    if (!orgId) return;
    setSimulatorError(null);
    try {
      const res = await api.ingestReadings(orgId, data.siteId, data.batchId, [
        {
          asset_id: data.assetId,
          recorder_device_id: data.deviceId,
          reading: data.reading,
          recorded_at: data.recordedAt,
        },
      ]);
      setLastBatchId(res.batch_id);
    } catch (e) {
      if (e instanceof DuplicateBatchError) {
        setSimulatorError(
          `Duplicate batch rejected: readings with batch ID ${e.batchId} were already ingested. Use a new batch ID.`
        );
      } else {
        setSimulatorError(
          e instanceof Error ? e.message : "Failed to ingest readings"
        );
      }
    }
  }

  async function handleRefreshRollups() {
    try {
      await api.refreshEmissions("48 hours");
      // Re-fetch dashboard data and trend
      fetchOrgData();
      setRefreshKey((k) => k + 1);
    } catch (e) {
      console.error(e);
    }
  }

  // ── Tab config ─────────────────────────────────────────────────────────
  const tabs: { id: Tab; label: string; badge?: number }[] = [
    { id: "dashboard", label: "Dashboard" },
    {
      id: "violations",
      label: "Violations",
      badge: violations.length > 0 ? violations.length : undefined,
    },
    { id: "policies", label: "Policies" },
  ];

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col">
      {/* Top bar */}
      <OrgSelector
        orgs={orgs}
        selectedOrg={selectedOrg}
        onSelect={setSelectedOrg}
        onCreate={handleCreateOrg}
      />

      {/* Tab navigation */}
      <div className="bg-white border-b border-gray-200 px-6">
        <nav className="flex gap-1">
          {tabs.map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`relative px-4 py-2.5 text-sm font-medium transition-colors border-b-2 ${
                tab === t.id
                  ? "border-blue-600 text-blue-600"
                  : "border-transparent text-gray-500 hover:text-gray-700"
              }`}
            >
              {t.label}
              {t.badge !== undefined && (
                <span className="ml-1.5 inline-flex items-center justify-center bg-red-500 text-white text-xs font-bold rounded-full w-5 h-5">
                  {t.badge}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>

      {/* Main content */}
      <main className="flex-1 p-6 max-w-7xl mx-auto w-full">
        {!selectedOrg ? (
          <div className="text-center py-20">
            <p className="text-gray-400 text-lg">
              Select or create an organization to get started.
            </p>
          </div>
        ) : (
          <>
            {tab === "dashboard" && (
              <div className="space-y-8">
                <DashboardSummary
                  summary={summary}
                  totalEmissions={totalEmissions}
                  violationCount={violations.length}
                  trendData={trendData}
                  grain={grain}
                  onGrainChange={setGrain}
                />

                <SitesAssets
                  orgId={selectedOrg.id}
                  sites={sites}
                  selectedSite={selectedSite}
                  onSelectSite={setSelectedSite}
                  onCreateSite={handleCreateSite}
                  assets={assets}
                  assetTypes={assetTypes}
                  onCreateAsset={handleCreateAsset}
                  refreshKey={refreshKey}
                />
              </div>
            )}

            {tab === "violations" && (
              <ViolationsPanel
                violations={violations}
                onAcknowledge={handleAcknowledge}
              />
            )}

            {tab === "policies" && (
              <PoliciesPanel
                policies={policies}
                orgId={selectedOrg.id}
                sites={sites}
                assets={allAssets}
                onCreatePolicy={handleCreatePolicy}
                onRetire={handleRetirePolicy}
              />
            )}
          </>
        )}
      </main>

      {/* Emission Simulator - always visible at bottom */}
      {selectedOrg && (
        <div className="px-6 pb-6 max-w-7xl mx-auto w-full">
          <EmissionSimulator
            sites={sites}
            assets={simAssets}
            devices={devices}
            onSelectSite={handleSimulatorSiteChange}
            onSubmitReading={handleSubmitReading}
            onRefreshRollups={handleRefreshRollups}
            lastBatchId={lastBatchId}
            error={simulatorError}
          />
        </div>
      )}
    </div>
  );
}
