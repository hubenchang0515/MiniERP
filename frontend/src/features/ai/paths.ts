export const aiPages = [
  { section: "resources", path: "/ai/providers", label: "供应商" },
  { section: "providerKeys", path: "/ai/provider-keys", label: "密钥管理" },
  { section: "models", path: "/ai/models", label: "模型映射" },
  { section: "keys", path: "/ai/allocations", label: "资源分配" },
] as const;

export type AISection = (typeof aiPages)[number]["section"];
