import Button from "../../components/Button";
import FormSection from "../../components/FormSection";
import { useAuth } from "../../app/AuthContext";
import { PageTitle } from "../../components/Shared";
import SystemResourcesPanel from "./SystemResourcesPanel";
const permissionNames: Record<string, string> = {
  "ai:use": "使用 AI",
  "users:read": "查看用户",
  "users:manage": "管理用户",
  "roles:read": "查看角色",
  "roles:manage": "管理角色",
};
export default function DashboardPage() {
  const a = useAuth(),
    u = a.user!;
  return (
    <>
      <PageTitle title="工作台" />
      {u.roles.some((role) => role.system) ? (
        <SystemResourcesPanel />
      ) : (
        <FormSection
          title="当前账号"
          className="max-w-[760px]  [&_dl]:mt-[22px] [&_dl]:grid [&_dl]:gap-4 [&_dl>div]:grid [&_dl>div]:grid-cols-[75px_1fr] [&_dl>div]:gap-4 [&_dt]:text-muted [&_dd]:[overflow-wrap:anywhere]"
        >
          <dl>
            <div>
              <dt>用户名</dt>
              <dd>{u.username}</dd>
            </div>
            {u.displayName && (
              <div>
                <dt>姓名</dt>
                <dd>{u.displayName}</dd>
              </div>
            )}
            <div>
              <dt>角色</dt>
              <dd>{u.roles.map((r) => r.name).join("、") || "未分配"}</dd>
            </div>
            <div>
              <dt>权限</dt>
              <dd>
                {u.permissions.map((p) => permissionNames[p] || p).join("、") ||
                  "暂无管理权限，请联系管理员分配角色。"}
              </dd>
            </div>
          </dl>
        </FormSection>
      )}
      {(a.can("users:read") || a.can("roles:read")) && (
        <div className="mt-5 flex gap-3">
          {a.can("users:read") && (
            <Button to="/users" variant="secondary">
              用户管理
            </Button>
          )}
          {a.can("roles:read") && (
            <Button to="/roles" variant="secondary">
              角色与权限
            </Button>
          )}
        </div>
      )}
    </>
  );
}
