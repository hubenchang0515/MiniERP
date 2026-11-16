import { api } from "../../lib/apiClient";

export type Usage = { percent: number; error?: string };
export type SystemResources = {
  cpu: Usage;
  memory: Usage & { total: number; used: number; available: number };
  disk: Usage & { path: string; total: number; used: number; free: number };
  sampledAt: string;
};

export const dashboardApi = {
  resources: () => api<SystemResources>("/system/resources"),
};
