import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ChangePasswordPage from "./ChangePasswordPage";
import { authApi } from "./api";

const auth = vi.hoisted(() => ({ logout: vi.fn() }));
vi.mock("../../app/AuthContext", () => ({ useAuth: () => auth }));
vi.mock("./api", () => ({ authApi: { changePassword: vi.fn() } }));
function mount() {
  render(
    <MemoryRouter initialEntries={["/account/password"]}>
      <Routes>
        <Route path="/account/password" element={<ChangePasswordPage />} />
        <Route path="/login" element={<div>重新登录</div>} />
      </Routes>
    </MemoryRouter>,
  );
}
function fill(confirmation = "changed123") {
  fireEvent.change(screen.getByLabelText("当前密码"), {
    target: { value: "password123" },
  });
  fireEvent.change(screen.getByLabelText("新密码"), {
    target: { value: "changed123" },
  });
  fireEvent.change(screen.getByLabelText("确认新密码"), {
    target: { value: confirmation },
  });
  fireEvent.click(screen.getByRole("button", { name: "保存新密码" }));
}
describe("change own password", () => {
  beforeEach(() => vi.clearAllMocks());
  it("rejects mismatched confirmation before sending a request", () => {
    mount();
    fill("different123");
    expect(screen.getByRole("alert")).toHaveTextContent(
      "两次输入的新密码不一致",
    );
    expect(authApi.changePassword).not.toHaveBeenCalled();
  });
  it("keeps the current login when the old password is incorrect", async () => {
    vi.mocked(authApi.changePassword).mockRejectedValue(
      new Error("当前密码不正确"),
    );
    mount();
    fill();
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "当前密码不正确",
    );
    expect(auth.logout).not.toHaveBeenCalled();
  });
  it("changes only the current account password and requires login again", async () => {
    vi.mocked(authApi.changePassword).mockResolvedValue(undefined);
    mount();
    fill();
    expect(await screen.findByText("重新登录")).toBeInTheDocument();
    expect(authApi.changePassword).toHaveBeenCalledWith({
      currentPassword: "password123",
      newPassword: "changed123",
    });
    expect(auth.logout).toHaveBeenCalledOnce();
  });
});
