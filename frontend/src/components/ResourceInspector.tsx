import { useEffect, useRef, useState } from "react";
import { X, Check, ArrowRight } from "lucide-react";
import { stringify } from "yaml";
import type { Drift, Resource } from "../types";
import {
  fieldGroup,
  fieldLabel,
  orderedDifferences,
  summary,
  valueText,
} from "../features/driftPresentation";
export function ResourceInspector({
  drift,
  resource,
  onClose,
}: {
  drift?: Drift;
  resource?: Resource;
  onClose: () => void;
}) {
  const r = resource || drift?.actual || drift?.desired;
  const [tab, setTab] = useState("changes");
  const panel = useRef<HTMLElement>(null);
  const close = useRef(onClose);
  close.current = onClose;
  useEffect(() => {
    setTab(drift ? "changes" : "yaml");
    const previous = document.activeElement as HTMLElement;
    const el = panel.current;
    el?.querySelector<HTMLButtonElement>("button")?.focus();
    const handle = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        close.current();
        return;
      }
      if (event.key === "Tab" && el) {
        const controls = Array.from(
          el.querySelectorAll<HTMLElement>(
            'button:not(:disabled),a[href],input,select,textarea,summary,[tabindex="0"]',
          ),
        );
        const first = controls[0],
          last = controls[controls.length - 1];
        if (event.shiftKey && document.activeElement === first) {
          event.preventDefault();
          last?.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault();
          first?.focus();
        }
      }
    };
    document.addEventListener("keydown", handle);
    return () => {
      document.removeEventListener("keydown", handle);
      if (previous?.isConnected) previous.focus();
    };
  }, [r?.id, drift?.id]);
  if (!r) return null;
  const differences = drift ? orderedDifferences(drift) : [];
  return (
    <div className="drawer-backdrop" onClick={onClose}>
      <aside
        ref={panel}
        className="drawer inspector"
        role="dialog"
        aria-modal="true"
        aria-labelledby="inspector-title"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="drawer-head">
          <div className="eyebrow">
            {drift?.environment || r.namespace || "CLUSTER"} / {r.kind}
          </div>
          <button aria-label="Close details" onClick={onClose}>
            <X size={20} />
          </button>
        </div>
        <h2 id="inspector-title">{r.name}</h2>
        <div className="inspector-summary">
          <span
            className={`badge ${(drift?.classification || "observed").toLowerCase().replaceAll("_", "-")}`}
          >
            {drift?.classification.replaceAll("_", " ") || "OBSERVED"}
          </span>
          <p>{drift ? summary(drift) : "Live cluster resource"}</p>
        </div>
        <div className="inspector-context">
          <span>
            Namespace <b>{r.namespace || "cluster-scoped"}</b>
          </span>
          <span>
            Last observed <b>{new Date(r.observedAt).toLocaleString()}</b>
          </span>
          {drift?.actual &&
            ["Degraded", "Failed", "Pending"].includes(drift.actual.health) && (
              <span>
                Runtime health <b>{drift.actual.health}</b>
              </span>
            )}
        </div>
        {drift?.reason && <div className="alert">{drift.reason}</div>}
        {r.kind === "Secret" && (
          <div className="notice">
            Metadata only. Secret values are never fetched or stored, and value
            drift cannot be assessed.
          </div>
        )}
        <div
          className="inspector-tabs"
          role="tablist"
          aria-label="Resource inspection"
        >
          <button
            role="tab"
            aria-selected={tab === "changes"}
            onClick={() => setTab("changes")}
          >
            Changes <span>{differences.length}</span>
          </button>
          <button
            role="tab"
            aria-selected={tab === "yaml"}
            onClick={() => setTab("yaml")}
          >
            Full YAML
          </button>
          <button
            role="tab"
            aria-selected={tab === "metadata"}
            onClick={() => setTab("metadata")}
          >
            Resource details
          </button>
        </div>
        {tab === "changes" && (
          <div role="tabpanel" aria-label="Changes">
            <div className="diff-legend">
              <span>Desired · Git</span>
              <ArrowRight size={16} />
              <span>Actual · Cluster</span>
            </div>
            {(["declared", "added", "metadata"] as const).map((group) => {
              const fields = differences.filter((x) => fieldGroup(x) === group);
              if (!fields.length) return null;
              return (
                <section key={group} className="diff-section">
                  <h3>
                    {group === "declared"
                      ? "Declared values changed"
                      : group === "added"
                        ? "Fields added in the live cluster"
                        : "Metadata changes"}{" "}
                    <span className="count">{fields.length}</span>
                  </h3>
                  {group === "added" && (
                    <p>
                      These fields were not declared in Git. They may be
                      defaults or mutations; review before deciding to ignore
                      them.
                    </p>
                  )}
                  {fields.map((d, i) => (
                    <article className="readable-diff" key={`${d.path}-${i}`}>
                      <header>
                        <strong>{fieldLabel(d.path)}</strong>
                        <code>{d.path}</code>
                      </header>
                      <div className="diff-values">
                        <div>
                          <small>Desired · Git</small>
                          <pre className="diff-minus">
                            − {valueText(d.desired)}
                          </pre>
                        </div>
                        <div>
                          <small>Actual · Cluster</small>
                          <pre className="diff-plus">
                            + {valueText(d.actual)}
                          </pre>
                        </div>
                      </div>
                    </article>
                  ))}
                </section>
              );
            })}
            {!differences.length && (
              <div className="review-empty">
                <Check size={25} />
                <h3>
                  {drift?.classification === "IN_SYNC"
                    ? "No configuration differences"
                    : drift?.classification === "MISSING"
                      ? "Resource not observed"
                      : drift?.classification === "EXTRA"
                        ? "Resource outside this Git mapping"
                        : "No field comparison available"}
                </h3>
                <p>
                  {drift?.classification === "EXTRA"
                    ? "This object exists in the cluster but is not declared by the selected environment mapping. Inspect its ownership before deciding whether it is unexpected."
                    : drift?.classification === "MISSING"
                      ? "Git declares this resource, but the scan did not find it in the cluster."
                      : "Use Full YAML or Resource details to inspect this object."}
                </p>
              </div>
            )}
          </div>
        )}
        {tab === "yaml" && (
          <div role="tabpanel" aria-label="Full YAML">
            <p className="footnote">
              Complete observed documents. Generated fields and status may
              appear here even when normalized out of the comparison.
            </p>
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
          </div>
        )}
        {tab === "metadata" && (
          <dl
            className="metadata"
            role="tabpanel"
            aria-label="Resource details"
          >
            <dt>Identity</dt>
            <dd>{r.id}</dd>
            <dt>API version</dt>
            <dd>{r.apiVersion}</dd>
            <dt>Source</dt>
            <dd>{drift?.desired?.sourceFile || "Cluster inventory"}</dd>
            <dt>Release</dt>
            <dd>{drift?.desired?.release || r.release || "Not identified"}</dd>
            <dt>Ownership</dt>
            <dd>{drift?.ownership || "Observed"}</dd>
            <dt>Labels</dt>
            <dd>
              {Object.entries(r.labels).map(([k, v]) => (
                <div key={k}>
                  <code>{k}</code> = {String(v)}
                </div>
              ))}
            </dd>
          </dl>
        )}
      </aside>
    </div>
  );
}
