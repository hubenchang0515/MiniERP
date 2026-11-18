import { useEffect, useState, type FormEvent } from "react";
import { useAuth } from "../../app/AuthContext";
import { aiApi, type Resource } from "./api";
import Button from "../../components/Button";
import Badge from "../../components/Badge";
import Card from "../../components/Card";
import { Dialog } from "../../components/Dialog";
import FormActions from "../../components/FormActions";
import { FormField, Input } from "../../components/FormControls";
import ListPanel from "../../components/ListPanel";
import ListToolbar from "../../components/ListToolbar";
import Alert from "../../components/Alert";
import { PageTitle, ErrorBox } from "../../components/Shared";
import {
  Table,
  TableHead,
  TableHeader,
  TableBody,
  TableCell,
} from "../../components/Table";

import AIModelsPanel from "./AIModelsPanel";
import AIProviderKeysPanel from "./AIProviderKeysPanel";
import {
  AIMyQuotaPanel,
  AIMyUsagePanel,
  AIQuotaAllocationPanel,
} from "./AIQuotaPanel";
import AIUserKeysPanel from "./AIUserKeysPanel";
import { aiPages, type AISection } from "./paths";

export default function AIPage({ section }: { section: AISection }) {
  const auth = useAuth();
  const admin = auth.user?.roles.some((role) => role.system) ?? false;
  const title =
    section === "allocations"
      ? "资源分配"
      : section === "myResources"
        ? "我的资源"
        : "AI 资源管理";
  return (
    <>
      <PageTitle title={title} />
      {admin && (
        <nav className="mb-4 flex flex-wrap gap-2" aria-label="AI 资源子页面">
          {aiPages.map((page) => (
            <Button
              key={page.section}
              to={page.path}
              aria-current={section === page.section ? "page" : undefined}
              variant={section === page.section ? "primary" : "secondary"}
            >
              {page.label}
            </Button>
          ))}
        </nav>
      )}
      {section === "allocations" && <AIQuotaAllocationPanel />}
      {section === "myResources" && (
        <>
          {!auth.can("ai:use") && (
            <Alert tone="info">
              当前账号没有“使用
              AI”权限，请联系管理员分配相应角色。已有密钥可撤销，但无法生成或调用。
            </Alert>
          )}
          <Card className="mb-4">
            <p className="text-[13px] text-muted">
              内部 API 基础地址：
              <code className="break-all text-ink">
                {window.location.origin}/ai/v1
              </code>
              。使用本人生成的密钥作为 Bearer
              凭证，可调用聊天、Responses、Embeddings
              和模型列表接口。支持流式响应。 客户端的 model 请填写内部模型
              ID，可通过 GET /ai/v1/models 查询。
            </p>
          </Card>
          <AIMyQuotaPanel />
          <AIUserKeysPanel />
          <AIMyUsagePanel />
        </>
      )}
      {section === "providerKeys" && <AIProviderKeysPanel />}
      {section === "models" && <AIModelsPanel />}
      <AISuppliersPanel active={section === "resources"} />
    </>
  );
}

