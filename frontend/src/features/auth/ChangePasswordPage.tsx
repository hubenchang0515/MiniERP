import FormActions from "../../components/FormActions";
import Button from "../../components/Button";
import Card from "../../components/Card";
import { PasswordInput, FormField } from "../../components/FormControls";
import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../app/AuthContext";
import { ErrorBox, PageTitle } from "../../components/Shared";
import { authApi } from "./api";
export default function ChangePasswordPage() {
  const auth = useAuth();
  const navigate = useNavigate();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit(event: FormEvent) {
    event.preventDefault();
    setError("");
    if (newPassword !== confirmation) {
      setError("两次输入的新密码不一致");
      return;
    }
    setBusy(true);
    try {
      await authApi.changePassword({
        currentPassword,
        newPassword,
      });
      auth.logout();
      navigate("/login?passwordChanged=1", {
        replace: true,
      });
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <PageTitle title="修改密码" />
      <Card as="section" className="max-w-[560px]">
        <p className="text-[13px] leading-[1.8] text-muted">
          新密码至少 8 个字符，最多 72
          字节。修改后需要重新登录，其他设备的登录也会失效。
        </p>
        <ErrorBox message={error} />
        <form onSubmit={submit} className="mt-6 grid gap-5">
          <FormField label="当前密码">
            <PasswordInput
              autoComplete="current-password"
              required
              disabled={busy}
              value={currentPassword}
              onChange={(event) => setCurrentPassword(event.target.value)}
            />
          </FormField>
          <FormField label="新密码">
            <PasswordInput
              autoComplete="new-password"
              required
              minLength={8}
              disabled={busy}
              value={newPassword}
              onChange={(event) => setNewPassword(event.target.value)}
            />
          </FormField>
          <FormField label="确认新密码">
            <PasswordInput
              autoComplete="new-password"
              required
              minLength={8}
              disabled={busy}
              value={confirmation}
              onChange={(event) => setConfirmation(event.target.value)}
            />
          </FormField>
          <FormActions>
            <Button to="/dashboard" variant="secondary">
              返回工作台
            </Button>
            <Button
              type="submit"
              disabled={busy}
              variant="primary"
              loading={busy}
            >
              {busy ? "正在保存…" : "保存新密码"}
            </Button>
          </FormActions>
        </form>
      </Card>
    </>
  );
}
