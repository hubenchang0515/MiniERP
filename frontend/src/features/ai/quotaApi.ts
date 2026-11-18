import { api } from "../../lib/apiClient";
import type { Page } from "./api";

export type AIQuotaBalance = {
  enabled: boolean;
  configured: boolean;
  week: string;
  resetsAt: string;
  weeklyPoints: number;
  addedMicros: string;
  usedMicros: string;
  reservedMicros: string;
  availableMicros: string;
  totalUsage: {
    inputTokens: string;
    outputTokens: string;
    chargedMicros: string;
  };
};

export type AIQuotaUsage = {
  requestId: string;
  model: string;
  week: string;
  status: "reserved" | "dispatched" | "actual" | "estimated" | "released";
  inputTokens: number | null;
  outputTokens: number | null;
  reservedMicros: string;
  chargedMicros: string;
  createdAt: string;
};

export type AIQuotaUser = AIQuotaBalance & {
  userId: number;
  username: string;
  displayName: string;
};

export const quotaApi = {
  mine: () => api<AIQuotaBalance>("/ai/quota"),
  usage: (page: number) =>
    api<Page<AIQuotaUsage>>(`/ai/quota/usage?page=${page}&pageSize=10`),
  settings: () => api<{ enabled: boolean }>("/ai/quota/settings"),
  setEnabled: (enabled: boolean) =>
    api<void>("/ai/quota/settings", "PATCH", { enabled }),
  setWeekly: (id: number, weeklyPoints: number) =>
    api<void>(`/ai/quota/users/${id}`, "PATCH", { weeklyPoints }),
  grant: (id: number, points: number, reason: string) =>
    api<void>(`/ai/quota/users/${id}/grants`, "POST", { points, reason }),
  users: (query: string, page: number) =>
    api<Page<AIQuotaUser>>(
      `/ai/quota/users?q=${encodeURIComponent(query)}&page=${page}&pageSize=10`,
    ),
};
