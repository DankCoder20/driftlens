import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { Detail, Empty, ResourceTable } from "./common";
import type { Drift, Resource } from "../types";
const resource: Resource = {
  id: "apps/Deployment/dev/api",
  group: "apps",
  apiVersion: "apps/v1",
  kind: "Deployment",
  namespace: "dev",
  name: "api",
  labels: { app: "api" },
  annotations: {},
  clusterId: "cluster",
  observedAt: "2026-01-01T00:00:00Z",
  health: "Degraded",
  desiredReplicas: 3,
  readyReplicas: 1,
  object: { spec: { replicas: 1 } },
};
const drift: Drift = {
  id: "mapping|" + resource.id,
  mappingId: "mapping",
  repositoryId: "repo",
  environment: "dev",
  clusterId: "cluster",
  kind: "Deployment",
  namespace: "dev",
  name: "api",
  classification: "MODIFIED",
  ownership: "mapped",
  desired: { ...resource, object: { spec: { replicas: 3 } } },
  actual: resource,
  differences: [
    {
      path: "/spec/replicas",
      desired: 3,
      actual: 1,
      classification: "MODIFIED",
    },
  ],
};
describe("resource experience", () => {
  it("opens a real resource from keyboard or click", () => {
    const select = vi.fn();
    render(<ResourceTable rows={[drift]} onSelect={select} />);
    fireEvent.click(screen.getByText("api"));
    expect(select).toHaveBeenCalledWith(drift);
    expect(screen.getByText("MODIFIED")).toBeInTheDocument();
  });
  it("shows exact replica changes and both YAML documents", () => {
    render(<Detail drift={drift} onClose={() => {}} />);
    expect(screen.getByText("/spec/replicas")).toBeInTheDocument();
    expect(screen.getByText("− 3")).toBeInTheDocument();
    expect(screen.getByText("+ 1")).toBeInTheDocument();
    expect(screen.getAllByText("Desired · Git").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Actual · Cluster").length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole("tab", { name: "Full YAML" }));
    expect(
      screen.getByRole("tabpanel", { name: "Full YAML" }),
    ).toHaveTextContent("replicas: 3");
    expect(
      screen.getByRole("tabpanel", { name: "Full YAML" }),
    ).toHaveTextContent("replicas: 1");
  });
  it("provides actionable onboarding", () => {
    const action = vi.fn();
    render(
      <Empty title="Connect your infrastructure" action={action}>
        No invented inventory.
      </Empty>,
    );
    fireEvent.click(
      screen.getByRole("button", { name: /Configure connections/ }),
    );
    expect(action).toHaveBeenCalledOnce();
  });
});
