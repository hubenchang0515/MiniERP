import { useEffect, useState, type FormEvent } from "react";
import { aiApi, type ProviderKeyDetails } from "./api";
import ResourceChoice, { type ResourceOption } from "./ResourceChoice";
import Button from "../../components/Button";
import Badge from "../../components/Badge";
import ListToolbar from "../../components/ListToolbar";
import ListPanel from "../../components/ListPanel";
import { Dialog, ConfirmDialog } from "../../components/Dialog";
import { FormField, Input, PasswordInput } from "../../components/FormControls";
import FormActions from "../../components/FormActions";
import { ErrorBox } from "../../components/Shared";
import {
  Table,
  TableHead,
  TableHeader,
  TableBody,
  TableCell,
} from "../../components/Table";

function status(key: ProviderKeyDetails) {
  if (!key.enabled) return "已停用";
  if (!key.resourceEnabled) return "供应商已停用";
  return {
    available: "可用",
    quota_exhausted: "额度耗尽",
    rate_limited: "临时限流",
    invalid: "密钥无效",
  }[key.state];
}

function KeyDialog({
  item,
  onClose,
  onSaved,
}: {
  item: ProviderKeyDetails | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [resource, setResource] = useState<ResourceOption | null>(
    item
      ? {
          id: item.resourceId,
          name: item.resourceName,
          enabled: item.resourceEnabled,
        }
      : null,
  );
  const [name, setName] = useState(item?.name ?? "");
  const [secret, setSecret] = useState("");
  const [enabled, setEnabled] = useState(item?.enabled ?? true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function save(e: FormEvent) {
    e.preventDefault();
    if (busy) return;
    if (!resource) {
      setError("请选择供应商");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await aiApi.saveProviderKey(item?.id ?? null, {
        resourceId: resource.id,
        name,
        apiKey: secret,
        enabled,
      });
      setSecret("");
      onSaved();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Dialog
      title={item ? "编辑供应商密钥" : "新增供应商密钥"}
      busy={busy}
      onClose={onClose}
    >
      <ErrorBox message={error} />
      <form onSubmit={save} className="grid gap-5">
        <FormField label="密钥名称" required>
          <Input
            required
            maxLength={128}
            disabled={busy}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </FormField>
        {item ? (
          <FormField label="供应商">
            <Input readOnly value={item.resourceName} />
          </FormField>
        ) : (
          <ResourceChoice
            value={resource}
            onChange={(r) => {
              setResource(r);
              setError("");
            }}
            disabled={busy}
          />
        )}
        <FormField
          label="外部 API KEY"
          hint={
            item
              ? "留空保留原值；填写新值会清除原不可用状态"
              : "仅加密保存在后端，不会回显"
          }
        >
          <PasswordInput
            required={!item}
            disabled={busy}
            autoComplete="new-password"
            value={secret}
            onChange={(e) => setSecret(e.target.value)}
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
          启用此密钥
        </label>
        <FormActions>
          <Button variant="secondary" disabled={busy} onClick={onClose}>
            取消
          </Button>
          <Button type="submit" loading={busy}>
            保存
          </Button>
        </FormActions>
      </form>
    </Dialog>
  );
}

export default function AIProviderKeysPanel() {
  const [text, setText] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [revision, setRevision] = useState(0);
  const [items, setItems] = useState<ProviderKeyDetails[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<ProviderKeyDetails | null | undefined>(
    undefined,
  );
  const [removing, setRemoving] = useState<ProviderKeyDetails | null>(null);
  const [busy, setBusy] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    aiApi
      .providerKeys(query, page)
      .then((r) => {
        if (active) {
          setItems(r.items);
          setTotal(r.total);
        }
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
  }, [query, page, revision]);
  async function remove() {
    if (!removing || busy) return;
    setBusy(true);
    setDeleteError("");
    try {
      await aiApi.deleteProviderKey(removing.id);
      setRemoving(null);
      if (items.length === 1 && page > 1) setPage((p) => p - 1);
      else setRevision((r) => r + 1);
    } catch (e) {
      setDeleteError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <ListToolbar
        label="搜索供应商密钥"
        placeholder="搜索密钥名称或供应商…"
        value={text}
        onChange={setText}
        disabled={busy}
        onSearch={() => {
          setQuery(text.trim());
          setPage(1);
          setRevision((r) => r + 1);
        }}
      >
        <Button onClick={() => setEditing(null)} disabled={busy}>
          新增密钥
        </Button>
      </ListToolbar>
      <ListPanel
        loading={loading}
        error={error}
        empty={!items.length}
        emptyText="暂无供应商密钥"
        page={page}
        total={total}
        disabled={busy}
        onPageChange={setPage}
      >
        <Table>
          <TableHead>
            <tr>
              <TableHeader>名称</TableHeader>
              <TableHeader>供应商</TableHeader>
              <TableHeader>状态</TableHeader>
              <TableHeader>使用次数</TableHeader>
              <TableHeader>最近使用</TableHeader>
              <TableHeader>下次试用</TableHeader>
              <TableHeader>操作</TableHeader>
            </tr>
          </TableHead>
          <TableBody>
            {items.map((key) => (
              <tr key={key.id}>
                <TableCell>{key.name}</TableCell>
                <TableCell>{key.resourceName}</TableCell>
                <TableCell>
                  <Badge>{status(key)}</Badge>
                </TableCell>
                <TableCell>{key.requests}</TableCell>
                <TableCell>
                  {key.lastUsedAt
                    ? new Date(key.lastUsedAt).toLocaleString()
                    : "—"}
                </TableCell>
                <TableCell>
                  {key.retryAt ? new Date(key.retryAt).toLocaleString() : "—"}
                </TableCell>
                <TableCell>
                  <div className="flex gap-3">
                    <Button variant="text" onClick={() => setEditing(key)}>
                      编辑
                    </Button>
                    <Button
                      variant="dangerText"
                      onClick={() => {
                        setDeleteError("");
                        setRemoving(key);
                      }}
                    >
                      删除
                    </Button>
                  </div>
                </TableCell>
              </tr>
            ))}
          </TableBody>
        </Table>
      </ListPanel>
      {error && (
        <Button variant="text" onClick={() => setRevision((r) => r + 1)}>
          重新加载
        </Button>
      )}
      {editing !== undefined && (
        <KeyDialog
          item={editing}
          onClose={() => setEditing(undefined)}
          onSaved={() => {
            setEditing(undefined);
            setRevision((r) => r + 1);
          }}
        />
      )}
      {removing && (
        <ConfirmDialog
          title="删除供应商密钥？"
          description="删除后将不再使用此密钥，已发出的请求可能继续执行。"
          busy={busy}
          error={deleteError}
          onClose={() => setRemoving(null)}
          onConfirm={() => void remove()}
        />
      )}
    </>
  );
}
