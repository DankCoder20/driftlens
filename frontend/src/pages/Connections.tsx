import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check, GitBranch, Link2, Plus, Server, Trash2 } from "lucide-react";
import { api } from "../services/api";
import type { Cluster, Mapping, Repository } from "../types";
import { Panel } from "../components/common";
export function Connections({
  repos,
  clusters,
  mappings,
}: {
  repos: Repository[];
  clusters: Cluster[];
  mappings: Mapping[];
}) {
  const q = useQueryClient();
  const [form, setForm] = useState("repository");
  const [notice, setNotice] = useState("");
  const settings = useQuery({
    queryKey: ["settings"],
    queryFn: () => api<{ scanInterval: number }>("/settings"),
  });
  const mutation = useMutation({
    mutationFn: ({
      path,
      method = "POST",
      body,
    }: {
      path: string;
      method?: string;
      body?: unknown;
    }) => api(path, method, body),
    onSuccess: (_, v) => {
      setNotice(
        v.path.endsWith("/test")
          ? "Connection verified."
          : v.method === "DELETE"
            ? "Connection removed."
            : "Configuration saved.",
      );
      void q.invalidateQueries();
    },
    onError: () => setNotice(""),
  });
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    const get = (k: string) => String(f.get(k) || "");
    let path = "",
      body: unknown;
    if (form === "repository") {
      path = "/repositories";
      body = { name: get("name"), url: get("url"), branch: get("branch") };
    } else if (form === "cluster") {
      path = "/clusters";
      body = {
        name: get("name"),
        kubeconfig: get("kubeconfig"),
        context: get("context"),
      };
    } else {
      path = "/mappings";
      body = {
        name: get("name"),
        repositoryId: get("repositoryId"),
        clusterId: get("clusterId"),
        chartPath: get("chartPath"),
        manifestPath: get("manifestPath"),
        release: get("release"),
        namespace: get("namespace"),
        valuesFiles: get("valuesFiles")
          .split("\n")
          .map((s) => s.trim())
          .filter(Boolean),
        ignorePaths: get("ignorePaths")
          .split("\n")
          .map((s) => s.trim())
          .filter(Boolean),
        kubeVersion: get("kubeVersion"),
        apiVersions: get("apiVersions")
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean),
      };
    }
    mutation.mutate({ path, body });
  }
  return (
    <>
      <div className="connection-layout">
        <Panel
          title="Connect your infrastructure"
          caption="Read-only access. Your credentials stay on your server."
        >
          <div className="segmented">
            {[
              ["repository", "Repository", GitBranch],
              ["cluster", "Cluster", Server],
              ["mapping", "Environment", Link2],
            ].map(([key, label, Icon]) => {
              const I = Icon as typeof GitBranch;
              return (
                <button
                  key={key as string}
                  className={form === key ? "active" : ""}
                  onClick={() => {
                    setForm(key as string);
                    mutation.reset();
                    setNotice("");
                  }}
                >
                  <I size={15} />
                  {label as string}
                </button>
              );
            })}
          </div>
          <form onSubmit={submit} key={form} className="connection-form">
            <label>
              {form === "mapping" ? "Environment name" : "Display name"}
              <input
                name="name"
                required
                placeholder={
                  form === "repository"
                    ? "platform-infra"
                    : form === "cluster"
                      ? "development"
                      : "development / api"
                }
              />
            </label>
            {form === "repository" ? (
              <>
                <label>
                  Repository URL or allowed local path
                  <input
                    name="url"
                    required
                    placeholder="https://github.com/team/infrastructure.git"
                  />
                </label>
                <label>
                  Branch
                  <input name="branch" placeholder="Default branch" />
                </label>
                <div className="notice">
                  Private repositories use the server’s SSH agent or Git
                  credential helper. Never put credentials in the URL. Local
                  paths must be inside DRIFTLENS_REPOSITORY_ROOT.
                </div>
              </>
            ) : form === "cluster" ? (
              <>
                <label>
                  Kubeconfig filename
                  <input
                    name="kubeconfig"
                    required
                    placeholder="development.kubeconfig"
                  />
                </label>
                <label>
                  Context (optional)
                  <input name="context" placeholder="Use current-context" />
                </label>
                <div className="notice">
                  Place a read-only kubeconfig in the server’s credential
                  directory. Enter its filename here; credentials are never sent
                  to the browser. Exec credential plugins are disabled by
                  default.
                </div>
              </>
            ) : (
              <>
                <div className="form-grid">
                  <label>
                    Repository
                    <select name="repositoryId" required>
                      <option value="">Select repository</option>
                      {repos.map((r) => (
                        <option key={r.id} value={r.id}>
                          {r.name}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label>
                    Cluster
                    <select name="clusterId" required>
                      <option value="">Select cluster</option>
                      {clusters.map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label>
                    Chart path
                    <input name="chartPath" placeholder="charts/platform" />
                  </label>
                  <label>
                    Raw manifest path (optional)
                    <input name="manifestPath" placeholder="manifests/dev" />
                  </label>
                  <label>
                    Helm release
                    <input name="release" required placeholder="platform" />
                  </label>
                  <label>
                    Namespace
                    <input
                      name="namespace"
                      required
                      placeholder="development"
                    />
                  </label>
                </div>
                <label>
                  Values files · one path per line, in merge order
                  <textarea
                    name="valuesFiles"
                    rows={3}
                    placeholder={
                      "charts/platform/values-dev.yaml\nenvironments/dev.yaml"
                    }
                  />
                </label>
                <div className="notice">
                  Helm loads the chart’s default values.yaml. Only the files
                  listed above are layered on top, from first to last.
                </div>
                <details>
                  <summary>Advanced rendering and ignore rules</summary>
                  <label>
                    Kubernetes version
                    <input name="kubeVersion" placeholder="1.34.0" />
                  </label>
                  <label>
                    API capabilities · comma-separated
                    <input
                      name="apiVersions"
                      placeholder="monitoring.coreos.com/v1"
                    />
                  </label>
                  <label>
                    Ignored JSON pointers · one per line
                    <textarea
                      name="ignorePaths"
                      placeholder="/metadata/annotations/example.com~1last-updated"
                    />
                  </label>
                </details>
              </>
            )}
            <button className="primary" disabled={mutation.isPending}>
              <Plus size={15} />
              {mutation.isPending
                ? "Saving…"
                : form === "mapping"
                  ? "Create mapping"
                  : "Add connection"}
            </button>
          </form>
          {mutation.error && (
            <div role="alert" className="alert">
              {mutation.error.message}
            </div>
          )}
          {notice && (
            <div className="success">
              <Check size={15} />
              {notice}
            </div>
          )}
        </Panel>
        <div className="stack">
          <Panel
            title="Connected sources"
            caption={`${repos.length} repositories · ${clusters.length} clusters`}
          >
            {[
              ...repos.map((r) => ({
                ...r,
                type: "repositories",
                icon: GitBranch,
                sub: r.url,
              })),
              ...clusters.map((c) => ({
                ...c,
                type: "clusters",
                icon: Server,
                sub: c.kubeconfig,
              })),
            ].map((c) => (
              <div className="connection-row" key={c.id}>
                <c.icon size={18} />
                <div>
                  <strong>{c.name}</strong>
                  <small>{c.sub}</small>
                </div>
                <button
                  disabled={mutation.isPending}
                  onClick={() =>
                    mutation.mutate({ path: `/${c.type}/${c.id}/test` })
                  }
                >
                  Test
                </button>
                <button
                  className="icon-button"
                  aria-label={`Remove ${c.name}`}
                  disabled={mutation.isPending}
                  onClick={() =>
                    mutation.mutate({
                      path: `/${c.type}/${c.id}`,
                      method: "DELETE",
                    })
                  }
                >
                  <Trash2 size={14} />
                </button>
              </div>
            ))}
            {!repos.length && !clusters.length && (
              <div className="small-empty">
                Add your first repository and cluster to get started.
              </div>
            )}
          </Panel>
          <Panel
            title="Environment mappings"
            caption="Explicit comparisons, scoped to a cluster namespace."
          >
            {mappings.map((m) => (
              <div className="mapping-card" key={m.id}>
                <div>
                  <Link2 size={16} />
                  <strong>{m.name}</strong>
                  <button
                    className="icon-button"
                    aria-label={`Remove ${m.name}`}
                    onClick={() =>
                      mutation.mutate({
                        path: `/mappings/${m.id}`,
                        method: "DELETE",
                      })
                    }
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
                <p>
                  {repos.find((r) => r.id === m.repositoryId)?.name}{" "}
                  <span>→</span>{" "}
                  {clusters.find((c) => c.id === m.clusterId)?.name}
                </p>
                <code>
                  {m.namespace} / {m.release}
                </code>
                <small>
                  {m.valuesFiles.length
                    ? m.valuesFiles.join(" → ")
                    : "Chart default values"}
                </small>
              </div>
            ))}
            {!mappings.length && (
              <div className="small-empty">
                Map a repository to a cluster to enable drift analysis.
              </div>
            )}
          </Panel>
          <Panel
            title="Scan schedule"
            caption="Snapshots retain the most recent 200 scans."
          >
            {settings.error ? (
              <div className="alert">{settings.error.message}</div>
            ) : settings.data ? (
              <form
                className="inline-form"
                onSubmit={(e) => {
                  e.preventDefault();
                  mutation.mutate({
                    path: "/settings",
                    method: "PUT",
                    body: {
                      scanInterval: Number(
                        new FormData(e.currentTarget).get("interval"),
                      ),
                    },
                  });
                }}
              >
                <label>
                  Interval in seconds
                  <input
                    name="interval"
                    type="number"
                    min={0}
                    max={86400}
                    defaultValue={settings.data.scanInterval}
                  />
                </label>
                <button disabled={mutation.isPending}>Save</button>
                <small>
                  0 = manual. Minimum periodic interval: 30 seconds.
                </small>
              </form>
            ) : (
              <div className="skeleton" />
            )}
          </Panel>
        </div>
      </div>
    </>
  );
}
