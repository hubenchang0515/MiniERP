import FormActions from "../../components/FormActions";
import Button from "../../components/Button";
import Alert from "../../components/Alert";
import FormSection from "../../components/FormSection";
import {
  Input,
  FormField,
  Textarea,
  CheckboxItem,
} from "../../components/FormControls";
import { useEffect, useState, type FormEvent } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Save, ShieldCheck } from "lucide-react";
import { rolesApi } from "./api";
import type { Permission } from "../../lib/apiClient";
import { Back, PageTitle, Loading, ErrorBox } from "../../components/Shared";
import { useAuth } from "../../app/AuthContext";
export default function RoleFormPage() {
  const { id } = useParams(),
    editing = !!id,
    navigate = useNavigate(),
    a = useAuth();
  const [name, setName] = useState(""),
    [description, setDescription] = useState(""),
    [permissions, setPermissions] = useState<string[]>([]),
    [options, setOptions] = useState<Permission[]>([]),
    [system, setSystem] = useState(false),
    [loading, setLoading] = useState(true),
    [ready, setReady] = useState(false),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    setLoading(true);
    setReady(false);
    Promise.all([
      rolesApi.permissions(),
      editing ? rolesApi.get(Number(id)) : Promise.resolve(null),
    ])
      .then(([ps, r]) => {
        if (!active) return;
        setOptions(ps);
        if (r) {
          setName(r.name);
          setDescription(r.description);
          setPermissions(r.permissions.map((p) => p.code));
          setSystem(r.system);
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
        name,
        description,
        permissions,
      };
      await (editing
        ? rolesApi.update(Number(id), input)
        : rolesApi.create(input));
      await a.reload();
      navigate("/roles");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <Back to="/roles" />
      <PageTitle title={editing ? "编辑角色" : "新建角色"} />
      <ErrorBox message={error} />
      {loading ? (
        <Loading />
      ) : (
        ready && (
          <form
            className="grid grid-cols-2 gap-[22px] max-[900px]:grid-cols-1"
            onSubmit={submit}
          >
            <FormSection title="角色信息">
              <div className="mt-5 grid gap-[21px]">
                <FormField label="角色名称" required>
                  <Input
                    required
                    maxLength={64}
                    value={name}
                    disabled={system}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="例如：人事专员"
                  />
                </FormField>
                <FormField label="角色说明">
                  <Textarea
                    maxLength={500}
                    rows={4}
                    value={description}
                    disabled={system}
                    onChange={(e) => setDescription(e.target.value)}
                    placeholder="描述该角色的职责与用途"
                  />
                </FormField>
              </div>
            </FormSection>
            <FormSection
              title={
                <>
                  <ShieldCheck size={18} /> 权限配置
                </>
              }
              description="管理权限包含对应的查看权限。"
            >
              <div className="mt-[17px] grid">
                {options.map((p) => (
                  <CheckboxItem
                    disabled={system}
                    checked={permissions.includes(p.code)}
                    onChange={(e) =>
                      setPermissions(
                        e.target.checked
                          ? [...permissions, p.code]
                          : permissions.filter((v) => v !== p.code),
                      )
                    }
                    key={p.code}
                    title={p.name}
                    description={p.description}
                  />
                ))}
              </div>
            </FormSection>
            {system && (
              <Alert tone="info">系统管理员角色受保护，无法修改。</Alert>
            )}
            <FormActions>
              <Button to="/roles" variant="secondary">
                取消
              </Button>
              <Button disabled={busy || system} variant="primary" type="submit">
                <Save size={16} />
                {busy ? "正在保存…" : "保存角色"}
              </Button>
            </FormActions>
          </form>
        )
      )}
    </>
  );
}
