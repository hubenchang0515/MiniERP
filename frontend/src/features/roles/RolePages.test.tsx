import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect, vi, beforeEach } from "vitest";
import RolesPage from "./RolesPage";
import RoleFormPage from "./RoleFormPage";
import { rolesApi } from "./api";
vi.mock("./api", () => ({
  rolesApi: {
    list: vi.fn(),
    permissions: vi.fn(),
    create: vi.fn(),
    remove: vi.fn(),
  },
}));
vi.mock("../../app/AuthContext", () => ({
  useAuth: () => ({ can: () => true, reload: async () => {} }),
}));
describe("roles", () => {
  beforeEach(() => vi.clearAllMocks());
  it("protects system roles and assigned roles", async () => {
    vi.mocked(rolesApi.list).mockResolvedValue({
      items: [
        {
          id: 1,
          name: "系统管理员",
          description: "所有权限",
          system: true,
          permissions: [],
          userCount: 1,
        },
        {
          id: 2,
          name: "阅读者",
          description: "",
          system: false,
          permissions: [],
          userCount: 1,
        },
      ],
      total: 2,
      page: 1,
      pageSize: 10,
    });
    render(
      <MemoryRouter>
        <RolesPage />
      </MemoryRouter>,
    );
    await screen.findByText("系统管理员");
    expect(
      screen.queryByRole("button", { name: "删除系统管理员" }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "删除阅读者" })).toBeDisabled();
  });
  it("sends selected permission codes", async () => {
    vi.mocked(rolesApi.permissions).mockResolvedValue([
      { code: "users:read", name: "查看用户", description: "查看成员资料" },
    ]);
    vi.mocked(rolesApi.create).mockResolvedValue({} as never);
    render(
      <MemoryRouter>
        <RoleFormPage />
      </MemoryRouter>,
    );
    await screen.findByText("查看用户");
    fireEvent.change(screen.getByLabelText(/角色名称/), {
      target: { value: "阅读者" },
    });
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(screen.getByRole("button", { name: "保存角色" }));
    await waitFor(() =>
      expect(rolesApi.create).toHaveBeenCalledWith({
        name: "阅读者",
        description: "",
        permissions: ["users:read"],
      }),
    );
  });
  it("paginates and resets to page one when searching", async () => {
    vi.mocked(rolesApi.list).mockResolvedValue({
      items: [],
      total: 21,
      page: 1,
      pageSize: 10,
    });
    render(
      <MemoryRouter>
        <RolesPage />
      </MemoryRouter>,
    );
    await screen.findByText("共 21 条 · 第 1 / 3 页");
    fireEvent.click(screen.getByRole("button", { name: "下一页" }));
    await waitFor(() => expect(rolesApi.list).toHaveBeenLastCalledWith("", 2));
    await screen.findByText("共 21 条 · 第 2 / 3 页");
    fireEvent.change(screen.getByLabelText("搜索角色"), {
      target: { value: "reader" },
    });
    fireEvent.click(screen.getByRole("button", { name: "搜索" }));
    await waitFor(() =>
      expect(rolesApi.list).toHaveBeenLastCalledWith("reader", 1),
    );
  });
  it("returns to the previous page after deleting the last row", async () => {
    const role = {
      id: 2,
      name: "阅读者",
      description: "",
      system: false,
      permissions: [],
      userCount: 0,
    };
    vi.mocked(rolesApi.list).mockResolvedValue({
      items: [role],
      total: 11,
      page: 1,
      pageSize: 10,
    });
    vi.mocked(rolesApi.remove).mockResolvedValue(undefined);
    render(
      <MemoryRouter>
        <RolesPage />
      </MemoryRouter>,
    );
    await screen.findByText("阅读者");
    fireEvent.click(screen.getByRole("button", { name: "下一页" }));
    await screen.findByText("共 11 条 · 第 2 / 2 页");
    fireEvent.click(screen.getByRole("button", { name: "删除阅读者" }));
    fireEvent.click(screen.getByRole("button", { name: "确认删除" }));
    await waitFor(() => expect(rolesApi.remove).toHaveBeenCalledWith(2));
    await waitFor(() => expect(rolesApi.list).toHaveBeenLastCalledWith("", 1));
  });
});
