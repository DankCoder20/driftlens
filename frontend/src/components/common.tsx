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
export { ResourceInspector as Detail } from "./ResourceInspector";
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
