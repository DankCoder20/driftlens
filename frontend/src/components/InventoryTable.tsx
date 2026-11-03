import { useState } from "react";
import type { Resource } from "../types";
import { Badge } from "./common";
const groups = [
  {
    id: "workloads",
    name: "Workloads",
    kinds: ["Deployment", "StatefulSet", "DaemonSet", "CronJob", "Job", "Pod"],
  },
  {
    id: "network",
    name: "Networking",
    kinds: ["Service", "Ingress", "HTTPRoute", "Gateway", "NetworkPolicy"],
  },
  {
    id: "config",
    name: "Configuration",
    kinds: [
      "ConfigMap",
      "Secret",
      "ExternalSecret",
      "SecretStore",
      "PersistentVolumeClaim",
    ],
  },
  {
    id: "runtime",
    name: "Runtime history",
    kinds: ["ReplicaSet", "ControllerRevision", "EndpointSlice", "Endpoints"],
  },
  { id: "events", name: "Events", kinds: ["Event"] },
  { id: "all", name: "All objects", kinds: [] },
];
export function InventoryTable({
  resources,
  onSelect,
}: {
  resources: Resource[];
  onSelect: (r: Resource) => void;
}) {
  const [group, setGroup] = useState("workloads");
  const [page, setPage] = useState(0);
  const kinds = groups.find((g) => g.id === group)!.kinds;
  const rows = resources
    .filter((r) => !kinds.length || kinds.includes(r.kind))
    .sort(
      (a, b) =>
        a.namespace.localeCompare(b.namespace) ||
        a.kind.localeCompare(b.kind) ||
        a.name.localeCompare(b.name),
    );
  const current = Math.min(page, Math.max(0, Math.ceil(rows.length / 30) - 1));
  return (
    <>
      <div className="inventory-tabs">
        {groups.map((g) => (
          <button
            aria-pressed={g.id === group}
            className={g.id === group ? "active" : ""}
            key={g.id}
            onClick={() => {
              setGroup(g.id);
              setPage(0);
            }}
          >
            {g.name}
            <span>
              {
                resources.filter(
                  (r) => !g.kinds.length || g.kinds.includes(r.kind),
                ).length
              }
            </span>
          </button>
        ))}
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Resource</th>
              <th>Kind</th>
              <th>Namespace</th>
              <th>Health</th>
              <th>Ready / desired</th>
            </tr>
          </thead>
          <tbody>
            {rows.slice(current * 30, current * 30 + 30).map((r) => (
              <tr
                key={r.id}
                tabIndex={0}
                onClick={() => onSelect(r)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") onSelect(r);
                }}
              >
                <td>{r.name}</td>
                <td>{r.kind}</td>
                <td>{r.namespace || "cluster-scoped"}</td>
                <td>
                  {r.health && r.health !== "Unknown" ? (
                    <Badge value={r.health} />
                  ) : (
                    <span className="muted">Not applicable</span>
                  )}
                </td>
                <td>
                  {["Deployment", "StatefulSet", "DaemonSet"].includes(r.kind)
                    ? `${r.readyReplicas} / ${r.desiredReplicas}`
                    : "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {!rows.length && (
        <div className="small-empty">
          No objects in this category and namespace selection.
        </div>
      )}
      <div className="pagination">
        <span>
          {rows.length ? current * 30 + 1 : 0}–
          {Math.min((current + 1) * 30, rows.length)} of {rows.length} objects
        </span>
        <div>
          <button disabled={!current} onClick={() => setPage(current - 1)}>
            Previous
          </button>
          <button
            disabled={(current + 1) * 30 >= rows.length}
            onClick={() => setPage(current + 1)}
          >
            Next
          </button>
        </div>
      </div>
    </>
  );
}
