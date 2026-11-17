import Button from "../../components/Button";
import Card from "../../components/Card";
import Logo from "../../components/Logo";
import { Input, FormField, PasswordInput } from "../../components/FormControls";
import Alert from "../../components/Alert";
import { useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { ArrowRight } from "lucide-react";
import { useAuth } from "../../app/AuthContext";
import { authorizedReturnTo } from "../../app/RouteGuards";
import { ErrorBox } from "../../components/Shared";
import { ApiError } from "../../lib/apiClient";
import ThemeSelect from "../../components/ThemeSelect";
import AccentSelect from "../../components/AccentSelect";
export default function AuthForm({ setup = false }: { setup?: boolean }) {
  const a = useAuth(),
    navigate = useNavigate(),
    [params] = useSearchParams();
  const [username, setUsername] = useState(""),
    [password, setPassword] = useState(""),
    [displayName, setDisplayName] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setBusy(true);
    try {
      const user = await (setup ? a.setup : a.login)({
        username,
        password,
        ...(setup
          ? {
              displayName,
            }
          : {}),
      });
      navigate(
        authorizedReturnTo(
          params.get("returnTo"),
          user.permissions,
          user.roles?.some((role) => role.system) ?? false,
        ),
        {
          replace: true,
        },
      );
    } catch (e) {
      if (setup && e instanceof ApiError && e.status === 409) {
        await a.reload();
      }
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="relative grid min-h-screen place-items-center bg-canvas px-6 pt-[72px] pb-6">
      <div className="absolute top-5 right-6 flex items-center gap-2 max-[760px]:top-[18px] max-[760px]:right-[18px]">
        <ThemeSelect />
        <AccentSelect />
      </div>
      <main className="w-full max-w-[380px]">
        <div className="mb-5 flex items-center justify-center gap-3 text-[22px] font-bold tracking-[-0.6px] text-accent">
          <Logo size="lg" />
          MiniERP
        </div>
        <Card
          padding="auth"
          className="w-full [&>h1]:mb-2 [&>p]:text-[12px] [&_form]:mt-6 [&_form]:grid [&_form]:gap-[21px]"
        >
          <h1 className="text-[24px] leading-[1.5] font-semibold tracking-[-0.8px]">
            {setup ? "创建管理员" : "登录 MiniERP"}
          </h1>
          {setup && (
            <p className="text-muted">首次使用，请创建系统管理员账号。</p>
          )}
          <form onSubmit={submit}>
            {params.get("passwordChanged") === "1" && (
              <Alert tone="success">密码已修改，请使用新密码登录。</Alert>
            )}
            <ErrorBox message={error} />
            {setup && (
              <FormField label="姓名">
                <Input
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                  maxLength={64}
                  autoComplete="name"
                  placeholder="如何称呼您（选填）"
                />
              </FormField>
            )}
            <FormField label="用户名">
              <Input
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                minLength={setup ? 3 : undefined}
                maxLength={32}
                pattern={setup ? "[a-zA-Z0-9_.-]{3,32}" : undefined}
                autoComplete="username"
                placeholder="请输入用户名"
              />
            </FormField>
            <FormField label="密码">
              <PasswordInput
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={setup ? 8 : undefined}
                autoComplete={setup ? "new-password" : "current-password"}
                placeholder={setup ? "至少 8 个字符" : "请输入密码"}
                reveal
              />
            </FormField>
            {setup && (
              <p className="text-[12px] leading-[1.8] text-muted">
                用户名为 3–32 位字母、数字、点、下划线或短横线。密码最多 72
                字节。
              </p>
            )}
            <Button
              disabled={busy}
              variant="primary"
              className="w-full"
              type="submit"
              loading={busy}
            >
              {busy
                ? "正在处理…"
                : setup
                  ? "创建管理员并进入系统"
                  : "登录工作台"}
              <ArrowRight size={17} />
            </Button>
          </form>
        </Card>
      </main>
    </div>
  );
}
