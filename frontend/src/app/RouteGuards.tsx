import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "./AuthContext";
import {
  aiAllocationsPath,
  aiMyResourcesPath,
  aiPages,
} from "../features/ai/paths";
export function safeReturnTo(value: string | null) {
  if (
    !value ||
    !value.startsWith("/") ||
    value.startsWith("//") ||
    /[\\\u0000-\u0020]/.test(value)
  )
    return "/dashboard";
  try {
    const u = new URL(value, "https://minioa.local");
    if (
      u.origin !== "https://minioa.local" ||
      /^\/(login|setup)(\/|$)/i.test(decodeURIComponent(u.pathname))
    )
      return "/dashboard";
    return u.pathname + u.search + u.hash;
  } catch {
    return "/dashboard";
  }
}
export function RootRedirect() {
  const a = useAuth();
  return (
    <Navigate
      replace
      to={!a.initialized ? "/setup" : a.user ? "/dashboard" : "/login"}
    />
  );
}
export function authorizedReturnTo(
  value: string | null,
  permissions: readonly string[],
  systemAdmin = false,
) {
  const target = safeReturnTo(value);
  const path =
    decodeURIComponent(
      new URL(target, "https://minioa.local").pathname,
    ).replace(/\/+$/, "") || "/";
  if (
    ["/", "/dashboard", "/account/password", "/ai", aiMyResourcesPath].includes(
      path,
    )
  )
    return target;
  if (path === aiAllocationsPath && !systemAdmin) return aiMyResourcesPath;
  if (aiPages.some((page) => page.path === path))
    return systemAdmin ? target : "/dashboard";
  const routes = [
    [/^\/users$/, "users:read"],
    [/^\/users\/(new|\d+\/edit)$/, "users:manage"],
    [/^\/roles$/, "roles:read"],
    [/^\/roles\/(new|\d+\/edit)$/, "roles:manage"],
  ] as const;
  return routes.some(
    ([pattern, code]) => pattern.test(path) && permissions.includes(code),
  )
    ? target
    : "/dashboard";
}
export function PublicGuard({ setup = false }: { setup?: boolean }) {
  const a = useAuth();
  const location = useLocation();
  if (!a.initialized && !setup) return <Navigate replace to="/setup" />;
  if (a.user)
    return (
      <Navigate
        replace
        to={authorizedReturnTo(
          new URLSearchParams(location.search).get("returnTo"),
          a.user.permissions,
          a.user.roles?.some((role) => role.system) ?? false,
        )}
      />
    );
  if (a.initialized && setup) return <Navigate replace to="/login" />;
  return <Outlet />;
}
export function ProtectedGuard() {
  const a = useAuth(),
    l = useLocation();
  if (!a.initialized) return <Navigate replace to="/setup" />;
  if (!a.user)
    return (
      <Navigate
        replace
        to={`/login?returnTo=${encodeURIComponent(l.pathname + l.search + l.hash)}`}
      />
    );
  return <Outlet />;
}
export function PermissionGuard({ code }: { code: string }) {
  return useAuth().can(code) ? <Outlet /> : <Navigate replace to="/403" />;
}
