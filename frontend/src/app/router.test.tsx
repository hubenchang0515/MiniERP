import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  safeReturnTo,
  authorizedReturnTo,
  PermissionGuard,
  ProtectedGuard,
  PublicGuard,
} from "./RouteGuards";
const state = vi.hoisted(() => ({
  initialized: true,
  user: null as unknown,
  can: vi.fn(() => false),
}));
vi.mock("./AuthContext", () => ({ useAuth: () => state }));
describe("returnTo", () => {
  it.each([
    "https://evil.com",
    "//evil.com",
    "/\\evil.com",
    "/login",
    "/LOGIN",
    "/setup?x=1",
    "/%6cogin",
    "/\n/evil.com",
  ])("rejects %s", (v) => expect(safeReturnTo(v)).toBe("/dashboard"));
  it("accepts internal page with query", () =>
    expect(safeReturnTo("/users?page=2")).toBe("/users?page=2"));
  it("restores only accessible destinations", () => {
    expect(authorizedReturnTo("/users?page=2", [])).toBe("/dashboard");
    expect(authorizedReturnTo("/users/new", ["users:read"])).toBe("/dashboard");
    expect(authorizedReturnTo("/users?page=2", ["users:read"])).toBe(
      "/users?page=2",
    );
    expect(authorizedReturnTo("/account/password", [])).toBe(
      "/account/password",
    );
  });
});
describe("route permissions", () => {
  beforeEach(() => {
    state.initialized = true;
    state.user = null;
    state.can.mockReturnValue(false);
  });
  it("redirects a logged out user", () => {
    render(
      <MemoryRouter initialEntries={["/users"]}>
        <Routes>
          <Route element={<ProtectedGuard />}>
            <Route path="/users" element={<div>用户页面</div>} />
          </Route>
          <Route path="/login" element={<div>请登录</div>} />
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByText("请登录")).toBeInTheDocument();
  });
  it("allows a logged in user without permissions to enter the dashboard", () => {
    state.user = { permissions: [] };
    render(
      <MemoryRouter initialEntries={["/dashboard"]}>
        <Routes>
          <Route element={<ProtectedGuard />}>
            <Route path="/dashboard" element={<div>工作台</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByText("工作台")).toBeInTheDocument();
  });
  it("redirects an authenticated user away from a forbidden login return target", () => {
    state.user = { permissions: [] };
    render(
      <MemoryRouter initialEntries={["/login?returnTo=%2Fusers"]}>
        <Routes>
          <Route element={<PublicGuard />}>
            <Route path="/login" element={<div>请登录</div>} />
          </Route>
          <Route path="/dashboard" element={<div>工作台</div>} />
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByText("工作台")).toBeInTheDocument();
  });
  it("blocks a forbidden page", () => {
    render(
      <MemoryRouter initialEntries={["/users"]}>
        <Routes>
          <Route element={<PermissionGuard code="users:read" />}>
            <Route path="/users" element={<div>用户页面</div>} />
          </Route>
          <Route path="/403" element={<div>禁止访问</div>} />
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByText("禁止访问")).toBeInTheDocument();
  });
  it("permits a matching permission", () => {
    state.can.mockReturnValue(true);
    render(
      <MemoryRouter initialEntries={["/users"]}>
        <Routes>
          <Route element={<PermissionGuard code="users:read" />}>
            <Route path="/users" element={<div>用户页面</div>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByText("用户页面")).toBeInTheDocument();
  });
  it("directs an empty database to setup", () => {
    state.initialized = false;
    render(
      <MemoryRouter initialEntries={["/login"]}>
        <Routes>
          <Route element={<PublicGuard />}>
            <Route path="/login" element={<div>请登录</div>} />
          </Route>
          <Route path="/setup" element={<div>初始化系统</div>} />
        </Routes>
      </MemoryRouter>,
    );
    expect(screen.getByText("初始化系统")).toBeInTheDocument();
  });
});
