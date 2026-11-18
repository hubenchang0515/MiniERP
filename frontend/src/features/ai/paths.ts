export const aiAllocationsPath = "/ai/allocations";
export const aiMyResourcesPath = "/ai/my-resources";

export const aiPages = [
  { section: "resources", path: "/ai/providers", label: "供应商" },
  { section: "providerKeys", path: "/ai/provider-keys", label: "密钥管理" },
  { section: "models", path: "/ai/models", label: "模型映射" },
  { section: "allocations", path: aiAllocationsPath, label: "资源分配" },
  { section: "myResources", path: aiMyResourcesPath, label: "我的资源" },
] as const;

export type AISection = (typeof aiPages)[number]["section"];
