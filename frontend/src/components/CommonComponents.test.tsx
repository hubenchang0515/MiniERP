import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import Button from "./Button";
import { FormField, Input, PasswordInput } from "./FormControls";
import { ConfirmDialog } from "./Dialog";

describe("shared component behavior", () => {
  it("does not submit a form unless explicitly requested and blocks loading actions", () => {
    const submit = vi.fn((event) => event.preventDefault());
    const action = vi.fn();
    render(
      <form onSubmit={submit}>
        <Button onClick={action}>普通操作</Button>
        <Button type="submit">保存</Button>
        <Button loading onClick={action}>
          等待
        </Button>
      </form>,
    );
    fireEvent.click(screen.getByRole("button", { name: "普通操作" }));
    expect(submit).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    expect(submit).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole("button", { name: "等待" }));
    expect(action).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: "等待" })).toBeDisabled();
  });
  it("preserves link semantics for navigation actions", () => {
    render(
      <MemoryRouter>
        <Button to="/users">用户</Button>
      </MemoryRouter>,
    );
    expect(screen.getByRole("link", { name: "用户" })).toHaveAttribute(
      "href",
      "/users",
    );
  });
  it("associates field labels, descriptions and errors with the control", () => {
    render(
      <FormField label="账号" hint="创建后不可修改" error="账号已存在">
        <Input readOnly value="member" />
      </FormField>,
    );
    const input = screen.getByRole("textbox", { name: "账号" });
    expect(input).toHaveAttribute("readonly");
    expect(input).toHaveAccessibleDescription("创建后不可修改 账号已存在");
    expect(input).toHaveAttribute("aria-invalid", "true");
  });
  it("reveals a password without losing its field label or value", () => {
    render(
      <FormField label="密码">
        <PasswordInput reveal defaultValue="password123" />
      </FormField>,
    );
    const input = screen.getByLabelText("密码");
    expect(input).toHaveAttribute("type", "password");
    fireEvent.click(screen.getByRole("button", { name: "显示密码" }));
    expect(input).toHaveAttribute("type", "text");
    expect(input).toHaveValue("password123");
  });
  it("traps focus, closes with Escape and restores the triggering button", () => {
    function Harness() {
      const [open, setOpen] = useState(false);
      return (
        <>
          <button onClick={() => setOpen(true)}>打开确认</button>
          {open && (
            <ConfirmDialog
              title="删除角色"
              onClose={() => setOpen(false)}
              onConfirm={() => {}}
            />
          )}
        </>
      );
    }
    const overflow = document.body.style.overflow;
    render(<Harness />);
    const trigger = screen.getByRole("button", { name: "打开确认" });
    trigger.focus();
    fireEvent.click(trigger);
    const cancel = screen.getByRole("button", { name: "取消" });
    const confirm = screen.getByRole("button", { name: "确认操作" });
    expect(cancel).toHaveFocus();
    confirm.focus();
    fireEvent.keyDown(document, { key: "Tab" });
    expect(cancel).toHaveFocus();
    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
    expect(confirm).toHaveFocus();
    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
    expect(document.body.style.overflow).toBe(overflow);
  });
  it("prevents dismissal and duplicate confirmation while submitting", () => {
    const close = vi.fn(),
      confirm = vi.fn();
    render(
      <ConfirmDialog
        title="删除角色"
        busy
        onClose={close}
        onConfirm={confirm}
      />,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    fireEvent.mouseDown(screen.getByRole("dialog").parentElement!);
    fireEvent.click(screen.getByRole("button", { name: "正在处理…" }));
    expect(close).not.toHaveBeenCalled();
    expect(confirm).not.toHaveBeenCalled();
  });
});
