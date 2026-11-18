import { api, type CurrentUser, type Session } from "../../lib/apiClient";
export type Credentials = {
  username: string;
  password: string;
};
export const authApi = {
  status: () => api<{ initialized: boolean }>("/auth/status"),
  me: () => api<CurrentUser>("/auth/me"),
  changePassword: (input: { currentPassword: string; newPassword: string }) =>
    api<void>("/auth/password", "POST", input),
  login: (input: Credentials) => api<Session>("/auth/login", "POST", input),
  bootstrap: (input: Credentials) =>
    api<Session>("/auth/bootstrap", "POST", input),
};
