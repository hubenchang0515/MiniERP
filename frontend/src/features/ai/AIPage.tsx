import { useEffect, useState, type FormEvent } from "react";
import { useAuth } from "../../app/AuthContext";
import { aiApi, type Resource, type InternalKey } from "./api";
import Button from "../../components/Button";
import Badge from "../../components/Badge";
import Card from "../../components/Card";
import { Dialog, ConfirmDialog } from "../../components/Dialog";
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
import { aiPages, type AISection } from "./paths";

export default function AIPage({ section: tab }: { section: AISection }) {
  const auth = useAuth();
  const admin = auth.user?.roles.some((r) => r.system) ?? false;
  const canUse = auth.can("ai:use");
  const [all, setAll] = useState(admin),
    [resourceText, setResourceText] = useState(""),
    [resourceQuery, setResourceQuery] = useState(""),
    [keyText, setKeyText] = useState(""),
    [keyQuery, setKeyQuery] = useState(""),
    [page, setPage] = useState(1),
    [total, setTotal] = useState(0),
    [resources, setResources] = useState<Resource[]>([]),
    [keys, setKeys] = useState<InternalKey[]>([]),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [revision, setRevision] = useState(0);
  const [modal, setModal] = useState<"resource" | "key" | null>(null),
    [editing, setEditing] = useState<Resource | null>(null),
    [name, setName] = useState(""),
    [baseURL, setBaseURL] = useState(""),
    [enabled, setEnabled] = useState(true),
    [busy, setBusy] = useState(false),
    [formError, setFormError] = useState(""),
    [secret, setSecret] = useState(""),
    [copied, setCopied] = useState(false),
    [action, setAction] = useState<{
      key: InternalKey;
      rotate: boolean;
    } | null>(null);
  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    const load = async () => {
      if (tab === "models" || tab === "providerKeys") return;
      if (tab === "resources") {
        const r = await aiApi.resources(resourceQuery, page);
        if (active) {
          setResources(r.items);
          setTotal(r.total);
        }
      } else {
        const r = await aiApi.keys(all, page, keyQuery);
        if (active) {
          setKeys(r.items);
          setTotal(r.total);
        }
      }
    };
    load()
      .catch((e) => {
        if (active) setError(e.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [tab, all, page, revision, resourceQuery, keyQuery]);
  function openResource(r: Resource | null) {
    setEditing(r);
    setName(r?.name ?? "");
    setBaseURL(r?.baseURL ?? "");
    setEnabled(r?.enabled ?? true);
    setFormError("");
    setModal("resource");
  }
  async function save(event: FormEvent) {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setFormError("");
    try {
      if (modal === "resource")
        await aiApi.saveResource(editing?.id ?? null, {
          name,
          baseURL,
          enabled,
        });
      else {
        const r = await aiApi.createKey({ name });
        setSecret(r.secret);
        setCopied(false);
      }
      setModal(null);
      setRevision((v) => v + 1);
    } catch (e) {
      setFormError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function confirm() {
    if (!action || busy) return;
    setBusy(true);
    setFormError("");
    try {
      if (action.rotate) {
        const r = await aiApi.rotate(action.key.id);
        setSecret(r.secret);
        setCopied(false);
      } else await aiApi.revoke(action.key.id);
      setAction(null);
      setRevision((v) => v + 1);
    } catch (e) {
      setFormError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  const closeModal = () => {
    setModal(null);
  };
  return (
    <>
      <PageTitle title={admin ? "AI 资源管理" : "资源分配"} />
      {admin && (
        <nav className="mb-4 flex flex-wrap gap-2" aria-label="AI 资源子页面">
          {aiPages.map((page) => (
            <Button
              key={page.section}
              to={page.path}
              aria-current={tab === page.section ? "page" : undefined}
              variant={tab === page.section ? "primary" : "secondary"}
              onClick={() => {
                setPage(1);
                if (page.section === "keys" && tab !== "keys") setAll(true);
              }}
            >
              {page.label}
            </Button>
          ))}
        </nav>
      )}
      {!canUse && (
        <Alert tone="info">
          当前账号没有“使用 AI”权限，请联系管理员分配相应角色。已有密钥
          可撤销，但无法生成或调用。
        </Alert>
      )}
      {admin && (
        <div className="mb-4 text-[13px] text-muted">
          在角色与权限中勾选“使用 AI”，授权用户即可自行生成密钥。
          <Button to="/roles" variant="text">
            设置角色权限
          </Button>
        </div>
      )}
      <Card className="mb-4">
        <p className="text-[13px] text-muted">
          内部 API 基础地址：
          <code className="break-all text-ink">
            {window.location.origin}/ai/v1
          </code>
          。使用本人生成的密钥作为 Bearer
          凭证，可调用聊天、Responses、Embeddings 和模型列表接口。支持流式响应。
          客户端的 model 请填写内部模型 ID，可通过 GET /ai/v1/models
          查询。一个密钥可调用所有已启用供应商的可用模型，系统按内部 ID
          选择供应商并替换外部模型 ID。
        </p>
      </Card>
      {tab === "providerKeys" ? (
        <AIProviderKeysPanel />
      ) : tab === "models" ? (
        <AIModelsPanel />
      ) : (
        <>
          <ListToolbar
            label={tab === "resources" ? "搜索供应商" : "搜索密钥"}
            placeholder={
              tab === "resources"
                ? "搜索供应商名称或 API 地址…"
                : "搜索密钥名称或用户名…"
            }
            value={tab === "resources" ? resourceText : keyText}
            onChange={tab === "resources" ? setResourceText : setKeyText}
            disabled={busy}
            onSearch={() => {
              if (tab === "resources") setResourceQuery(resourceText.trim());
              else setKeyQuery(keyText.trim());
              setPage(1);
              setRevision((v) => v + 1);
            }}
          >
            {admin && tab === "keys" && (
              <label className="flex items-center gap-2 text-[13px]">
                <input
                  type="checkbox"
                  className="size-4 shrink-0 accent-accent"
                  checked={!all}
                  disabled={busy}
                  onChange={(e) => {
                    setAll(!e.target.checked);
                    setPage(1);
                  }}
                />
                只看我的
              </label>
            )}
            {((tab === "resources" && admin) || (tab === "keys" && canUse)) && (
              <Button
                disabled={busy}
                onClick={() => {
                  if (tab === "resources") openResource(null);
                  else {
                    setName("");
                    setFormError("");
                    setModal("key");
                  }
                }}
              >
                {tab === "resources" ? "新增供应商" : "生成密钥"}
              </Button>
            )}
          </ListToolbar>
          <ListPanel
            loading={loading}
            error={error}
            empty={tab === "resources" ? !resources.length : !keys.length}
            emptyText={tab === "resources" ? "暂无供应商" : "暂无密钥"}
            page={page}
            total={total}
            disabled={busy}
            onPageChange={setPage}
          >
            {tab === "resources" ? (
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
                            (key) =>
                              key.enabled && key.state === "quota_exhausted",
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
            ) : (
              <Table>
                <TableHead>
                  <tr>
                    <TableHeader>名称</TableHeader>
                    <TableHeader>用户</TableHeader>
                    <TableHeader>状态</TableHeader>
                    <TableHeader>转发次数</TableHeader>
                    <TableHeader>最近使用</TableHeader>
                    <TableHeader>操作</TableHeader>
                  </tr>
                </TableHead>
                <TableBody>
                  {keys.map((k) => (
                    <tr key={k.id}>
                      <TableCell>
                        <strong>{k.name}</strong>
                      </TableCell>
                      <TableCell>{k.username}</TableCell>
                      <TableCell>
                        <Badge>{k.enabled ? "已启用" : "已撤销"}</Badge>
                      </TableCell>
                      <TableCell>{k.requests}</TableCell>
                      <TableCell>
                        {k.lastUsedAt
                          ? new Date(k.lastUsedAt).toLocaleString()
                          : "—"}
                      </TableCell>
                      <TableCell>
                        {k.enabled && (
                          <div className="flex gap-3">
                            {canUse && k.userId === auth.user?.id && (
                              <Button
                                variant="text"
                                onClick={() => {
                                  setFormError("");
                                  setAction({ key: k, rotate: true });
                                }}
                              >
                                重新生成
                              </Button>
                            )}
                            <Button
                              variant="dangerText"
                              onClick={() => {
                                setFormError("");
                                setAction({ key: k, rotate: false });
                              }}
                            >
                              撤销
                            </Button>
                          </div>
                        )}
                      </TableCell>
                    </tr>
                  ))}
                </TableBody>
              </Table>
            )}
          </ListPanel>
          {error && (
            <Button variant="text" onClick={() => setRevision((v) => v + 1)}>
              重新加载
            </Button>
          )}
        </>
      )}
      {modal && (
        <Dialog
          title={
            modal === "resource"
              ? editing
                ? "编辑供应商"
                : "新增供应商"
              : "生成密钥"
          }
          busy={busy}
          onClose={closeModal}
        >
          <ErrorBox message={formError} />
          <form onSubmit={save} className="grid gap-5">
            <FormField label="名称" required>
              <Input
                required
                maxLength={128}
                value={name}
                disabled={busy}
                onChange={(e) => setName(e.target.value)}
              />
            </FormField>
            {modal === "resource" ? (
              <>
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
                    onChange={(e) => setBaseURL(e.target.value)}
                  />
                </FormField>
                <label className="flex items-center gap-2 text-[13px]">
                  <input
                    type="checkbox"
                    className="size-4 shrink-0 accent-accent"
                    checked={enabled}
                    disabled={busy}
                    onChange={(e) => setEnabled(e.target.checked)}
                  />
                  启用供应商
                </label>
              </>
            ) : null}
            <FormActions>
              <Button variant="secondary" disabled={busy} onClick={closeModal}>
                取消
              </Button>
              <Button type="submit" loading={busy}>
                {busy ? "正在保存…" : "保存"}
              </Button>
            </FormActions>
          </form>
        </Dialog>
      )}
      {action && (
        <ConfirmDialog
          title={action.rotate ? "重新生成密钥？" : "撤销密钥？"}
          description="原密钥将立即失效，已发出的请求可能继续执行。"
          busy={busy}
          error={formError}
          onClose={() => setAction(null)}
          onConfirm={() => void confirm()}
        />
      )}
      {secret && (
        <Dialog
          title="保存密钥"
          description="完整密钥仅显示这一次，请保存到安全的位置。关闭后无法再次查看，可重新生成。"
          onClose={() => setSecret("")}
        >
          <Alert tone="info">请勿将密钥发到聊天群或提交到代码仓库。</Alert>
          <Input aria-label="新生成的密钥" readOnly value={secret} />
          <FormActions className="mt-5">
            <Button
              variant="secondary"
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(secret);
                  setCopied(true);
                } catch {
                  setCopied(false);
                }
              }}
            >
              {copied ? "已复制" : "复制"}
            </Button>
            <Button onClick={() => setSecret("")}>我已保存</Button>
          </FormActions>
        </Dialog>
      )}
    </>
  );
}
