import { api, type Role, type Permission } from "../../lib/apiClient";
export type RoleInput = {
  name: string;
  description: string;
  permissions: string[];
};
export type RoleOption = Pick<Role, "id" | "name" | "description">;
export type RolePage<T = Role> = {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
};
export const rolesApi = {
  list: (q = "", page = 1) =>
    api<RolePage>(`/roles?q=${encodeURIComponent(q)}&page=${page}&pageSize=10`),
  options: (q = "", page = 1) =>
    api<RolePage<RoleOption>>(
      `/roles/options?q=${encodeURIComponent(q)}&page=${page}&pageSize=10`,
    ),
  get: (id: number) => api<Role>(`/roles/${id}`),
  permissions: () => api<Permission[]>("/permissions"),
  create: (input: RoleInput) => api<Role>("/roles", "POST", input),
  update: (id: number, input: RoleInput) =>
    api<Role>(`/roles/${id}`, "PATCH", input),
  remove: (id: number) => api<void>(`/roles/${id}`, "DELETE"),
};
