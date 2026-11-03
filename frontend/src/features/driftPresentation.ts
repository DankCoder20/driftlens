import type { Drift } from "../types";
export type Difference = Drift["differences"][number];
export type ReviewGroup =
  | "declared"
  | "added"
  | "metadata"
  | "extra"
  | "unknown"
  | "sync";
export const reviewGroups: {
  id: ReviewGroup;
  label: string;
  description: string;
}[] = [
  {
    id: "declared",
    label: "Declared changes",
    description:
      "A value declared in Git differs, or a declared resource is missing. Review these first.",
  },
  {
    id: "added",
    label: "Live-added fields",
    description:
      "Non-metadata fields exist in the cluster but are absent from Git. These may be defaults or controller mutations; they are not automatically harmless.",
  },
  {
    id: "metadata",
    label: "Metadata only",
    description:
      "Only labels, annotations, or other metadata differ. Operational metadata can still matter.",
  },
  {
    id: "extra",
    label: "Outside this mapping",
    description:
      "Standalone resources observed in the namespace but not declared by this mapping. They are not automatically errors.",
  },
  {
    id: "unknown",
    label: "Unable to compare",
    description:
      "A failed observation prevents a reliable comparison. Check the scan error before making decisions.",
  },
  {
    id: "sync",
    label: "In sync",
    description: "No differences were found by the comparison engine.",
  },
];
export function isMetadata(d: Difference) {
  return /(^|\/)metadata(\/|$)/.test(d.path);
}
export function fieldGroup(d: Difference) {
  return isMetadata(d)
    ? "metadata"
    : d.desired == null && d.actual != null
      ? "added"
      : "declared";
}
export function reviewGroup(d: Drift): ReviewGroup {
  if (d.classification === "MISSING") return "declared";
  if (d.classification === "EXTRA") return "extra";
  if (d.classification === "UNKNOWN") return "unknown";
  if (d.classification === "IN_SYNC") return "sync";
  if (d.differences.some((x) => fieldGroup(x) === "declared"))
    return "declared";
  if (d.differences.some((x) => fieldGroup(x) === "added")) return "added";
  return "metadata";
}
export function fieldLabel(path: string) {
  if (path.endsWith("/image")) return "Container image";
  if (path.endsWith("/replicas")) return "Replica count";
  if (path.includes("/resources/")) return "CPU / memory configuration";
  if (path.includes("/env")) return "Environment configuration";
  if (path.startsWith("/data/")) return "Configuration value";
  if (path.includes("/annotations")) return "Annotations";
  if (path.includes("/labels")) return "Labels";
  return path
    .split("/")
    .filter(Boolean)
    .slice(-2)
    .join(" / ")
    .replaceAll("~1", "/")
    .replaceAll("~0", "~");
}
export function orderedDifferences(d: Drift) {
  const rank = { declared: 0, added: 1, metadata: 2 };
  return [...d.differences].sort(
    (a, b) =>
      rank[fieldGroup(a)] - rank[fieldGroup(b)] || a.path.localeCompare(b.path),
  );
}
export function summary(d: Drift) {
  if (d.classification === "MISSING")
    return "Declared in Git · not found in cluster";
  if (d.classification === "EXTRA")
    return "Found in cluster · not in this mapping";
  if (d.classification === "UNKNOWN")
    return d.reason || "Observation unavailable";
  if (d.classification === "IN_SYNC")
    return "Matches the normalized Git configuration";
  const fields = orderedDifferences(d);
  const primary =
    reviewGroup(d) === "declared"
      ? fields.filter((x) => fieldGroup(x) === "declared")
      : fields;
  const labels = [...new Set(primary.map((x) => fieldLabel(x.path)))];
  return (
    labels.slice(0, 2).join(" · ") +
    (labels.length > 2 ? ` +${labels.length - 2} more` : "")
  );
}
export function priority(d: Drift) {
  if (d.classification === "MISSING") return 0;
  const g = reviewGroup(d);
  if (
    g === "declared" &&
    d.differences.some(
      (x) => x.path.endsWith("/image") || x.path.endsWith("/replicas"),
    )
  )
    return 1;
  return 2 + reviewGroups.findIndex((x) => x.id === g);
}
export function valueText(v: unknown) {
  return v == null
    ? "Not present"
    : typeof v === "string"
      ? v
      : JSON.stringify(v);
}
