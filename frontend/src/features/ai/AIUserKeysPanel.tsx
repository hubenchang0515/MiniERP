import { useEffect, useState, type FormEvent } from "react";
import { useAuth } from "../../app/AuthContext";
import Button from "../../components/Button";
import Badge from "../../components/Badge";
import { Dialog, ConfirmDialog } from "../../components/Dialog";
import FormActions from "../../components/FormActions";
import { FormField, Input } from "../../components/FormControls";
import ListPanel from "../../components/ListPanel";
import ListToolbar from "../../components/ListToolbar";
import Alert from "../../components/Alert";
import { ErrorBox } from "../../components/Shared";
import {
  Table,
  TableHead,
  TableHeader,
  TableBody,
  TableCell,
} from "../../components/Table";
import { aiApi, type InternalKey } from "./api";

export default function AIUserKeysPanel() {
  const auth = useAuth();
  const canManageOwn = auth.can("ai:use");
  const [text, setText] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [keys, setKeys] = useState<InternalKey[]>([]);
  const [capacity, setCapacity] = useState<{ count: number; limit: number }>();
  const atLimit = capacity !== undefined && capacity.count >= capacity.limit;
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [revision, setRevision] = useState(0);
  const [creating, setCreating] = useState(false);
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState("");
  const [secret, setSecret] = useState("");
  const [copied, setCopied] = useState(false);
  const [action, setAction] = useState<{
    key: InternalKey;
    rotate: boolean;
  } | null>(null);

  useEffect(() => {
    let current = true;
    setLoading(true);
    setError("");
    aiApi
      .keys(false, page, query)
      .then((result) => {
        if (current) {
          setKeys(result.items);
          setTotal(result.total);
          if (result.activeCount !== undefined && result.limit !== undefined) {
            setCapacity({ count: result.activeCount, limit: result.limit });
          }
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
  }, [page, query, revision]);

  function showSecret(value: string) {
    setSecret(value);
    setCopied(false);
  }

  async function save(event: FormEvent) {
    event.preventDefault();
    if (busy || !canManageOwn) return;
    setBusy(true);
    setFormError("");
    try {
      showSecret((await aiApi.createKey({ name })).secret);
      setCreating(false);
      setRevision((value) => value + 1);
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
      if (action.rotate) showSecret((await aiApi.rotate(action.key.id)).secret);
      else await aiApi.revoke(action.key.id);
      setAction(null);
      setRevision((value) => value + 1);
    } catch (e) {
      setFormError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <ListToolbar
        label="搜索密钥"
        placeholder="搜索密钥名称…"
        value={text}
        onChange={setText}
        disabled={busy}
        onSearch={() => {
          setQuery(text.trim());
          setPage(1);
          setRevision((value) => value + 1);
        }}
      >
        {canManageOwn && (
          <div className="flex items-center gap-3">
            {capacity && (
              <span className="text-[12px] text-muted">
                有效密钥 {capacity.count} / {capacity.limit}
              </span>
            )}
            <Button
              disabled={busy || atLimit}
              title={atLimit ? "请先撤销一个有效密钥" : undefined}
              onClick={() => {
                setName("");
                setFormError("");
                setCreating(true);
              }}
            >
              生成密钥
            </Button>
          </div>
        )}
      </ListToolbar>
      <ListPanel
        loading={loading}
        error={error}
        empty={!keys.length}
        emptyText="暂无密钥"
        page={page}
        total={total}
        disabled={busy}
        onPageChange={setPage}
      >
        <Table>
          <TableHead>
            <tr>
              <TableHeader>名称</TableHeader>
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
                <TableCell>
                  <Badge>{k.enabled ? "已启用" : "已撤销"}</Badge>
                </TableCell>
                <TableCell>{k.requests}</TableCell>
                <TableCell>
                  {k.lastUsedAt ? new Date(k.lastUsedAt).toLocaleString() : "—"}
                </TableCell>
                <TableCell>
                  {k.enabled && (
                    <div className="flex gap-3">
                      {canManageOwn && k.userId === auth.user?.id && (
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
      </ListPanel>
      {error && (
        <Button
          variant="text"
          onClick={() => setRevision((value) => value + 1)}
        >
          重新加载
        </Button>
      )}
      {creating && (
        <Dialog title="生成密钥" busy={busy} onClose={() => setCreating(false)}>
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
            <FormActions>
              <Button
                variant="secondary"
                disabled={busy}
                onClick={() => setCreating(false)}
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
