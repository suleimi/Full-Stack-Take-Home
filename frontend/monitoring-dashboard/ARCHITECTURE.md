# Frontend Architecture

A single-page dashboard for the EIAE emissions monitoring platform. Designed to be simple enough for a beginner to understand, extend, and maintain.

## Technology Choices

| Technology | Version | Why |
| --- | --- | --- |
| **Next.js** | 16 | React framework with zero-config setup, file-based routing, and built-in TypeScript support. Only the app router is used -- no server components, no SSR; the entire dashboard is a client-side SPA. |
| **React** | 19 | Component model with hooks (`useState`, `useEffect`) for state and side effects. No external state library is needed at this scale. |
| **TypeScript** | 5 | Catches type mismatches between the frontend and the API contract at compile time. Every API response is typed. |
| **Tailwind CSS** | 4 | Utility-first CSS that keeps styling co-located with markup. No CSS files to manage per component. |
| **Recharts** | 2 | Declarative React charting library. Wraps D3 internals but exposes a simple component API (`<BarChart>`, `<LineChart>`, `<XAxis>`, etc.). Chosen over Chart.js for its React-native composability. |

## Why No State Management Library

The dashboard has a single page with ~8 pieces of top-level state (selected org, sites, assets, violations, etc.). This fits comfortably in `useState` hooks inside `page.tsx`. Introducing Redux, Zustand, or React Context would add indirection without reducing complexity at this scale.

**Rule of thumb:** if you can describe all state transitions in one file without scrolling past 300 lines of state logic, you do not need a state library.

## Component Structure

```
app/
├── layout.tsx          Root HTML shell (fonts, metadata)
├── page.tsx            Main page -- owns ALL state, fetches data, passes props down
└── globals.css         Tailwind import + base theme variables

components/
├── OrgSelector.tsx     Org dropdown + create-org form (top bar)
├── DashboardSummary.tsx  Summary cards + emission trend chart (recharts)
├── SitesAssets.tsx      Two-column panel: sites list + assets list with create forms
├── ViolationsPanel.tsx  Violations table with acknowledge action
├── PoliciesPanel.tsx    Policies table with create form + retire action
└── EmissionSimulator.tsx  Collapsible panel to simulate readings + trigger rollup refresh

lib/
└── api.ts              Typed fetch wrapper -- every API call is a named function
```

## Data Flow

```
page.tsx (state owner)
  │
  ├── useEffect: fetch orgs on mount
  ├── useEffect: when selectedOrg changes → fetch sites, summary, violations, policies, devices
  ├── useEffect: when selectedSite changes → fetch assets
  │
  ├── OrgSelector         props: orgs, selectedOrg, onSelect, onCreate
  ├── DashboardSummary    props: orgId, summary, emissionTotal
  ├── SitesAssets          props: sites, assets, assetTypes, selectedSite, onSelectSite, onCreate*
  ├── ViolationsPanel     props: violations, onAcknowledge
  ├── PoliciesPanel       props: policies, sites, assets, orgId, onCreate, onRetire
  └── EmissionSimulator   props: orgId, sites, assets, devices, onSimulate, onRefresh
```

**All data fetching happens in `page.tsx`.** Components are pure presentation + forms. They call callbacks (e.g. `onCreate`) which trigger a re-fetch in the parent. This makes every component testable in isolation by passing mock props.

## API Client Pattern

`lib/api.ts` exports one async function per API call. Each function:

1. Builds the URL with path and query parameters
2. Calls `fetch()` with the appropriate method and headers
3. Parses the JSON response
4. Returns a typed result

```typescript
// Example
export async function listSites(orgId: string): Promise<{ data: Site[]; total: number }> {
  const res = await fetch(`${BASE}/orgs/${orgId}/sites`);
  return res.json();
}
```

No Axios, no SWR, no React Query. Plain `fetch` is sufficient for a dashboard that re-fetches on user actions rather than polling. If polling or caching becomes necessary, the fetch calls are trivially wrappable.

## Styling Conventions

- **Tailwind only** -- no CSS modules, no inline `style` props, no styled-components.
- Cards use `rounded-lg border bg-white p-4 shadow-sm`.
- Action buttons use `bg-blue-600 text-white rounded px-3 py-1` (blue for primary actions).
- Destructive / alert elements use red (`text-red-600`, `bg-red-50`).
- Layout is responsive via Tailwind's grid/flex utilities.
- Light theme only -- no dark mode toggle.

## Extending the Dashboard

### Adding a new section

1. Create `components/NewSection.tsx` as a presentational component.
2. Add state and fetch calls in `page.tsx`.
3. Pass data as props and wire callbacks.
4. Add a tab button in the tab bar.

### Adding a new API call

1. Add the typed function to `lib/api.ts`.
2. Define the response type alongside it.
3. Call it from `page.tsx` in the appropriate `useEffect` or callback.

### Adding real-time updates

Wrap the relevant fetch calls in a `setInterval` inside `useEffect` (with cleanup). Or replace individual fetch calls with a WebSocket subscription. The component layer does not change -- it still receives data via props.
