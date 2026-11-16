import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, it, expect, vi, beforeEach } from "vitest";
import UsersPage from "./UsersPage";
import UserFormPage from "./UserFormPage";
import { usersApi } from "./api";
import { rolesApi } from "../roles/api";
import type { Role, User } from "../../lib/apiClient";
vi.mock("./api", () => ({
  usersApi: { list: vi.fn(), create: vi.fn(), get: vi.fn(), update: vi.fn() },
}));
vi.mock("../roles/api", () => ({ rolesApi: { options: vi.fn() } }));
vi.mock("../../app/AuthContext", () => ({
  useAuth: () => ({ can: () => true, user: { id: 1 }, reload: vi.fn() }),
}));

const systemRole: Role = {
  id: 1,
  name: "系统管理员",
  description: "内置角色",
  system: true,
  permissions: [],
  userCount: 1,
};
const ordinaryRole: Role = {
  id: 2,
  name: "阅读者",
  description: "查看用户",
  system: false,
  permissions: [],
  userCount: 0,
};
const renderEditor = (user: User) => {
  vi.mocked(rolesApi.options).mockResolvedValue({
    items: [ordinaryRole],
    total: 1,
    page: 1,
    pageSize: 10,
  });
  vi.mocked(usersApi.get).mockResolvedValue(user);
  vi.mocked(usersApi.update).mockResolvedValue(user);
  return render(
    <MemoryRouter initialEntries={[`/users/${user.id}/edit`]}>
      <Routes>
        <Route path="/users/:id/edit" element={<UserFormPage />} />
        <Route path="/users" element={<div>用户列表</div>} />
      </Routes>
    </MemoryRouter>,
  );
};
describe("user management", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });
  it("searches and paginates via API", async () => {
    vi.mocked(usersApi.list).mockResolvedValue({
      items: [
        {
          id: 1,
          username: "alice",
          displayName: "张三",
          enabled: true,
          roles: [],
          createdAt: "2026-11-15",
        },
      ],
      total: 21,
      page: 1,
      pageSize: 10,
    });
    render(
      <MemoryRouter>
        <UsersPage />
      </MemoryRouter>,
    );
    await screen.findByText("张三");
    fireEvent.click(screen.getByRole("button", { name: "下一页" }));
    await waitFor(() => expect(usersApi.list).toHaveBeenLastCalledWith("", 2));
    fireEvent.change(screen.getByLabelText("搜索用户"), {
      target: { value: "alice" },
    });
    fireEvent.click(screen.getByRole("button", { name: "搜索" }));
    await waitFor(() =>
      expect(usersApi.list).toHaveBeenLastCalledWith("alice", 1),
    );
  });
  it("submits user details and selected roles", async () => {
    vi.mocked(rolesApi.options).mockResolvedValue({
      items: [
        {
          id: 2,
          name: "阅读者",
          description: "查看用户",
        },
      ],
      total: 1,
      page: 1,
      pageSize: 10,
    });
    vi.mocked(usersApi.create).mockResolvedValue({} as never);
    render(
      <MemoryRouter>
        <UserFormPage />
      </MemoryRouter>,
    );
    await screen.findByText("阅读者");
    expect(
      screen.queryByRole("checkbox", { name: /系统管理员/ }),
    ).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText(/用户名/), {
      target: { value: "alice" },
    });
    fireEvent.change(screen.getByLabelText("姓名"), {
      target: { value: "张三" },
    });
    fireEvent.change(screen.getByLabelText(/初始密码/), {
      target: { value: "password123" },
    });
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: "保存用户" }));
    await waitFor(() =>
      expect(usersApi.create).toHaveBeenCalledWith({
        username: "alice",
        displayName: "张三",
        password: "password123",
        roleIds: [2],
      }),
    );
  });
  it("keeps admin roles fixed while allowing profile updates", async () => {
    renderEditor({
      id: 1,
      username: "admin",
      displayName: "管理员",
      enabled: true,
      roles: [systemRole],
      createdAt: "2026-11-15",
    });
    await screen.findByText("系统管理员账号的角色不可修改。");
    expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "停用账号" }),
    ).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("姓名"), {
      target: { value: "新姓名" },
    });
    fireEvent.click(screen.getByRole("button", { name: "保存用户" }));
    await waitFor(() =>
      expect(usersApi.update).toHaveBeenCalledWith(1, {
        displayName: "新姓名",
      }),
    );
  });
  it("offers only ordinary roles for other accounts", async () => {
    renderEditor({
      id: 2,
      username: "member",
      displayName: "成员",
      enabled: true,
      roles: [ordinaryRole],
      createdAt: "2026-11-15",
    });
    const checkbox = await screen.findByRole("checkbox", { name: /阅读者/ });
    expect(
      screen.queryByRole("checkbox", { name: /系统管理员/ }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "停用账号" }),
    ).toBeInTheDocument();
    fireEvent.click(checkbox);
    fireEvent.click(screen.getByRole("button", { name: "保存用户" }));
    await waitFor(() =>
      expect(usersApi.update).toHaveBeenCalledWith(2, {
        displayName: "成员",
        roleIds: [],
      }),
    );
  });
});
