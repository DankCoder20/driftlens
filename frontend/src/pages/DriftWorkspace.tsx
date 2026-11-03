import { useState } from "react";
import {
  ArrowRight,
  Box,
  ChevronDown,
  ChevronRight,
  GitCompareArrows,
  Layers3,
  Search,
  SlidersHorizontal,
} from "lucide-react";
import type { Cluster, Drift, History, Repository, Snapshot } from "../types";
import { Badge, Empty, Panel } from "../components/common";
import {
  priority,
  reviewGroup,
  reviewGroups,
  summary,
} from "../features/driftPresentation";

type Props = {
  rows: Drift[];
  repos: Repository[];
  clusters: Cluster[];
  onSelect: (d: Drift) => void;
  onConnect: () => void;
  mapped: boolean;
  snapshot?: Snapshot;
};
export function DriftPage(props: Props) {
  return <DriftWorkspace {...props} />;
}
export function Overview({
  snapshot,
  repos,
  clusters,
  onConnect,
  onSelect,
}: {
  snapshot: Snapshot;
  repos: Repository[];
  clusters: Cluster[];
  history: History[];
  onConnect: () => void;
  onDrift: () => void;
  onSelect: (d: Drift) => void;
}) {
  return (
    <DriftWorkspace
      rows={snapshot.drift}
      repos={repos}
      clusters={clusters}
      snapshot={snapshot}
      mapped={snapshot.drift.length > 0}
      onConnect={onConnect}
      onSelect={onSelect}
    />
  );
}
export function DriftWorkspace({
  rows,
  repos,
  clusters,
  onSelect,
  onConnect,
  mapped,
  snapshot,
}: Props) {
  const [environment, setEnvironment] = useState("");
  const [group, setGroup] = useState("declared");
  const [search, setSearch] = useState("");
  const [kind, setKind] = useState("");
  const [repo, setRepo] = useState("");
  const [cluster, setCluster] = useState("");
  const [page, setPage] = useState(0);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const reset = () => setPage(0);
  const environments = [...new Set(rows.map((r) => r.environment))].sort();
  const scoped = rows.filter(
    (r) =>
      (!environment || r.environment === environment) &&
      (!repo || r.repositoryId === repo) &&
      (!cluster || r.clusterId === cluster),
  );
  const counts = Object.fromEntries(
    reviewGroups.map((g) => [
      g.id,
      scoped.filter((r) => reviewGroup(r) === g.id).length,
    ]),
  );
  const selectedGroup = reviewGroups.find((g) => g.id === group);
  const filtered = scoped
    .filter(
      (r) =>
        (group === "all" || reviewGroup(r) === group) &&
        (!kind || r.kind === kind) &&
        `${r.name} ${r.kind} ${r.namespace} ${summary(r)}`
          .toLowerCase()
          .includes(search.toLowerCase()),
    )
    .sort(
      (a, b) =>
        priority(a) - priority(b) ||
        a.environment.localeCompare(b.environment) ||
        a.name.localeCompare(b.name),
    );
  const currentPage = Math.min(
    page,
    Math.max(0, Math.ceil(filtered.length / 20) - 1),
  );
  const visible = filtered.slice(currentPage * 20, currentPage * 20 + 20);
  const scopedResources = new Set(
    scoped
      .filter((r) => r.desired)
      .map((r) => `${r.clusterId}|${r.desired!.id}`),
  ).size;
  if (!mapped && !rows.length)
    return (
      <>
        <Empty title="Start with a clear comparison" action={onConnect}>
          Connect a repository and map a Helm release to its cluster namespace.
          DriftLens will organize the changes by environment.
        </Empty>
        {snapshot && <InventoryExplanation snapshot={snapshot} />}
      </>
    );
  return (
    <>
      <div className="review-intro">
        <div>
          <span className="eyebrow">START HERE</span>
          <h2>What changed in your environments?</h2>
          <p>
            Choose an environment, review the changed resources, then open a
            change to see Git and cluster values.
          </p>
        </div>
        <span className="readonly-pill">Read-only · no automatic fixes</span>
      </div>
      <div className="environment-grid" aria-label="Choose environment">
        {["", ...environments].map((e) => {
          const rr = e ? rows.filter((r) => r.environment === e) : rows;
          const changes = rr.filter(
            (r) => reviewGroup(r) === "declared",
          ).length;
          return (
            <button
              key={e}
              aria-pressed={environment === e}
              className={`environment-card ${environment === e ? "selected" : ""}`}
              onClick={() => {
                setEnvironment(e);
                reset();
              }}
            >
              <div>
                <Layers3 size={17} />
                <strong>{e || "All environments"}</strong>
                {environment === e && (
                  <span className="scope-selected">Selected</span>
                )}
              </div>
              <span className="environment-count">
                {changes}
                <small>declared changes to review</small>
              </span>
              <p>
                {rr.filter((r) => r.desired).length} Git-declared comparisons ·{" "}
                {rr.filter((r) => r.classification === "IN_SYNC").length} in
                sync
              </p>
            </button>
          );
        })}
      </div>
      <div className="scope-summary">
        <span>
          <Box size={15} />
          <b>{scopedResources}</b> Git-declared resources in this scope
        </span>
        <span>
          <GitCompareArrows size={15} />
          <b>{counts.declared}</b> declared configuration changes
        </span>
        <span>
          <b>{counts.added + counts.metadata}</b> live-added / metadata reviews
        </span>
      </div>
      <div className="review-layout">
        <aside className="review-nav" aria-label="Review categories">
          <span className="eyebrow">REVIEW QUEUES</span>
          {reviewGroups.map((g) => (
            <button
              key={g.id}
              aria-pressed={group === g.id}
              className={group === g.id ? "active" : ""}
              onClick={() => {
                setGroup(g.id);
                reset();
              }}
            >
              <span className={`queue-dot ${g.id}`} />
              <span>{g.label}</span>
              <b>{counts[g.id]}</b>
            </button>
          ))}
          <button
            className={group === "all" ? "active" : ""}
            aria-pressed={group === "all"}
            onClick={() => {
              setGroup("all");
              reset();
            }}
          >
            <span>All comparisons</span>
            <b>{scoped.length}</b>
          </button>
          <p>
            These queues organize the existing results. Nothing is ignored or
            marked resolved.
          </p>
        </aside>
        <Panel
          title={selectedGroup?.label || "All comparisons"}
          caption={
            selectedGroup?.description ||
            "Every comparison, including unchanged resources and observations outside the mapping."
          }
          action={
            <span className="subtle-tag">{filtered.length} resources</span>
          }
        >
          <div className="review-tools">
            <label className="search">
              <Search size={16} />
              <input
                aria-label="Search changes"
                placeholder="Search resource, kind, or changed field…"
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  reset();
                }}
              />
            </label>
            <select
              aria-label="Resource kind"
              value={kind}
              onChange={(e) => {
                setKind(e.target.value);
                reset();
              }}
            >
              <option value="">All kinds</option>
              {[...new Set(scoped.map((r) => r.kind))].sort().map((k) => (
                <option key={k}>{k}</option>
              ))}
            </select>
            <details className="advanced-filters">
              <summary>
                <SlidersHorizontal size={14} />
                More filters
              </summary>
              <div>
                <label>
                  Repository
                  <select
                    value={repo}
                    onChange={(e) => {
                      setRepo(e.target.value);
                      reset();
                    }}
                  >
                    <option value="">All repositories</option>
                    {repos.map((r) => (
                      <option value={r.id} key={r.id}>
                        {r.name}
                      </option>
                    ))}
                  </select>
                </label>
                <label>
                  Cluster
                  <select
                    value={cluster}
                    onChange={(e) => {
                      setCluster(e.target.value);
                      reset();
                    }}
                  >
                    <option value="">All clusters</option>
                    {clusters.map((c) => (
                      <option value={c.id} key={c.id}>
                        {c.name}
                      </option>
                    ))}
                  </select>
                </label>
              </div>
            </details>
          </div>
          {(search || kind || repo || cluster) && (
            <div className="filter-note">
              Filtered results
              <button
                className="text-button"
                onClick={() => {
                  setSearch("");
                  setKind("");
                  setRepo("");
                  setCluster("");
                  reset();
                }}
              >
                Clear filters
              </button>
            </div>
          )}
          <div className="change-list">
            {visible.map((d) => (
              <div
                key={d.id}
                className={`change-item ${d.classification === "MISSING" ? "missing" : ""}`}
              >
                <button className="change-open" onClick={() => onSelect(d)}>
                  <span className="change-kind">
                    <Box size={18} />
                  </span>
                  <span className="change-main">
                    <strong>{d.name}</strong>
                    <span>
                      {d.kind} <i>·</i> {d.environment} <i>·</i>{" "}
                      {d.namespace || "cluster-scoped"}
                    </span>
                    <p>{summary(d)}</p>
                  </span>
                  <span className="change-state">
                    <Badge value={d.classification} />
                    <small>
                      {d.differences.length
                        ? `${d.differences.length} changed fields`
                        : d.ownership === "unmanaged"
                          ? "Unmanaged resource"
                          : ""}
                    </small>
                  </span>
                  <ChevronRight size={18} />
                </button>
                {d.differences.length > 0 && (
                  <div className="change-expand">
                    <button
                      aria-expanded={!!expanded[d.id]}
                      onClick={() =>
                        setExpanded({ ...expanded, [d.id]: !expanded[d.id] })
                      }
                    >
                      <ChevronDown size={13} />{" "}
                      {expanded[d.id] ? "Hide" : "Preview"} changed paths
                    </button>
                    {expanded[d.id] && (
                      <ul>
                        {d.differences.slice(0, 5).map((x) => (
                          <li key={x.path}>
                            <code>{x.path}</code>
                          </li>
                        ))}
                        {d.differences.length > 5 && (
                          <li>
                            +{d.differences.length - 5} more in the inspector
                          </li>
                        )}
                      </ul>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
          {!filtered.length && (
            <div className="review-empty">
              <GitCompareArrows size={26} />
              <h3>No resources in this view</h3>
              <p>
                {search || kind
                  ? "Try clearing a filter."
                  : "Other queues may still contain differences requiring review."}
              </p>
              <button
                onClick={() => {
                  setGroup("all");
                  setSearch("");
                  setKind("");
                  reset();
                }}
              >
                See all comparisons <ArrowRight size={14} />
              </button>
            </div>
          )}
          <div className="pagination">
            <span>
              {filtered.length ? currentPage * 20 + 1 : 0}–
              {Math.min((currentPage + 1) * 20, filtered.length)} of{" "}
              {filtered.length} resources
            </span>
            <div>
              <button
                disabled={currentPage === 0}
                onClick={() => setPage(currentPage - 1)}
              >
                Previous
              </button>
              <span>Page {currentPage + 1}</span>
              <button
                disabled={(currentPage + 1) * 20 >= filtered.length}
                onClick={() => setPage(currentPage + 1)}
              >
                Next
              </button>
            </div>
          </div>
        </Panel>
      </div>
      {snapshot && <InventoryExplanation snapshot={snapshot} />}
      <div className="notice">
        Review grouping is a presentation aid, not a severity score.
        “Live-added” does not mean safe to ignore. Kubernetes defaults and
        controller mutations may need investigation.
      </div>
    </>
  );
}
function InventoryExplanation({ snapshot: s }: { snapshot: Snapshot }) {
  const entries = Object.entries(s.summary.kinds).sort((a, b) => b[1] - a[1]);
  return (
    <details className="inventory-explanation">
      <summary>
        <div>
          <Box size={18} />
          <span>
            <strong>
              Why are there {s.summary.total.toLocaleString()} observed objects?
            </strong>
            <small>
              Cluster inventory includes events, runtime children,
              access-control objects, and system resources. This is not a count
              of application drifts.
            </small>
          </span>
        </div>
        <ChevronDown size={18} />
      </summary>
      <div className="inventory-breakdown">
        {entries.map(([kind, count]) => (
          <div key={kind}>
            <span>{kind}</span>
            <b>{count.toLocaleString()}</b>
            <meter min={0} max={s.summary.total || 1} value={count} />
          </div>
        ))}
      </div>
      <p>
        Inventory covers the connected cluster. The review queues above cover
        only explicitly configured environment mappings. Open Clusters for the
        full inventory.
      </p>
    </details>
  );
}
