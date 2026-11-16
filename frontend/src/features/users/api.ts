import { api, type User } from "../../lib/apiClient";
export type UserInput = {
  username: string;
  displayName: string;
  password?: string;
  roleIds: number[];
};
export const usersApi = {
  list: (q = "", page = 1) =>
    api<{ items: User[]; total: number; page: number; pageSize: number }>(
      `/users?q=${encodeURIComponent(q)}&page=${page}&pageSize=10`,
    ),
  get: (id: number) => api<User>(`/users/${id}`),
  create: (input: UserInput) => api<User>("/users", "POST", input),
  update: (
    id: number,
    input: Omit<UserInput, "username" | "password" | "roleIds"> & {
      roleIds?: number[];
    },
  ) => api<User>(`/users/${id}`, "PATCH", input),
  status: (id: number, enabled: boolean) =>
    api<void>(`/users/${id}/status`, "PATCH", { enabled }),
  reset: (id: number, password: string) =>
    api<void>(`/users/${id}/password-reset`, "POST", { password }),
};
