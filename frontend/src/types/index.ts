export type Resource = {
  id: string;
  group: string;
  apiVersion: string;
  kind: string;
  namespace: string;
  name: string;
  labels: Record<string, unknown>;
  annotations: Record<string, unknown>;
  repositoryId?: string;
  sourceFile?: string;
  release?: string;
  clusterId: string;
  observedAt: string;
  health: string;
  desiredReplicas: number;
  readyReplicas: number;
  object: Record<string, unknown>;
};
export type Repository = {
  id: string;
  name: string;
  url: string;
  branch: string;
};
export type Cluster = {
  id: string;
  name: string;
  kubeconfig: string;
  context: string;
};
export type Mapping = {
  id: string;
  name: string;
  repositoryId: string;
  clusterId: string;
  chartPath: string;
  manifestPath?: string;
  release: string;
  namespace: string;
  valuesFiles: string[];
  kubeVersion?: string;
  apiVersions?: string[];
  ignorePaths: string[];
};
export type Drift = {
  id: string;
  mappingId: string;
  repositoryId: string;
  environment: string;
  clusterId: string;
  kind: string;
  namespace: string;
  name: string;
  classification: string;
  ownership: string;
  reason?: string;
  desired?: Resource;
  actual?: Resource;
  differences: {
    path: string;
    desired: unknown;
    actual: unknown;
    classification: string;
  }[];
};
export type Discovery = {
  commit: string;
  files: string[];
  charts: {
    path: string;
    name: string;
    version: string;
    description: string;
    valuesFiles: string[];
  }[];
};
export type Summary = {
  total: number;
  classifications: Record<string, number>;
  kinds: Record<string, number>;
  namespaces: Record<string, number>;
  clusters: Record<string, number>;
  health: Record<string, number>;
  images: Record<string, number>;
  desiredReplicas: number;
  readyReplicas: number;
  unhealthy: number;
};
export type Change = {
  type: string;
  resource: string;
  before?: string;
  after?: string;
};
export type Snapshot = {
  id: string;
  createdAt: string;
  status: string;
  resources: Resource[];
  drift: Drift[];
  summary: Summary;
  clusters: Record<
    string,
    {
      version: string;
      nodeCount: number;
      namespaces: string[];
      status: string;
      failures: Record<string, string>;
    }
  >;
  repositories: Record<string, Discovery>;
  changes: Change[];
  errors: Record<string, string>;
};
export type History = Omit<Snapshot, "resources" | "drift" | "clusters">;
