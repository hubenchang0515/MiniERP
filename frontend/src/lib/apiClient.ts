import { authStorage } from "./authStorage";
export type Permission = { code: string; name: string; description: string };
export type Role = {
  id: number;
  name: string;
  description: string;
  system: boolean;
  permissions: Permission[];
  userCount: number;
};
export type User = {
  id: number;
  username: string;
  displayName: string;
  enabled: boolean;
  roles: Role[];
  createdAt: string;
};
export type CurrentUser = User & { permissions: string[] };
export type Session = { token: string; user: CurrentUser };
export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}
export async function api<T>(
  path: string,
  method = "GET",
  data?: unknown,
): Promise<T> {
  const token = authStorage.get();
  let response: Response;
  try {
    response = await fetch(`/api${path}`, {
      method,
      headers: {
        ...(data !== undefined ? { "Content-Type": "application/json" } : {}),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: data === undefined ? undefined : JSON.stringify(data),
    });
  } catch {
    throw new Error("无法连接服务器，请检查网络后重试");
  }
  if (!response.ok) {
    const error = await response
      .json()
      .catch(() => ({ message: "请求失败，请稍后重试" }));
    if (response.status === 401 && path !== "/auth/login") {
      authStorage.clear();
      window.dispatchEvent(new Event("auth-expired"));
    }
    throw new ApiError(response.status, error.message);
  }
  return response.status === 204 ? (undefined as T) : response.json();
}
