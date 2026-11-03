import { expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DriftWorkspace } from "./DriftWorkspace";
import type { Drift } from "../types";
const make = (
  name: string,
  environment = "dev",
  path = "/spec/replicas",
  desired: unknown = 3,
): Drift => ({
  id: name,
  mappingId: "m",
  repositoryId: "r",
  clusterId: "c",
  environment,
  name,
  kind: "Deployment",
  namespace: environment,
  classification: "MODIFIED",
  ownership: "mapped",
  differences: [{ path, desired, actual: 1, classification: "MODIFIED" }],
});
it("starts with declared changes, supports environment selection and keeps live-added results accessible", () => {
  const select = vi.fn();
  render(
    <DriftWorkspace
      rows={[
        make("dev-api"),
        make("staging-api", "staging"),
        make("defaulted", "dev", "/spec/default", null),
      ]}
      repos={[]}
      clusters={[]}
      mapped
      onConnect={() => {}}
      onSelect={select}
    />,
  );
  expect(screen.getByText("dev-api")).toBeInTheDocument();
  expect(screen.queryByText("defaulted")).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: /^dev / }));
  expect(screen.queryByText("staging-api")).not.toBeInTheDocument();
  fireEvent.click(screen.getByText("dev-api"));
  expect(select).toHaveBeenCalledWith(
    expect.objectContaining({ name: "dev-api" }),
  );
  fireEvent.click(screen.getByRole("button", { name: /Live-added fields/ }));
  expect(screen.getByText("defaulted")).toBeInTheDocument();
});
it("paginates long queues instead of rendering every row", () => {
  render(
    <DriftWorkspace
      rows={Array.from({ length: 25 }, (_, i) =>
        make(`api-${String(i).padStart(2, "0")}`),
      )}
      repos={[]}
      clusters={[]}
      mapped
      onConnect={() => {}}
      onSelect={() => {}}
    />,
  );
  expect(screen.queryByText("api-24")).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Next" }));
  expect(screen.getByText("api-24")).toBeInTheDocument();
});
