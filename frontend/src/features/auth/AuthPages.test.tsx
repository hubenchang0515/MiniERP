import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { describe, it, expect, vi, beforeEach } from "vitest";
import AuthForm from "./AuthForm";
import { ThemeProvider } from "../../app/ThemeContext";
const auth = vi.hoisted(() => ({
  login: vi.fn(),
  setup: vi.fn(),
  reload: vi.fn(),
}));
vi.mock("../../app/AuthContext", () => ({ useAuth: () => auth }));
function mount(setup = false) {
  render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/login?returnTo=%2Fusers"]}>
        <Routes>
          <Route path="/login" element={<AuthForm setup={setup} />} />
          <Route path="/users" element={<div>用户列表</div>} />
          <Route path="/dashboard" element={<div>工作台</div>} />
        </Routes>
      </MemoryRouter>
    </ThemeProvider>,
  );
}
describe("authentication pages", () => {
  beforeEach(() => vi.clearAllMocks());
  it("submits login and restores target", async () => {
    auth.login.mockResolvedValue({ permissions: ["users:read"] });
    mount();
    fireEvent.change(screen.getByLabelText("用户名"), {
      target: { value: "admin" },
    });
    fireEvent.change(screen.getByLabelText("密码"), {
      target: { value: "password123" },
    });
    fireEvent.click(screen.getByRole("button", { name: "登录工作台" }));
    await screen.findByText("用户列表");
    expect(auth.login).toHaveBeenCalledWith({
      username: "admin",
      password: "password123",
    });
  });
  it("shows server login errors", async () => {
    auth.login.mockRejectedValue(new Error("账号或密码错误"));
    mount();
    fireEvent.change(screen.getByLabelText("用户名"), {
      target: { value: "admin" },
    });
    fireEvent.change(screen.getByLabelText("密码"), {
      target: { value: "wrong" },
    });
    fireEvent.click(screen.getByRole("button", { name: "登录工作台" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "账号或密码错误",
    );
  });
  it("creates the first administrator", async () => {
    auth.setup.mockResolvedValue({ permissions: ["users:read"] });
    mount(true);
    fireEvent.change(screen.getByLabelText("姓名"), {
      target: { value: "管理员" },
    });
    fireEvent.change(screen.getByLabelText("用户名"), {
      target: { value: "admin" },
    });
    fireEvent.change(screen.getByLabelText("密码"), {
      target: { value: "password123" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "创建管理员并进入系统" }),
    );
    await waitFor(() =>
      expect(auth.setup).toHaveBeenCalledWith({
        username: "admin",
        password: "password123",
        displayName: "管理员",
      }),
    );
  });
  it("sends a user without permissions to the dashboard instead of a forbidden return target", async () => {
    auth.login.mockResolvedValue({ permissions: [] });
    mount();
    fireEvent.change(screen.getByLabelText("用户名"), {
      target: { value: "member" },
    });
    fireEvent.change(screen.getByLabelText("密码"), {
      target: { value: "password123" },
    });
    fireEvent.click(screen.getByRole("button", { name: "登录工作台" }));
    expect(await screen.findByText("工作台")).toBeInTheDocument();
  });
});
