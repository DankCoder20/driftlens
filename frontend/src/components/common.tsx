import {
  ArrowUpRight,
  Box,
  Check,
  ChevronRight,
  GitBranch,
  Search,
  X,
} from "lucide-react";
import { stringify } from "yaml";
import type { ReactNode } from "react";
import type { Drift, Resource } from "../types";
export function Badge({ value }: { value: string }) {
  return (
    <span className={`badge ${value.toLowerCase().replaceAll("_", "-")}`}>
      <i />
      {value.replaceAll("_", " ")}
    </span>
  );
}
export function Panel({
  title,
  caption,
  children,
  action,
}: {
  title: string;
  caption?: string;
  children: ReactNode;
  action?: ReactNode;
}) {
  return (
    <section className="panel">
      <div className="panel-head">
        <div>
          <h2>{title}</h2>
          {caption && <p>{caption}</p>}
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}
export function Empty({
  title,
  children,
  action,
}: {
  title: string;
  children: ReactNode;
  action?: () => void;
}) {
  return (
    <div className="empty">
      <div className="empty-icon">
        <GitBranch size={28} />
      </div>
      <h2>{title}</h2>
      <p>{children}</p>
      {action && (
        <button className="primary" onClick={action}>
          Configure connections <ArrowUpRight size={15} />
        </button>
      )}
    </div>
  );
}
export function ResourceTable({
  rows,
  onSelect,
}: {
  rows: Drift[];
  onSelect: (d: Drift) => void;
}) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Resource</th>
            <th>Kind</th>
            <th>Namespace</th>
            <th>Environment</th>
            <th>State</th>
            <th>Health</th>
            <th />
          </tr>
        </thead>
        <tbody>
          {rows.map((d) => (
            <tr
              key={d.id}
              tabIndex={0}
              onClick={() => onSelect(d)}
              onKeyDown={(e) => {
                if (e.key === "Enter") onSelect(d);
              }}
            >
              <td>
                <span className="resource-name">
                  <Box size={15} />
                  {d.name}
                </span>
              </td>
              <td>{d.kind}</td>
              <td className="mono muted">{d.namespace || "cluster-scoped"}</td>
              <td>{d.environment}</td>
              <td>
                <Badge value={d.classification} />
              </td>
              <td>
                {d.actual ? (
                  <Badge value={d.actual.health} />
                ) : (
                  <span className="muted">—</span>
                )}
              </td>
              <td>
                <ChevronRight size={15} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {rows.length === 0 && (
        <div className="small-empty">No resources match these filters.</div>
      )}
    </div>
  );
}
export function Detail({
  drift,
  resource,
  onClose,
}: {
  drift?: Drift;
  resource?: Resource;
  onClose: () => void;
}) {
  const r = resource || drift?.actual || drift?.desired;
  if (!r) return null;
  return (
    <div className="drawer-backdrop" onClick={onClose}>
      <aside
        className="drawer"
        role="dialog"
        aria-modal="true"
        aria-label={`Resource ${r.name}`}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="drawer-head">
          <div className="eyebrow">RESOURCE INSPECTOR</div>
          <button aria-label="Close details" onClick={onClose}>
            <X size={20} />
          </button>
        </div>
        <h2>{r.name}</h2>
        <p className="mono muted">
          {r.kind} / {r.namespace || "cluster-scoped"}
        </p>
        <div className="tags">
          {drift && <Badge value={drift.classification} />}
          <Badge value={r.health} />
          <span>{drift?.ownership || "observed"}</span>
        </div>
        {drift?.reason && <div className="alert">{drift.reason}</div>}
        <dl className="metadata">
          <dt>Identity</dt>
          <dd>{r.id}</dd>
          <dt>Observed</dt>
          <dd>{new Date(r.observedAt).toLocaleString()}</dd>
          <dt>Source</dt>
          <dd>{drift?.desired?.sourceFile || "Cluster inventory"}</dd>
          <dt>Labels</dt>
          <dd>
            {Object.entries(r.labels)
              .map(([k, v]) => `${k}=${v}`)
              .join(", ") || "None"}
          </dd>
        </dl>
        {r.kind === "Secret" && (
          <div className="notice">
            Secret metadata only. Secret values are never fetched or stored.
          </div>
        )}
        {drift && (
          <>
            <h3>
              Field-level changes{" "}
              <span className="count">{drift.differences.length}</span>
            </h3>
            {drift.differences.map((d, i) => (
              <div className="field-diff" key={i}>
                <code>{d.path}</code>
                <div>
                  <span className="diff-minus">
                    − {JSON.stringify(d.desired) ?? "(absent)"}
                  </span>
                  <span className="diff-plus">
                    + {JSON.stringify(d.actual) ?? "(absent)"}
                  </span>
                </div>
              </div>
            ))}
            {!drift.differences.length && (
              <p className="muted">
                {drift.classification === "IN_SYNC" ? (
                  <>
                    <Check size={14} /> No configuration differences.
                  </>
                ) : (
                  "No matched field comparison is available."
                )}
              </p>
            )}
          </>
        )}
        <div className="yaml-grid">
          <div>
            <h3>Desired · Git</h3>
            <pre>
              {drift?.desired
                ? stringify(drift.desired.object)
                : "Not declared in this mapping"}
            </pre>
          </div>
          <div>
            <h3>Actual · Cluster</h3>
            <pre>
              {drift?.actual
                ? stringify(drift.actual.object)
                : resource
                  ? stringify(resource.object)
                  : "Not observed"}
            </pre>
          </div>
        </div>
      </aside>
    </div>
  );
}
export function SearchInput({
  value,
  onChange,
  placeholder = "Search resources…",
}: {
  value: string;
  onChange: (s: string) => void;
  placeholder?: string;
}) {
  return (
    <label className="search">
      <Search size={16} />
      <input
        aria-label={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
      />
    </label>
  );
}
