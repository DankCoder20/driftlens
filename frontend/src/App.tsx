import { InventoryTable } from "./components/InventoryTable";
import { Overview, DriftPage } from "./pages/DriftWorkspace";
import { reviewGroup } from "./features/driftPresentation";
import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  ArrowRight,
  ArrowUpRight,
  BarChart3,
  Box,
  CheckCheck,
  ChevronRight,
  Clock3,
  Command,
  Database,
  FileCode2,
  GitBranch,
  GitCompareArrows,
  History as HistoryIcon,
  Layers3,
  LayoutDashboard,
  Link2,
  Network,
  RefreshCw,
  Server,
  ShieldCheck,
  TriangleAlert,
} from "lucide-react";
import { api } from "./services/api";
import { useLive } from "./hooks/useLive";
import {
  Badge,
  Detail,
  Empty,
  Panel,
  ResourceTable,
  SearchInput,
} from "./components/common";
import { Distribution, Trend } from "./components/charts";
import { Connections } from "./pages/Connections";
import { Topology } from "./pages/Topology";
import type {
  Cluster,
  Discovery,
  Drift,
  History,
  Mapping,
  Repository,
  Resource,
  Snapshot,
} from "./types";
const pages = [
  {
    name: "Overview",
    icon: LayoutDashboard,
    description: "A clear picture of what is declared, deployed, and drifting.",
  },
  {
    name: "Repositories",
    icon: GitBranch,
    description: "The source of truth for your infrastructure.",
  },
  {
    name: "Clusters",
    icon: Server,
    description: "Live inventory and workload health across your environments.",
  },
  {
    name: "Topology",
    icon: Network,
    description: "Follow the relationships behind your infrastructure.",
  },
  {
    name: "Drift Analysis",
    icon: GitCompareArrows,
    description: "Find the distance between desired and actual state.",
  },
  {
    name: "Analytics",
    icon: BarChart3,
    description: "Patterns and trends from real infrastructure snapshots.",
  },
  {
    name: "Change History",
    icon: HistoryIcon,
    description: "An observed timeline of configuration and health changes.",
  },
  {
    name: "Connections",
    icon: Link2,
    description: "Connect sources. Map environments. Set your scan cadence.",
  },
];
function timestamp(s?: string) {
  return s && !s.startsWith("0001")
    ? new Date(s).toLocaleString()
    : "No scans yet";
}
function trendRows(history: History[]) {
  return [...history].reverse().map((h) => ({
    time: new Date(h.createdAt).toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    }),
    drift:
      (h.summary.classifications.MODIFIED || 0) +
      (h.summary.classifications.MISSING || 0) +
      (h.summary.classifications.EXTRA || 0),
    resources: h.summary.total,
    unhealthy: h.summary.unhealthy,
    ready: h.summary.readyReplicas,
    desired: h.summary.desiredReplicas,
    pending: h.summary.health.Pending || 0,
    failed: h.summary.health.Failed || 0,
  }));
}
export default function App() {
  const readPage = () => {
    try {
      const name = decodeURIComponent(location.hash.slice(1));
      return pages.some((p) => p.name === name) ? name : "Overview";
    } catch {
      return "Overview";
    }
  };
  const [page, setPage] = useState(readPage);
  useEffect(() => {
    const changed = () => setPage(readPage());
    window.addEventListener("hashchange", changed);
    return () => window.removeEventListener("hashchange", changed);
  }, []);
  const [selected, setSelected] = useState<Drift>();
  const [selectedResource, setSelectedResource] = useState<Resource>();
  const [auth, setAuth] = useState(
    sessionStorage.getItem("driftlens-token") || "",
  );
  const q = useQueryClient();
  useLive();
  function navigate(p: string) {
    setPage(p);
    location.hash = p;
  }
  const state = useQuery({
    queryKey: ["state"],
    queryFn: () =>
      api<{ snapshot: Snapshot; scanning: boolean }>("/state?compact=true"),
    refetchInterval: 5000,
  });
  const repos = useQuery({
    queryKey: ["repositories"],
    queryFn: () => api<Repository[]>("/repositories"),
  });
  const clusters = useQuery({
    queryKey: ["clusters"],
    queryFn: () => api<Cluster[]>("/clusters"),
  });
  const mappings = useQuery({
    queryKey: ["mappings"],
    queryFn: () => api<Mapping[]>("/mappings"),
  });
  const history = useQuery({
    queryKey: ["history"],
    queryFn: () => api<History[]>("/history"),
  });
  const scan = useMutation({
    mutationFn: () => api("/scans", "POST", {}),
    onSuccess: () => q.invalidateQueries(),
  });
  const s = state.data?.snapshot;
  const rs = repos.data || [],
    cs = clusters.data || [],
    ms = mappings.data || [],
    hs = history.data || [];
  const meta = pages.find((p) => p.name === page) || pages[0];
  const error =
    state.error ||
    repos.error ||
    clusters.error ||
    mappings.error ||
    history.error;
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <a
          className="brand"
          href="#Overview"
          onClick={() => navigate("Overview")}
        >
          <span className="brand-mark">
            <Layers3 size={22} />
          </span>
          driftlens<span className="beta">BETA</span>
        </a>
        <div className="workspace">
          <span className="workspace-icon">DL</span>
          <div>
            Local workspace<small>Self-hosted infrastructure</small>
          </div>
          <ChevronRight size={14} />
        </div>
        <div className="nav-label">WORKSPACE</div>
        <nav>
          {pages.slice(0, 7).map((p) => (
            <button
              key={p.name}
              className={page === p.name ? "active" : ""}
              onClick={() => navigate(p.name)}
            >
              <p.icon size={17} />
              {p.name}
              {p.name === "Drift Analysis" &&
                !!s?.drift.filter((d) => reviewGroup(d) === "declared")
                  .length && (
                  <span className="nav-count">
                    {
                      s.drift.filter((d) => reviewGroup(d) === "declared")
                        .length
                    }
                  </span>
                )}
            </button>
          ))}
        </nav>
        <div className="sidebar-bottom">
          <div className="nav-label">CONFIGURATION</div>
          <nav>
            <button
              className={page === "Connections" ? "active" : ""}
              onClick={() => navigate("Connections")}
            >
              <Link2 size={17} />
              Connections
            </button>
          </nav>
          <div className="read-only">
            <ShieldCheck size={17} />
            <div>
              Observe. Understand.<small>Always read-only.</small>
            </div>
          </div>
          <div className="profile">
            <span>SN</span>
            <div>
              Infrastructure workspace<small>DriftLens / v0.1</small>
            </div>
            <Command size={15} />
          </div>
        </div>
      </aside>
      <div className="main-shell">
        <header className="topbar">
          <div className="breadcrumb">
            Workspace <ChevronRight size={13} />
            <span>{meta.name}</span>
          </div>
          <div className="topbar-right">
            <span className="connection-status">
              <i className={error ? "bad" : ""} />
              {error
                ? "Connection error"
                : state.data?.scanning
                  ? "Collecting inventory"
                  : "Local connection"}
            </span>
            <span className="top-divider" />
            <ShieldCheck size={14} />
            <span>Read-only</span>
          </div>
        </header>
        <main>
          <div className="page-heading">
            <div>
              <div className="eyebrow">INFRASTRUCTURE INTELLIGENCE</div>
              <h1>{meta.name}</h1>
              <p>{meta.description}</p>
            </div>
            <div className="heading-actions">
              <span className="last-scan">
                <Clock3 size={13} />
                {timestamp(s?.createdAt)}
              </span>
              <button
                className="primary"
                onClick={() => scan.mutate()}
                disabled={
                  scan.isPending ||
                  state.data?.scanning ||
                  (!cs.length && !rs.length)
                }
              >
                <RefreshCw
                  size={15}
                  className={state.data?.scanning ? "spin" : ""}
                />
                {state.data?.scanning ? "Scanning…" : "Run scan"}
              </button>
            </div>
          </div>
          {error ? (
            <div className="connection-error">
              <TriangleAlert size={25} />
              <h2>Unable to connect</h2>
              <p role="alert">{error.message}</p>
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  sessionStorage.setItem("driftlens-token", auth);
                  void q.invalidateQueries();
                }}
              >
                <label>
                  Server API token
                  <input
                    aria-label="Server API token"
                    type="password"
                    value={auth}
                    onChange={(e) => setAuth(e.target.value)}
                    placeholder="Enter DRIFTLENS_API_TOKEN"
                  />
                </label>
                <button className="primary">Connect</button>
              </form>
              <p className="muted">
                The token is kept in this browser tab’s session. Check that the
                backend is running.
              </p>
            </div>
          ) : state.isLoading ? (
            <div className="loading-grid">
              {Array.from({ length: 6 }, (_, i) => (
                <div key={i} className="skeleton tall" />
              ))}
            </div>
          ) : (
            <>
              {scan.error && (
                <div className="alert" role="alert">
                  {scan.error.message}
                </div>
              )}
              {s && Object.keys(s.errors).length > 0 && (
                <details className="scan-errors">
                  <summary>
                    <TriangleAlert size={15} />
                    Last scan was incomplete · {
                      Object.keys(s.errors).length
                    }{" "}
                    connection or permission issues
                  </summary>
                  {Object.entries(s.errors).map(([k, v]) => (
                    <p key={k}>
                      <code>
                        {rs.find((r) => r.id === k)?.name ||
                          cs.find((c) => c.id === k)?.name ||
                          ms.find((m) => m.id === k)?.name ||
                          k}
                      </code>{" "}
                      — {v}
                    </p>
                  ))}
                </details>
              )}
              {page === "Connections" ? (
                <Connections repos={rs} clusters={cs} mappings={ms} />
              ) : page === "Topology" ? (
                <Topology clusters={cs} drift={s?.drift || []} />
              ) : page === "Repositories" ? (
                <Repositories
                  repos={rs}
                  snapshot={s}
                  mappings={ms}
                  onConnect={() => navigate("Connections")}
                  onSelect={setSelected}
                />
              ) : page === "Clusters" ? (
                <Clusters
                  clusters={cs}
                  snapshot={s}
                  onConnect={() => navigate("Connections")}
                  onSelect={setSelectedResource}
                />
              ) : page === "Drift Analysis" ? (
                <DriftPage
                  rows={s?.drift || []}
                  repos={rs}
                  clusters={cs}
                  onSelect={setSelected}
                  onConnect={() => navigate("Connections")}
                  mapped={ms.length > 0}
                />
              ) : page === "Change History" ? (
                <HistoryPage history={hs} />
              ) : page === "Analytics" && s ? (
                <Analytics snapshot={s} history={hs} clusters={cs} />
              ) : s ? (
                <Overview
                  snapshot={s}
                  repos={rs}
                  clusters={cs}
                  history={hs}
                  onConnect={() => navigate("Connections")}
                  onDrift={() => navigate("Drift Analysis")}
                  onSelect={setSelected}
                />
              ) : null}
            </>
          )}
          <footer>
            <span>
              <Layers3 size={13} /> DRIFTLENS <i /> Infrastructure, in focus.
            </span>
            <span>Git → Render → Observe → Compare</span>
          </footer>
        </main>
      </div>
      {(selected || selectedResource) && (
        <Detail
          drift={selected}
          resource={selectedResource}
          onClose={() => {
            setSelected(undefined);
            setSelectedResource(undefined);
          }}
        />
      )}
    </div>
  );
}
function Repositories({
  repos,
  snapshot: s,
  mappings,
  onConnect,
  onSelect,
}: {
  repos: Repository[];
  snapshot?: Snapshot;
  mappings: Mapping[];
  onConnect: () => void;
  onSelect: (d: Drift) => void;
}) {
  const [selected, setSelected] = useState("");
  const [discovery, setDiscovery] = useState<Record<string, Discovery>>({});
  const [chart, setChart] = useState("");
  const test = useMutation({
    mutationFn: (id: string) =>
      api<Discovery>(`/repositories/${id}/test`, "POST"),
    onSuccess: (d, id) => setDiscovery({ ...discovery, [id]: d }),
  });
  if (!repos.length)
    return (
      <Empty title="Your source of truth belongs here" action={onConnect}>
        Connect public or private Git repositories, or a local infrastructure
        directory.
      </Empty>
    );
  const repo = repos.find((r) => r.id === selected) || repos[0];
  const d = discovery[repo.id] || s?.repositories[repo.id];
  const chosen = d?.charts.find((c) => c.path === chart) || d?.charts[0];
  return (
    <>
      <div className="source-cards">
        {repos.map((r) => (
          <button
            key={r.id}
            className={`source-card ${repo.id === r.id ? "selected" : ""}`}
            onClick={() => {
              setSelected(r.id);
              setChart("");
            }}
          >
            <GitBranch size={21} />
            <strong>{r.name}</strong>
            <small>{r.url}</small>
            <span className="subtle-tag">{r.branch || "default branch"}</span>
          </button>
        ))}
      </div>
      <div className="repo-layout">
        <Panel
          title={repo.name}
          caption={`Commit: ${d?.commit || "Not scanned"}`}
          action={
            <button
              disabled={test.isPending}
              onClick={() => test.mutate(repo.id)}
            >
              {test.isPending ? "Discovering…" : "Discover files"}
            </button>
          }
        >
          {test.error && <div className="alert">{test.error.message}</div>}
          <div className="file-tree">
            {d?.files.map((f) => (
              <div key={f}>
                <FileCode2 size={14} />
                <code>{f}</code>
              </div>
            ))}
            {!d && (
              <div className="small-empty">
                Discover this repository or run a scan to browse its contents.
              </div>
            )}
          </div>
        </Panel>
        <div className="stack">
          <Panel
            title="Helm charts"
            caption="Discovered chart metadata and explicit values configuration"
          >
            {d?.charts.map((c) => (
              <button
                className={`chart-card ${chosen?.path === c.path ? "selected" : ""}`}
                key={c.path}
                onClick={() => setChart(c.path)}
              >
                <Box size={22} />
                <div>
                  <strong>{c.name}</strong>
                  <small>{c.path}</small>
                </div>
                <span className="subtle-tag">v{c.version}</span>
              </button>
            ))}
            {d && !d.charts.length && (
              <div className="small-empty">
                No Helm charts found. Raw manifest paths can still be mapped.
              </div>
            )}
            {chosen && (
              <div className="chart-detail">
                <p>{chosen.description}</p>
                <h3>Available values files</h3>
                {chosen.valuesFiles.map((v) => (
                  <code key={v}>{v}</code>
                ))}
                <h3>Configured environments</h3>
                {mappings
                  .filter(
                    (m) =>
                      m.repositoryId === repo.id && m.chartPath === chosen.path,
                  )
                  .map((m) => (
                    <div key={m.id}>
                      <strong>{m.name}</strong>
                      <p className="mono">
                        {m.valuesFiles.join(" → ") || "Chart defaults only"}
                      </p>
                    </div>
                  ))}
              </div>
            )}
          </Panel>
          <Panel
            title="Rendered resources"
            caption="Select a resource for source metadata and desired YAML"
          >
            <ResourceTable
              rows={(s?.drift || []).filter(
                (d) =>
                  d.repositoryId === repo.id &&
                  d.desired &&
                  (!chosen ||
                    mappings.find((m) => m.id === d.mappingId)?.chartPath ===
                      chosen.path),
              )}
              onSelect={onSelect}
            />
          </Panel>
        </div>
      </div>
    </>
  );
}
function Clusters({
  clusters,
  snapshot: s,
  onConnect,
  onSelect,
}: {
  clusters: Cluster[];
  snapshot?: Snapshot;
  onConnect: () => void;
  onSelect: (r: Resource) => void;
}) {
  const [id, setId] = useState("");
  const [ns, setNs] = useState("__mapped__");
  const [search, setSearch] = useState("");
  const inventory = useQuery({
    queryKey: ["inventory", s?.id],
    queryFn: () => api<Resource[]>("/inventory"),
    enabled: clusters.length > 0,
  });
  if (!clusters.length)
    return (
      <Empty title="See what is actually running" action={onConnect}>
        Connect a non-production cluster with a read-only kubeconfig to discover
        workloads and their health.
      </Empty>
    );
  const selected = id || clusters[0].id;
  const resources =
    inventory.data?.filter((r) => r.clusterId === selected) || [];
  const mappedNamespaces = new Set(
    s?.drift
      .filter((d) => d.clusterId === selected)
      .map((d) => d.namespace)
      .filter(Boolean),
  );
  const filtered = resources.filter(
    (r) =>
      (!ns ||
        (ns === "__mapped__"
          ? !mappedNamespaces.size || mappedNamespaces.has(r.namespace)
          : r.namespace === ns)) &&
      `${r.name} ${r.kind}`.toLowerCase().includes(search.toLowerCase()),
  );
  return (
    <>
      <div className="source-cards">
        {clusters.map((c) => {
          const status = s?.clusters[c.id];
          const pods =
            inventory.data?.filter(
              (r) => r.clusterId === c.id && r.kind === "Pod",
            ) || [];
          return (
            <button
              className={`source-card cluster-card ${selected === c.id ? "selected" : ""}`}
              key={c.id}
              onClick={() => {
                setId(c.id);
                setNs("__mapped__");
              }}
            >
              <div className="card-title">
                <Server size={22} />
                <strong>{c.name}</strong>
                <Badge value={status?.status || "Not scanned"} />
              </div>
              <small>Kubernetes {status?.version || "—"}</small>
              <div className="cluster-stats">
                <span>
                  <b>{status?.nodeCount ?? "—"}</b>Nodes
                </span>
                <span>
                  <b>{status?.namespaces.length ?? "—"}</b>Namespaces
                </span>
                <span>
                  <b>{s?.summary.clusters[c.id] ?? "—"}</b>Resources
                </span>
              </div>
              <small>
                {pods.filter((p) => p.health === "Running").length} running ·{" "}
                {pods.filter((p) => p.health === "Pending").length} pending ·{" "}
                {pods.filter((p) => p.health === "Failed").length} failed pods
              </small>
            </button>
          );
        })}
      </div>
      <Panel
        title="Live inventory"
        caption="Starts with workloads in mapped environments. Other categories remain available below."
      >
        {inventory.isLoading && (
          <div className="small-empty">Loading cluster inventory…</div>
        )}
        {inventory.error && (
          <div role="alert" className="alert">
            {inventory.error.message}
          </div>
        )}
        <div className="toolbar">
          <SearchInput value={search} onChange={setSearch} />
          <select
            aria-label="Cluster namespace"
            value={ns}
            onChange={(e) => setNs(e.target.value)}
          >
            <option value="__mapped__">Mapped environments</option>
            <option value="">All namespaces</option>
            {s?.clusters[selected]?.namespaces.map((n) => (
              <option key={n}>{n}</option>
            ))}
          </select>
          <span className="muted">{filtered.length} objects</span>
        </div>
        <InventoryTable resources={filtered} onSelect={onSelect} />
      </Panel>
    </>
  );
}
function Analytics({
  snapshot: s,
  history,
  clusters,
}: {
  snapshot: Snapshot;
  history: History[];
  clusters: Cluster[];
}) {
  const rows = trendRows(history);
  return (
    <>
      <div className="notice">
        <Database size={16} /> Charts use {history.length} stored snapshots. No
        synthetic values or projected history.
      </div>
      <div className="charts-grid">
        <Panel title="Configuration drift">
          <Trend
            rows={rows}
            series={[{ key: "drift", name: "Drifted", color: "#f3be6e" }]}
          />
        </Panel>
        <Panel title="Resource growth">
          <Trend
            rows={rows}
            series={[{ key: "resources", name: "Resources", color: "#7599ff" }]}
          />
        </Panel>
        <Panel title="Pod and workload health">
          <Trend
            rows={rows}
            series={[
              {
                key: "unhealthy",
                name: "Unhealthy workloads",
                color: "#f77f8e",
              },
              { key: "pending", name: "Pending pods", color: "#f3be6e" },
              { key: "failed", name: "Failed pods", color: "#b69bff" },
            ]}
          />
        </Panel>
        <Panel title="Deployment & StatefulSet availability">
          <Trend
            rows={rows}
            series={[
              { key: "desired", name: "Desired replicas", color: "#7599ff" },
              { key: "ready", name: "Ready replicas", color: "#5ce4b0" },
            ]}
          />
        </Panel>
        <Panel title="Cluster resource distribution">
          <Distribution
            data={Object.fromEntries(
              clusters.map((c) => [c.name, s.summary.clusters[c.id] || 0]),
            )}
          />
        </Panel>
        <Panel title="Namespace distribution">
          <Distribution data={s.summary.namespaces} />
        </Panel>
        <Panel
          title="Container image versions"
          caption="Images observed on actual Pods"
        >
          <Distribution data={s.summary.images} />
        </Panel>
        <Panel title="Desired versus ready replicas">
          <Distribution
            data={{
              Desired: s.summary.desiredReplicas,
              Ready: s.summary.readyReplicas,
            }}
          />
        </Panel>
      </div>
    </>
  );
}
function HistoryPage({ history }: { history: History[] }) {
  if (!history.length)
    return (
      <Empty title="Every snapshot tells a story">
        Run a scan to begin tracking resource changes, drift transitions and
        workload health.
      </Empty>
    );
  return (
    <div className="history-list">
      {history.map((h) => (
        <Panel
          key={h.id}
          title={timestamp(h.createdAt)}
          caption={`Scan ${h.id.slice(0, 8)} · ${h.summary.total} observed resources`}
          action={<Badge value={h.status} />}
        >
          {Object.entries(h.repositories).map(([id, r]) => (
            <div className="commit-row" key={id}>
              <GitBranch size={14} />
              <code>{r.commit}</code>
              <span>{r.charts.length} charts scanned</span>
            </div>
          ))}
          {Object.values(h.errors).map((e, i) => (
            <div className="alert" key={i}>
              {e}
            </div>
          ))}
          {h.changes.length ? (
            h.changes.map((c, i) => (
              <div className="history-row" key={i}>
                <span className={`history-dot ${c.type}`} />
                <div>
                  <strong>{c.type.replaceAll("_", " ")}</strong>
                  <code>{c.resource.split("|")[1] || c.resource}</code>
                </div>
                <span>
                  {c.before || "—"} <ArrowRight size={12} /> {c.after || "—"}
                </span>
              </div>
            ))
          ) : (
            <div className="small-empty">
              No resource or health changes from the previous snapshot.
            </div>
          )}
        </Panel>
      ))}
    </div>
  );
}
