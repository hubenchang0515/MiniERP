import { api } from "../../lib/apiClient";
export type Page<T> = {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
};
export type Resource = {
  id: number;
  name: string;
  baseURL: string;
  enabled: boolean;
  keys: ProviderKey[];
};
export type ProviderKey = {
  id: number;
  name: string;
  enabled: boolean;
  state: "available" | "quota_exhausted" | "rate_limited" | "invalid";
  retryAt: string | null;
  requests: number;
  lastUsedAt: string | null;
};
export type ProviderKeyDetails = ProviderKey & {
  resourceId: number;
  resourceName: string;
  resourceEnabled: boolean;
};
export type ModelMapping = {
  id: number;
  internalId: string;
  externalId: string;
  resourceId: number;
  resourceName: string;
  resourceEnabled: boolean;
  enabled: boolean;
  pricing?: ModelPricing;
};
export type ModelPricing = {
  inputPerMillion: number;
  outputPerMillion: number;
  reservePoints: number;
  skipStreamUsage: boolean;
};
export type InternalKey = {
  id: number;
  name: string;
  userId: number;
  username: string;
  enabled: boolean;
  requests: number;
  lastUsedAt: string | null;
};
export const aiApi = {
  upstreamModels: (input: {
    resourceId?: number;
    baseURL?: string;
    apiKey?: string;
  }) => api<{ modelIds: string[] }>("/ai/resources/models", "POST", input),
  resourceOptions: (q = "", page = 1) =>
    api<Page<Pick<Resource, "id" | "name" | "enabled">>>(
      `/ai/resources/options?q=${encodeURIComponent(q)}&page=${page}&pageSize=10`,
    ),
  resources: (q = "", page = 1) =>
    api<Page<Resource>>(
      `/ai/resources?q=${encodeURIComponent(q)}&page=${page}&pageSize=10`,
    ),
  saveResource: (id: number | null, input: Omit<Resource, "id" | "keys">) =>
    api<Resource>(
      `/ai/resources${id ? `/${id}` : ""}`,
      id ? "PATCH" : "POST",
      input,
    ),
  keys: (all = false, page = 1, q = "") =>
    api<Page<InternalKey> & { activeCount?: number; limit?: number }>(
      `/ai/keys?all=${all}&page=${page}&pageSize=10&q=${encodeURIComponent(q)}`,
    ),
  providerKeys: (q = "", page = 1) =>
    api<Page<ProviderKeyDetails>>(
      `/ai/provider-keys?q=${encodeURIComponent(q)}&page=${page}&pageSize=10`,
    ),
  saveProviderKey: (
    id: number | null,
    input: {
      resourceId: number;
      name: string;
      apiKey: string;
      enabled: boolean;
    },
  ) =>
    api<ProviderKey>(
      `/ai/provider-keys${id ? `/${id}` : ""}`,
      id ? "PATCH" : "POST",
      input,
    ),
  deleteProviderKey: (id: number) =>
    api<void>(`/ai/provider-keys/${id}`, "DELETE"),
  models: (q = "", page = 1) =>
    api<Page<ModelMapping>>(
      `/ai/models?q=${encodeURIComponent(q)}&page=${page}&pageSize=10`,
    ),
  saveModel: (
    id: number | null,
    input: Pick<
      ModelMapping,
      "internalId" | "externalId" | "resourceId" | "enabled" | "pricing"
    >,
  ) =>
    api<ModelMapping>(
      `/ai/models${id ? `/${id}` : ""}`,
      id ? "PATCH" : "POST",
      input,
    ),
  deleteModel: (id: number) => api<void>(`/ai/models/${id}`, "DELETE"),
  createKey: (input: { name: string }) =>
    api<{ secret: string }>("/ai/keys", "POST", input),
  rotate: (id: number) =>
    api<{ secret: string }>(`/ai/keys/${id}/rotate`, "POST"),
  revoke: (id: number) => api<void>(`/ai/keys/${id}`, "DELETE"),
};
