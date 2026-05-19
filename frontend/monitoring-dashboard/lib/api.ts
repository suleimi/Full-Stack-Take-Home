const BASE = process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080/v1";

// ── Types ────────────────────────────────────────────────────────────────

export interface Organization {
  id: string;
  name: string;
  description: string | null;
  office_location: string | null;
  email: string;
  created_at: string;
  updated_at: string;
}

export interface OrgSummary {
  org_id: string;
  site_count: number;
  asset_count: number;
  last_update: string | null;
}

export interface Site {
  id: string;
  org_id: string;
  name: string;
  location: string | null;
  created_at: string;
  updated_at: string;
}

export interface AssetType {
  id: string;
  name: string;
  description: string | null;
  created_at: string;
  updated_at: string;
}

export interface Asset {
  id: string;
  org_id: string;
  site_id: string;
  asset_type_id: string | null;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface RecorderDevice {
  id: string;
  org_id: string | null;
  type: "field_device" | "sensor" | "satellite";
  device_metadata: Record<string, unknown> | null;
  created_at: string;
  updated_at: string;
}

export interface EmissionRollup {
  entity_type: "org" | "site" | "asset";
  entity_id: string;
  bucket_start: string;
  total_emission: number;
  reading_count: number;
  min_reading: number;
  max_reading: number;
  unit: string;
  computed_at: string;
}

export interface EmissionTotalToDate {
  entity_type: string;
  entity_id: string;
  total_to_date: number;
  reading_count: number;
  as_of: string;
}

export interface EmissionPolicy {
  id: string;
  org_id: string;
  entity_type: "org" | "site" | "asset";
  entity_id: string;
  emission_limit: number;
  unit: string;
  period: string;
  effective_from: string;
  effective_to: string | null;
  created_at: string;
}

export interface EmissionViolation {
  id: string;
  org_id: string;
  entity_type: "org" | "site" | "asset";
  entity_id: string;
  policy_id: string;
  period: string;
  limit_value: number;
  unit: string;
  period_start: string;
  measured_value: number;
  overage: number;
  detected_at: string;
  last_evaluated_at: string;
  acknowledged_at: string | null;
}

// ── Helpers ──────────────────────────────────────────────────────────────

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`);
  if (!res.ok) throw new Error(`GET ${path} failed: ${res.status}`);
  return res.json();
}

async function post<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(`POST ${path} failed: ${res.status}`);
  return res.json();
}

// ── Organizations ────────────────────────────────────────────────────────

export function listOrgs() {
  return get<{ data: Organization[]; total: number }>("/orgs");
}

export function createOrg(body: {
  name: string;
  email: string;
  description?: string;
  office_location?: string;
}) {
  return post<Organization>("/orgs", body);
}

export function getOrgSummary(orgId: string) {
  return get<OrgSummary>(`/orgs/${orgId}/summary`);
}

// ── Sites ────────────────────────────────────────────────────────────────

export function listSites(orgId: string) {
  return get<{ data: Site[]; total: number }>(`/orgs/${orgId}/sites`);
}

export function createSite(orgId: string, body: { name: string; location?: string }) {
  return post<Site>(`/orgs/${orgId}/sites`, body);
}

// ── Asset Types ──────────────────────────────────────────────────────────

export function listAssetTypes() {
  return get<{ data: AssetType[] }>("/asset-types");
}

// ── Assets ───────────────────────────────────────────────────────────────

export function listAssets(orgId: string, siteId: string) {
  return get<{ data: Asset[]; total: number }>(
    `/orgs/${orgId}/sites/${siteId}/assets`
  );
}

export function createAsset(
  orgId: string,
  siteId: string,
  body: { name: string; asset_type_id?: string }
) {
  return post<Asset>(`/orgs/${orgId}/sites/${siteId}/assets`, body);
}

// ── Devices ──────────────────────────────────────────────────────────────

export function listOrgDevices(orgId: string) {
  return get<{ data: RecorderDevice[] }>(`/orgs/${orgId}/devices`);
}

export function listFieldDevices() {
  return get<{ data: RecorderDevice[] }>("/devices/field");
}

// ── Readings ─────────────────────────────────────────────────────────────

export async function ingestReadings(
  orgId: string,
  siteId: string,
  batchId: string,
  readings: {
    asset_id: string;
    recorder_device_id: string;
    reading: number;
    unit?: string;
    recorded_at: string;
  }[]
) {
  const res = await fetch(`${BASE}/orgs/${orgId}/sites/${siteId}/readings`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ batch_id: batchId, readings }),
  });
  if (res.status === 409) {
    const body = await res.json();
    throw new DuplicateBatchError(body.batch_id ?? batchId);
  }
  if (!res.ok) throw new Error(`POST readings failed: ${res.status}`);
  return res.json() as Promise<{ batch_id: string; count: number }>;
}

export class DuplicateBatchError extends Error {
  batchId: string;
  constructor(batchId: string) {
    super(`Duplicate batch_id: readings already ingested (${batchId})`);
    this.name = "DuplicateBatchError";
    this.batchId = batchId;
  }
}

// ── Emissions ────────────────────────────────────────────────────────────

export function getEmissionTrend(
  orgId: string,
  params: {
    entity_type: string;
    entity_id: string;
    grain: string;
    from: string;
    to: string;
  }
) {
  const qs = new URLSearchParams(params).toString();
  return get<{ data: EmissionRollup[] }>(
    `/orgs/${orgId}/emissions/trend?${qs}`
  );
}

export function getEmissionTotal(
  orgId: string,
  entityType: string,
  entityId: string
) {
  const qs = new URLSearchParams({
    entity_type: entityType,
    entity_id: entityId,
  }).toString();
  return get<EmissionTotalToDate>(`/orgs/${orgId}/emissions/total?${qs}`);
}

export function refreshEmissions(window?: string) {
  return post<{ status: string }>("/emissions/refresh", window ? { window } : {});
}

// ── Policies ─────────────────────────────────────────────────────────────

export function listPolicies(orgId: string, activeOnly = true) {
  return get<{ data: EmissionPolicy[]; total: number }>(
    `/orgs/${orgId}/policies?active_only=${activeOnly}`
  );
}

export function createPolicy(
  orgId: string,
  body: {
    entity_type: string;
    entity_id: string;
    emission_limit: number;
    period: string;
    unit?: string;
    effective_from?: string;
  }
) {
  return post<EmissionPolicy>(`/orgs/${orgId}/policies`, body);
}

export function retirePolicy(policyId: string) {
  return post<{ status: string }>(`/policies/${policyId}/retire`);
}

// ── Violations ───────────────────────────────────────────────────────────

export function listViolations(orgId: string, openOnly = true) {
  return get<{ data: EmissionViolation[]; total: number }>(
    `/orgs/${orgId}/violations?open_only=${openOnly}`
  );
}

export function acknowledgeViolation(violationId: string) {
  return post<{ status: string }>(`/violations/${violationId}/acknowledge`);
}
