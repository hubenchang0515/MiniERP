import Button from "../components/Button";
import ResultState from "../components/ResultState";
import { Route, Routes } from "react-router-dom";
import { ShieldX, FileQuestion } from "lucide-react";
import {
  RootRedirect,
  PublicGuard,
  ProtectedGuard,
  PermissionGuard,
} from "./RouteGuards";
import AppLayout from "../components/AppLayout";
import SetupPage from "../features/auth/SetupPage";
import LoginPage from "../features/auth/LoginPage";
import ChangePasswordPage from "../features/auth/ChangePasswordPage";
import DashboardPage from "../features/dashboard/DashboardPage";
import UsersPage from "../features/users/UsersPage";
import UserFormPage from "../features/users/UserFormPage";
import RolesPage from "../features/roles/RolesPage";
import RoleFormPage from "../features/roles/RoleFormPage";
import { aiRoutes } from "../features/ai/routes";
function StatusPage({ forbidden = false }: { forbidden?: boolean }) {
  return (
    <ResultState
      icon={forbidden ? ShieldX : FileQuestion}
      title={forbidden ? "暂时没有访问权限" : "页面未找到"}
      description={
        forbidden
          ? "请联系系统管理员为您分配相应角色。"
          : "这个地址可能有误，或页面已经移动。"
      }
      actions={
        <Button to="/" variant="primary">
          返回工作台
        </Button>
      }
    />
  );
}
export default function AppRoutes() {
  return (
    <Routes>
      <Route path="/" element={<RootRedirect />} />
      <Route element={<PublicGuard setup />}>
        <Route path="/setup" element={<SetupPage />} />
      </Route>
      <Route element={<PublicGuard />}>
        <Route path="/login" element={<LoginPage />} />
      </Route>
      <Route element={<ProtectedGuard />}>
        <Route element={<AppLayout />}>
          <Route path="/dashboard" element={<DashboardPage />} />
          {aiRoutes}
          <Route path="/account/password" element={<ChangePasswordPage />} />
          <Route element={<PermissionGuard code="users:read" />}>
            <Route path="/users" element={<UsersPage />} />
          </Route>
          <Route element={<PermissionGuard code="users:manage" />}>
            <Route path="/users/new" element={<UserFormPage />} />
            <Route path="/users/:id/edit" element={<UserFormPage />} />
          </Route>
          <Route element={<PermissionGuard code="roles:read" />}>
            <Route path="/roles" element={<RolesPage />} />
          </Route>
          <Route element={<PermissionGuard code="roles:manage" />}>
            <Route path="/roles/new" element={<RoleFormPage />} />
            <Route path="/roles/:id/edit" element={<RoleFormPage />} />
          </Route>
          <Route path="/403" element={<StatusPage forbidden />} />
        </Route>
      </Route>
      <Route path="*" element={<StatusPage />} />
    </Routes>
  );
}
