import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { MemoryRouter } from "react-router-dom";
import { expect, it, vi } from "vitest";
import RolePicker from "./RolePicker";
import { rolesApi } from "./api";
vi.mock("./api", () => ({ rolesApi: { options: vi.fn() } }));
it("preserves selected roles across pages and searches without submitting the parent form", async () => {
  const save = vi.fn();
  const first = { id: 1, name: "阅读者", description: "" };
  const second = { id: 2, name: "编辑者", description: "" };
  const finance = { id: 3, name: "财务", description: "" };
  vi.mocked(rolesApi.options).mockImplementation(async (q = "", page = 1) => ({
    items: q ? [finance] : page === 1 ? [first] : [second],
    total: q ? 1 : 11,
    page,
    pageSize: 10,
  }));
  function Harness() {
    const [ids, setIds] = useState([99]);
    return (
      <form
        onSubmit={(event) => {
          event.preventDefault();
          save(ids);
        }}
      >
        <RolePicker
          value={ids}
          onChange={setIds}
          initialOptions={[{ id: 99, name: "已有角色", description: "" }]}
        />
        <button type="submit">保存分配</button>
      </form>
    );
  }
  render(
    <MemoryRouter>
      <Harness />
    </MemoryRouter>,
  );
  fireEvent.click(await screen.findByRole("checkbox", { name: "阅读者" }));
  fireEvent.click(screen.getByRole("button", { name: "下一页" }));
  fireEvent.click(await screen.findByRole("checkbox", { name: "编辑者" }));
  fireEvent.change(screen.getByLabelText("搜索可分配角色"), {
    target: { value: "finance" },
  });
  expect(
    fireEvent.keyDown(screen.getByLabelText("搜索可分配角色"), {
      key: "Enter",
    }),
  ).toBe(false);
  fireEvent.click(await screen.findByRole("checkbox", { name: "财务" }));
  await waitFor(() =>
    expect(rolesApi.options).toHaveBeenLastCalledWith("finance", 1),
  );
  expect(
    screen.getByRole("button", { name: "移除角色已有角色" }),
  ).toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: "移除角色阅读者" }),
  ).toBeInTheDocument();
  expect(save).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "保存分配" }));
  expect(save).toHaveBeenLastCalledWith([99, 1, 2, 3]);
  fireEvent.click(screen.getByRole("button", { name: "移除角色阅读者" }));
  fireEvent.click(screen.getByRole("button", { name: "保存分配" }));
  expect(save).toHaveBeenLastCalledWith([99, 2, 3]);
});
