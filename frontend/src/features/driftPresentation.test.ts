import { expect, it } from "vitest";
import { reviewGroup, summary } from "./driftPresentation";
import type { Drift } from "../types";
const row = (path: string, desired: unknown, actual: unknown) =>
  ({
    classification: "MODIFIED",
    differences: [{ path, desired, actual, classification: "MODIFIED" }],
  }) as Drift;
it("separates declared changes from live-added fields without hiding either", () => {
  expect(reviewGroup(row("/spec/replicas", 3, 1))).toBe("declared");
  expect(reviewGroup(row("/spec/defaulted", null, "value"))).toBe("added");
  expect(reviewGroup(row("/metadata/labels/team", "a", "b"))).toBe("metadata");
  expect(
    reviewGroup({
      ...row("/metadata/labels/team", "a", "b"),
      classification: "MISSING",
    }),
  ).toBe("declared");
});
it("does not mistake a missing declared value or a false value for a live default", () => {
  expect(reviewGroup(row("/spec/enabled", false, true))).toBe("declared");
  expect(reviewGroup(row("/spec/replicas", 0, 1))).toBe("declared");
  expect(reviewGroup(row("/spec/setting", "declared", null))).toBe("declared");
});
it("uses readable field labels and preserves operational metadata review", () => {
  expect(
    summary(row("/spec/template/spec/containers[name=api]/image", "a", "b")),
  ).toBe("Container image");
  expect(
    reviewGroup(row("/spec/template/metadata/annotations/x", null, "injected")),
  ).toBe("metadata");
});
