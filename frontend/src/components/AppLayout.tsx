import { NavLink, Outlet } from "react-router-dom";
import {
  LayoutDashboard,
  Users,
  ShieldCheck,
  LogOut,
  Menu,
  KeyRound,
  Cpu,
} from "lucide-react";
import { useState } from "react";
import { useAuth } from "../app/AuthContext";
import ThemeSelect from "./ThemeSelect";
import AccentSelect from "./AccentSelect";
import IconButton from "./IconButton";
import Logo from "./Logo";
export default function AppLayout() {
  const a = useAuth();
  const [open, setOpen] = useState(false);
  const accountName = a.user?.displayName || a.user?.username || "用户";
  const navClass = ({ isActive }: { isActive: boolean }) =>
    `flex items-center gap-[10px] rounded-[7px] px-3 py-[11px] text-[13px] ${isActive ? "bg-nav-active font-semibold text-nav-ink" : "text-muted hover:bg-hover"}`;
  return (
    <div className="flex min-h-screen">
      <aside
        className={`fixed inset-y-0 left-0 z-30 flex w-[200px] flex-col border-r border-line bg-surface px-[14px] pt-6 pb-4 transition-transform duration-200 min-[761px]:translate-x-0 ${open ? "translate-x-0" : "-translate-x-full"}`}
      >
        <NavLink
          className="mb-7 flex items-center gap-2 px-3 text-[21px] font-bold tracking-[-0.6px] text-accent"
          to="/dashboard"
          onClick={() => setOpen(false)}
        >
          <Logo />
          MiniOA
        </NavLink>
        <nav
          className="flex flex-col gap-[7px]"
          aria-label="主导航"
          onClick={() => setOpen(false)}
        >
          <NavLink className={navClass} to="/dashboard">
            <LayoutDashboard size={18} />
            工作台
          </NavLink>
          <NavLink className={navClass} to="/ai">
            <Cpu size={18} />
            AI 资源
          </NavLink>
          {a.can("users:read") && (
            <NavLink className={navClass} to="/users">
              <Users size={18} />
              用户管理
            </NavLink>
          )}
          {a.can("roles:read") && (
            <NavLink className={navClass} to="/roles">
              <ShieldCheck size={18} />
              角色与权限
            </NavLink>
          )}
        </nav>
        <div className="mt-auto border-t border-line pt-4">
          <div className="flex min-w-0 items-center gap-[10px] px-1">
            <span
              className="grid size-9 shrink-0 place-items-center rounded-full bg-nav-active text-[14px] font-semibold text-nav-ink"
              aria-hidden="true"
            >
              {Array.from(accountName)[0].toUpperCase()}
            </span>
            <div className="min-w-0 flex-1">
              <strong
                className="block truncate text-[13px] leading-5 font-semibold text-ink"
                title={accountName}
              >
                {accountName}
              </strong>
              {a.user?.displayName &&
                a.user.displayName !== a.user.username && (
                  <span
                    className="block truncate text-[11px] leading-[18px] text-muted"
                    title={a.user.username}
                  >
                    @{a.user.username}
                  </span>
                )}
            </div>
          </div>
          <div className="mt-3 flex items-center justify-between gap-1">
            <ThemeSelect />
            <AccentSelect />
            <IconButton
              icon={KeyRound}
              label="修改密码"
              to="/account/password"
              onClick={() => setOpen(false)}
            />
            <IconButton
              icon={LogOut}
              label="退出登录"
              onClick={() => {
                setOpen(false);
                a.logout();
              }}
            />
          </div>
        </div>
      </aside>
      {open && (
        <button
          aria-label="关闭导航"
          className="fixed inset-0 z-20 cursor-pointer bg-black/40 min-[761px]:hidden"
          onClick={() => setOpen(false)}
        />
      )}
      <div className="ml-[200px] flex min-w-0 flex-1 flex-col max-[760px]:ml-0">
        <header className="flex items-center gap-3 border-b border-line bg-surface px-[18px] py-[14px] min-[761px]:hidden">
          <button
            className="cursor-pointer"
            aria-label="打开导航"
            onClick={() => setOpen(!open)}
          >
            <Menu size={20} />
          </button>
          <span className="flex items-center gap-2 font-semibold text-accent">
            <Logo size="sm" />
            MiniOA
          </span>
        </header>
        <main className="mx-auto w-full max-w-[1500px] flex-1 px-8 pt-7 pb-10 max-[900px]:p-6 max-[760px]:px-4 max-[760px]:py-[22px]">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
