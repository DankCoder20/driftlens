import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Background,
  Controls,
  MiniMap,
  ReactFlow,
  MarkerType,
  Position,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import type { Cluster, Drift, Resource } from "../types";
import { api } from "../services/api";
import { Detail, Empty } from "../components/common";
export function Topology({
  clusters,
  drift,
}: {
  clusters: Cluster[];
  drift: Drift[];
}) {
  const [cluster, setCluster] = useState("");
  const [ns, setNs] = useState("");
  const [source, setSource] = useState("actual");
  const [selected, setSelected] = useState<Resource>();
  const id = cluster || clusters[0]?.id;
  const { data, error, isLoading } = useQuery({
    queryKey: ["topology", id, source],
    queryFn: () =>
      api<{
        nodes: Resource[];
        edges: { id: string; source: string; target: string; label: string }[];
      }>(`/topology/${id}?source=${source}`),
    enabled: !!id,
  });
  const namespaces = [
    ...new Set(data?.nodes.map((n) => n.namespace).filter(Boolean)),
  ].sort();
  const graph = useMemo(() => {
    const ranks: Record<string, number> = {
      Namespace: 0,
      Ingress: 1,
      Service: 2,
      Deployment: 3,
      StatefulSet: 3,
      DaemonSet: 3,
      ReplicaSet: 4,
      Pod: 5,
      ConfigMap: 6,
      Secret: 6,
      PersistentVolumeClaim: 6,
    };
    const counts: Record<number, number> = {};
    const filtered = (data?.nodes || []).filter(
      (r) =>
        (!ns ||
          r.namespace === ns ||
          (r.kind === "Namespace" && r.name === ns)) &&
        !["Event", "Endpoints", "EndpointSlice"].includes(r.kind),
    );
    const visible = new Set(filtered.map((r) => r.id));
    return {
      nodes: filtered.map((r) => {
        const col = ranks[r.kind] ?? 7;
        const row = counts[col] || 0;
        counts[col] = row + 1;
        const d = drift.find(
          (d) => d.clusterId === id && (d.actual?.id || d.desired?.id) === r.id,
        );
        return {
          id: r.id,
          position: { x: col * 245, y: row * 110 },
          sourcePosition: Position.Right,
          targetPosition: Position.Left,
          data: {
            label: (
              <div className="topology-node">
                <small>{r.kind}</small>
                <strong>{r.name}</strong>
                <span>{d?.classification || r.health}</span>
              </div>
            ),
          },
          style: {
            background: "#141d29",
            color: "#dce6f1",
            border: `1px solid ${d?.classification === "MODIFIED" ? "#e8b365" : r.health === "Degraded" ? "#e77e8e" : "#34465b"}`,
            borderRadius: 9,
            width: 205,
          },
        };
      }),
      edges: (data?.edges || [])
        .filter((e) => visible.has(e.source) && visible.has(e.target))
        .map((e) => ({
          ...e,
          style: { stroke: "#42546c" },
          labelStyle: { fill: "#98a6bb", fontSize: 10 },
          labelBgStyle: { fill: "#0e151f" },
          markerEnd: { type: MarkerType.ArrowClosed, color: "#42546c" },
        })),
    };
  }, [data, ns, drift, id]);
  if (!clusters.length)
    return (
      <Empty title="Your infrastructure, connected">
        Add a cluster and run a scan to discover real relationships.
      </Empty>
    );
  return (
    <>
      <div className="toolbar">
        <select
          aria-label="Topology source"
          value={source}
          onChange={(e) => {
            setSource(e.target.value);
            setSelected(undefined);
          }}
        >
          <option value="actual">Actual · Cluster</option>
          <option value="desired">Desired · Git</option>
        </select>
        <select
          aria-label="Topology cluster"
          value={id}
          onChange={(e) => {
            setCluster(e.target.value);
            setNs("");
          }}
        >
          {clusters.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
        <select
          aria-label="Topology namespace"
          value={ns}
          onChange={(e) => setNs(e.target.value)}
        >
          <option value="">All namespaces</option>
          {namespaces.map((n) => (
            <option key={n}>{n}</option>
          ))}
        </select>
        <span className="muted">
          {graph.nodes.length} resources · {graph.edges.length} verified
          relationships
        </span>
      </div>
      {error && <div className="alert">{error.message}</div>}
      {isLoading ? (
        <div className="skeleton tall" />
      ) : !graph.nodes.length ? (
        <Empty title="No topology collected">
          Run a scan to discover selectors, owner references and resource
          dependencies.
        </Empty>
      ) : (
        <div className="topology-canvas">
          <ReactFlow
            key={`${id}-${ns}-${source}`}
            nodes={graph.nodes}
            edges={graph.edges}
            fitView
            minZoom={0.08}
            maxZoom={2}
            colorMode="dark"
            onNodeClick={(_, n) =>
              setSelected(data?.nodes.find((r) => r.id === n.id))
            }
          >
            <Background color="#263142" gap={22} />
            <Controls />
            <MiniMap nodeColor="#426a60" maskColor="#0b101ac9" />
          </ReactFlow>
        </div>
      )}
      <p className="footnote">
        Edges come from namespace membership, owner references, service
        selectors and explicit resource references. Runtime children remain in
        inventory without becoming extra-resource drift.
      </p>
      {selected && (
        <Detail
          resource={source === "actual" ? selected : undefined}
          drift={drift.find(
            (d) => (d.actual?.id || d.desired?.id) === selected.id && d.clusterId === id,
          )}
          onClose={() => setSelected(undefined)}
        />
      )}
    </>
  );
}
