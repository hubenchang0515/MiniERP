import FormActions from "../../components/FormActions";
import { Dialog } from "../../components/Dialog";
import AccountActionRow from "./AccountActionRow";
import Button from "../../components/Button";
import FormSection from "../../components/FormSection";
import { Input, FormField, PasswordInput } from "../../components/FormControls";
import Alert from "../../components/Alert";
import { useEffect, useState, type FormEvent } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Save, ShieldCheck, KeyRound } from "lucide-react";
import { usersApi } from "./api";
import RolePicker from "../roles/RolePicker";
import type { User } from "../../lib/apiClient";
import { Back, PageTitle, Loading, ErrorBox } from "../../components/Shared";
import { useAuth } from "../../app/AuthContext";
export default function UserFormPage() {
  const { id } = useParams(),
    editing = !!id,
    navigate = useNavigate(),
    a = useAuth();
  const [username, setUsername] = useState(""),
    [displayName, setDisplayName] = useState(""),
    [password, setPassword] = useState(""),
    [roleIds, setRoleIds] = useState<number[]>([]),
    [user, setUser] = useState<User | null>(null),
    [loading, setLoading] = useState(true),
    [ready, setReady] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [confirm, setConfirm] = useState<"status" | "password" | null>(null);
  const isSystemAdmin = user?.roles.some((role) => role.system) ?? false;
  useEffect(() => {
    let active = true;
    setLoading(true);
    setReady(false);
    setError("");
    (editing ? usersApi.get(Number(id)) : Promise.resolve(null))
      .then((u) => {
        if (!active) return;
        setUser(u);
        if (u) {
          setUsername(u.username);
          setDisplayName(u.displayName);
          setRoleIds(u.roles.map((r) => r.id));
        }
        setReady(true);
      })
      .catch((e) => {
        if (active) setError(e.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [id, editing]);
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const input = {
        displayName,
        roleIds,
      };
      if (editing) {
        await usersApi.update(
          Number(id),
          isSystemAdmin
            ? {
                displayName,
              }
            : input,
        );
        if (a.user?.id === Number(id)) await a.reload();
      } else {
        await usersApi.create({
          ...input,
          username,
          password,
        });
      }
      navigate("/users");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function sensitive() {
    if (!user) return;
    setBusy(true);
    setError("");
    try {
      if (confirm === "status") {
        await usersApi.status(user.id, !user.enabled);
        setUser({
          ...user,
          enabled: !user.enabled,
        });
        setNotice(user.enabled ? "账号已停用" : "账号已启用");
      } else {
        await usersApi.reset(user.id, password);
        setPassword("");
        setNotice("密码已重设，原有登录凭证已失效");
        if (a.user?.id === user.id) {
          a.logout();
          return;
        }
      }
      setConfirm(null);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <Back to="/users" />
      <PageTitle title={editing ? "编辑用户" : "新建用户"} />
      <ErrorBox message={error} />
      {notice && <Alert tone="success">{notice}</Alert>}
      {loading ? (
        <Loading />
      ) : (
        ready && (
          <>
            <form
              className="grid grid-cols-2 gap-[22px] max-[900px]:grid-cols-1"
              onSubmit={submit}
            >
              <FormSection title="基本资料">
                <div className="mt-5 grid gap-[21px]">
                  <FormField
                    label="用户名"
                    required
                    hint={
                      editing
                        ? "用户名创建后不可修改"
                        : "3–32 位字母、数字、点、下划线或短横线"
                    }
                  >
                    <Input
                      required
                      pattern="[a-zA-Z0-9_.-]{3,32}"
                      minLength={3}
                      maxLength={32}
                      value={username}
                      readOnly={editing}
                      onChange={(e) => setUsername(e.target.value)}
                      autoComplete="off"
                    />
                  </FormField>
                  <FormField label="姓名">
                    <Input
                      maxLength={64}
                      value={displayName}
                      onChange={(e) => setDisplayName(e.target.value)}
                      placeholder="请输入姓名（选填）"
                    />
                  </FormField>
                  {!editing && (
                    <FormField
                      label="初始密码"
                      required
                      hint="至少 8 个字符，最多 72 字节"
                    >
                      <PasswordInput
                        minLength={8}
                        required
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        autoComplete="new-password"
                      />
                    </FormField>
                  )}
                </div>
              </FormSection>
              <FormSection
                title={
                  <>
                    <ShieldCheck size={18} />{" "}
                    {isSystemAdmin ? "账号角色" : "分配角色"}
                  </>
                }
                description={
                  isSystemAdmin
                    ? "系统管理员账号的角色不可修改。"
                    : "可选择多个角色，权限将自动合并。"
                }
              >
                <div className="mt-[17px] grid">
                  {isSystemAdmin ? (
                    <span className="text-[13px] text-ink">
                      {user?.roles.map((role) => role.name).join("、")}
                    </span>
                  ) : (
                    <RolePicker
                      value={roleIds}
                      onChange={setRoleIds}
                      initialOptions={user?.roles}
                      disabled={busy}
                    />
                  )}
                </div>
              </FormSection>
              <FormActions>
                <Button to="/users" variant="secondary">
                  取消
                </Button>
                <Button
                  disabled={busy}
                  variant="primary"
                  type="submit"
                  loading={busy}
                >
                  <Save size={16} />
                  {busy ? "正在保存…" : "保存用户"}
                </Button>
              </FormActions>
            </form>
            {editing && user && (
              <FormSection
                title={
                  <>
                    <KeyRound size={18} /> 账号安全
                  </>
                }
                className="mt-[27px]"
              >
                <AccountActionRow
                  title={<>账号状态：{user.enabled ? "已启用" : "已停用"}</>}
                  description={
                    isSystemAdmin
                      ? "系统管理员账号不可停用。"
                      : "停用后将立即撤销全部登录凭证。"
                  }
                  action={
                    (!isSystemAdmin || !user.enabled) && (
                      <Button
                        type="button"
                        disabled={busy}
                        onClick={() => {
                          setConfirm("status");
                          setError("");
                        }}
                        variant="secondary"
                      >
                        {user.enabled ? "停用账号" : "启用账号"}
                      </Button>
                    )
                  }
                />
                <AccountActionRow
                  title="重设密码"
                  description="重设后，成员需要使用新密码重新登录。"
                  action={
                    <Button
                      type="button"
                      disabled={busy}
                      onClick={() => {
                        setConfirm("password");
                        setError("");
                      }}
                      variant="secondary"
                    >
                      重设密码
                    </Button>
                  }
                />
              </FormSection>
            )}
          </>
        )
      )}
      {confirm && (
        <Dialog
          title={
            confirm === "password"
              ? "重设用户密码"
              : user?.enabled
                ? "停用账号"
                : "启用账号"
          }
          busy={busy}
          description={
            confirm === "password"
              ? "此操作将使该账号现有登录凭证失效。"
              : "请确认是否更新此成员的账号状态。"
          }
          onClose={() => setConfirm(null)}
        >
          <ErrorBox message={error} />
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void sensitive();
            }}
            className="mt-4 grid gap-5"
          >
            {confirm === "password" && (
              <FormField label="新密码">
                <PasswordInput
                  autoFocus
                  required
                  minLength={8}
                  autoComplete="new-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                />
              </FormField>
            )}
            <FormActions>
              <Button
                type="button"
                disabled={busy}
                onClick={() => setConfirm(null)}
                variant="secondary"
              >
                取消
              </Button>
              <Button
                disabled={busy}
                variant="primary"
                type="submit"
                loading={busy}
              >
                {busy ? "处理中…" : "确认操作"}
              </Button>
            </FormActions>
          </form>
        </Dialog>
      )}
    </>
  );
}