function AISuppliersPanel({ active }: { active: boolean }) {
  const [text, setText] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [resources, setResources] = useState<Resource[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [revision, setRevision] = useState(0);
  const [editing, setEditing] = useState<Resource | null | undefined>(
    undefined,
  );
  const [name, setName] = useState("");
  const [baseURL, setBaseURL] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState("");

  useEffect(() => {
    if (!active) return;
    let current = true;
    setLoading(true);
    setError("");
    aiApi
      .resources(query, page)
      .then((result) => {
        if (current) {
          setResources(result.items);
          setTotal(result.total);
        }
      })
      .catch((e: Error) => {
        if (current) setError(e.message);
      })
      .finally(() => {
        if (current) setLoading(false);
      });
    return () => {
      current = false;
    };
  }, [active, page, query, revision]);

  function openResource(resource: Resource | null) {
    setEditing(resource);
    setName(resource?.name ?? "");
    setBaseURL(resource?.baseURL ?? "");
    setEnabled(resource?.enabled ?? true);
    setFormError("");
  }

  async function save(event: FormEvent) {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setFormError("");
    try {
      await aiApi.saveResource(editing?.id ?? null, { name, baseURL, enabled });
      setEditing(undefined);
      setRevision((value) => value + 1);
    } catch (e) {
      setFormError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  if (!active) return null;
  return (
    <>
      <ListToolbar
        label="搜索供应商"
        placeholder="搜索供应商名称或 API 地址…"
        value={text}
        onChange={setText}
        disabled={busy}
        onSearch={() => {
          setQuery(text.trim());
          setPage(1);
          setRevision((value) => value + 1);
        }}
      >
        <Button disabled={busy} onClick={() => openResource(null)}>
          新增供应商
        </Button>
      </ListToolbar>
      <ListPanel
        loading={loading}
        error={error}
        empty={!resources.length}
        emptyText="暂无供应商"
        page={page}
        total={total}
        disabled={busy}
        onPageChange={setPage}
      >
        <Table>
          <TableHead>
            <tr>
              <TableHeader>供应商</TableHeader>
              <TableHeader>API 基础地址</TableHeader>
              <TableHeader>供应商密钥</TableHeader>
              <TableHeader>状态</TableHeader>
              <TableHeader>操作</TableHeader>
            </tr>
          </TableHead>
          <TableBody>
            {resources.map((r) => (
              <tr key={r.id}>
                <TableCell>{r.name}</TableCell>
                <TableCell>{r.baseURL}</TableCell>
                <TableCell>
                  <div className="grid gap-1">
                    <span>
                      {r.keys?.filter(
                        (key) => key.enabled && key.state === "available",
                      ).length ?? 0}{" "}
                      可用 / {r.keys?.length ?? 0} 总数
                    </span>
                    {r.keys?.some(
                      (key) => key.enabled && key.state === "quota_exhausted",
                    ) && (
                      <span className="text-[12px] text-muted">
                        部分密钥额度耗尽，等待恢复
                      </span>
                    )}
                  </div>
                </TableCell>
                <TableCell>
                  <Badge>{r.enabled ? "已启用" : "已停用"}</Badge>
                </TableCell>
                <TableCell>
                  <Button variant="text" onClick={() => openResource(r)}>
                    编辑
                  </Button>
                </TableCell>
              </tr>
            ))}
          </TableBody>
        </Table>
      </ListPanel>
      {error && (
        <Button
          variant="text"
          onClick={() => setRevision((value) => value + 1)}
        >
          重新加载
        </Button>
      )}
      {editing !== undefined && (
        <Dialog
          title={editing ? "编辑供应商" : "新增供应商"}
          busy={busy}
          onClose={() => setEditing(undefined)}
        >
          <ErrorBox message={formError} />
          <form onSubmit={save} className="grid gap-5">
            <FormField label="名称" required>
              <Input
                required
                maxLength={128}
                value={name}
                disabled={busy}
                onChange={(event) => setName(event.target.value)}
              />
            </FormField>
            <FormField
              label="API 基础地址"
              hint="填写包含版本路径的地址，例如 https://api.example.com/v1"
              required
            >
              <Input
                required
                type="url"
                value={baseURL}
                disabled={busy}
                onChange={(event) => setBaseURL(event.target.value)}
              />
            </FormField>
            <label className="flex items-center gap-2 text-[13px]">
              <input
                type="checkbox"
                className="size-4 shrink-0 accent-accent"
                checked={enabled}
                disabled={busy}
                onChange={(event) => setEnabled(event.target.checked)}
              />
              启用供应商
            </label>
            <FormActions>
              <Button
                variant="secondary"
                disabled={busy}
                onClick={() => setEditing(undefined)}
              >
                取消
              </Button>
              <Button type="submit" loading={busy}>
                保存
              </Button>
            </FormActions>
          </form>
        </Dialog>
      )}
    </>
  );
}
