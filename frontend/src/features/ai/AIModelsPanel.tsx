import { useCallback, useEffect, useId, useState, type FormEvent } from "react";
import { aiApi, type ModelMapping } from "./api";
import ResourceChoice, { type ResourceOption } from "./ResourceChoice";
import Autocomplete from "../../components/Autocomplete";
import Button from "../../components/Button";
import Badge from "../../components/Badge";
import ListToolbar from "../../components/ListToolbar";
import ListPanel from "../../components/ListPanel";
import { Dialog, ConfirmDialog } from "../../components/Dialog";
import { FormField, Input } from "../../components/FormControls";
import FormActions from "../../components/FormActions";
import { ErrorBox } from "../../components/Shared";
import {
  Table,
  TableHead,
  TableHeader,
  TableBody,
  TableCell,
} from "../../components/Table";

function MappingDialog({
  mapping,
  onClose,
  onSaved,
}: {
  mapping: ModelMapping | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const streamUsageHintId = useId();
  const [internalId, setInternalId] = useState(mapping?.internalId ?? "");
  const [externalId, setExternalId] = useState(mapping?.externalId ?? "");
  const [enabled, setEnabled] = useState(mapping?.enabled ?? true);
  const [inputPrice, setInputPrice] = useState(
    String(mapping?.pricing?.inputPerMillion ?? 0),
  );
  const [outputPrice, setOutputPrice] = useState(
    String(mapping?.pricing?.outputPerMillion ?? 0),
  );
  const [reservePoints, setReservePoints] = useState(
    String(mapping?.pricing?.reservePoints ?? 0),
  );
  const [skipStreamUsage, setSkipStreamUsage] = useState(
    mapping?.pricing?.skipStreamUsage ?? false,
  );
  const [resource, setResource] = useState<ResourceOption | null>(
    mapping
      ? {
          id: mapping.resourceId,
          name: mapping.resourceName,
          enabled: mapping.resourceEnabled,
        }
      : null,
  );
  const [suggestions, setSuggestions] = useState<string[] | null>(null);
  const [fetching, setFetching] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [fetchError, setFetchError] = useState("");
  const loadModelOptions = useCallback(
    async (query: string) => {
      const search = query.toLowerCase();
      return {
        items: (suggestions ?? []).filter((id) =>
          id.toLowerCase().includes(search),
        ),
        hasMore: false,
      };
    },
    [suggestions],
  );
  async function fetchModels() {
    if (!resource || busy || fetching) return;
    setFetching(true);
    setFetchError("");
    setSuggestions(null);
    try {
      setSuggestions(
        (await aiApi.upstreamModels({ resourceId: resource.id })).modelIds,
      );
    } catch (e) {
      setFetchError((e as Error).message);
    } finally {
      setFetching(false);
    }
  }
  async function save(e: FormEvent) {
    e.preventDefault();
    if (busy || fetching) return;
    if (!resource) {
      setError("请选择供应商");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await aiApi.saveModel(mapping?.id ?? null, {
        internalId,
        externalId,
        resourceId: resource.id,
        enabled,
        pricing: {
          inputPerMillion: Number(inputPrice),
          outputPerMillion: Number(outputPrice),
          reservePoints: Number(reservePoints),
          skipStreamUsage,
        },
      });
      onSaved();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Dialog
      title={mapping ? "编辑模型映射" : "新增模型映射"}
      description="内部模型 ID 在整个系统中唯一。系统据此选择供应商，再替换为对应的外部模型 ID。"
      busy={busy || fetching}
      onClose={onClose}
    >
      <ErrorBox message={error} />
      <form onSubmit={save} className="grid gap-5">
        <FormField label="内部模型 ID" required>
          <Input
            required
            maxLength={256}
            disabled={busy}
            value={internalId}
            onChange={(e) => setInternalId(e.target.value)}
            placeholder="例如 chat-default"
          />
        </FormField>
        <ResourceChoice
          value={resource}
          disabled={busy || fetching}
          onChange={(r) => {
            setResource(r);
            setError("");
            setSuggestions(null);
            setFetchError("");
          }}
        />
        <Button
          variant="secondary"
          disabled={busy || !resource}
          loading={fetching}
          onClick={() => void fetchModels()}
        >
          获取上游模型
        </Button>
        <ErrorBox message={fetchError} />
        {suggestions !== null && (
          <p role="status" className="text-[13px] text-muted">
            {suggestions.length
              ? `已获取 ${suggestions.length} 个模型，可在外部模型 ID 输入框中选择，也可手动填写。`
              : "上游模型列表为空，可手动填写外部模型 ID。"}
          </p>
        )}
        <Autocomplete
          label="外部模型 ID"
          required
          maxLength={256}
          disabled={busy}
          value={externalId}
          onChange={(id) => setExternalId(id ?? "")}
          createOption={(text) => text}
          loadOptions={suggestions?.length ? loadModelOptions : undefined}
          getOptionKey={(id) => id}
          getOptionLabel={(id) => id}
          emptyMessage="没有匹配的上游模型，可手动填写"
        />
        <label className="flex items-center gap-2 text-[13px]">
          <input
            type="checkbox"
            className="size-4 shrink-0 accent-accent"
            checked={enabled}
            disabled={busy}
            onChange={(e) => setEnabled(e.target.checked)}
          />
          启用映射
        </label>
        <fieldset
          disabled={busy || fetching}
          className="grid gap-3 rounded-md border border-line p-3"
        >
          <legend className="px-1 text-[13px] font-medium">积分价格</legend>
          <div className="grid grid-cols-2 gap-3">
            <FormField label="每百万输入 Token（积分）" required>
              <Input
                type="number"
                min={0}
                max={1_000_000_000}
                step={1}
                required
                value={inputPrice}
                onChange={(e) => setInputPrice(e.target.value)}
              />
            </FormField>
            <FormField label="每百万输出 Token（积分）" required>
              <Input
                type="number"
                min={0}
                max={1_000_000_000}
                step={1}
                required
                value={outputPrice}
                onChange={(e) => setOutputPrice(e.target.value)}
              />
            </FormField>
          </div>
          <FormField
            label="单次预留（积分）"
            hint="设置单价时至少预留 1 积分。三项均填 0 表示未定价。"
            required
          >
            <Input
              type="number"
              min={Number(inputPrice) > 0 || Number(outputPrice) > 0 ? 1 : 0}
              max={1_000_000_000}
              step={1}
              required
              value={reservePoints}
              onChange={(e) => setReservePoints(e.target.value)}
            />
          </FormField>
          <label className="flex items-center gap-2 text-[13px]">
            <input
              type="checkbox"
              className="size-4 shrink-0 accent-accent"
              aria-describedby={streamUsageHintId}
              checked={!skipStreamUsage}
              onChange={(e) => setSkipStreamUsage(!e.target.checked)}
            />
            主动请求聊天流式用量
          </label>
          <p
            id={streamUsageHintId}
            className="pl-6 text-[12px] leading-relaxed text-muted"
          >
            若供应商不支持该参数，可以取消勾选。
          </p>
        </fieldset>
        <FormActions>
          <Button
            variant="secondary"
            disabled={busy || fetching}
            onClick={onClose}
          >
            取消
          </Button>
          <Button type="submit" loading={busy} disabled={fetching}>
            保存
          </Button>
        </FormActions>
      </form>
    </Dialog>
  );
}

export default function AIModelsPanel() {
  const [text, setText] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [revision, setRevision] = useState(0);
  const [items, setItems] = useState<ModelMapping[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState<ModelMapping | null | undefined>(
    undefined,
  );
  const [removing, setRemoving] = useState<ModelMapping | null>(null);
  const [busy, setBusy] = useState(false);
  const [deleteError, setDeleteError] = useState("");
  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    aiApi
      .models(query, page)
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
      await aiApi.deleteModel(removing.id);
      setRemoving(null);
      if (items.length === 1 && page > 1) setPage((p) => p - 1);
      else setRevision((v) => v + 1);
    } catch (e) {
      setDeleteError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <ListToolbar
        label="搜索模型映射"
        placeholder="搜索内部 ID、外部 ID 或供应商…"
        value={text}
        onChange={setText}
        disabled={busy}
        onSearch={() => {
          setQuery(text.trim());
          setPage(1);
          setRevision((v) => v + 1);
        }}
      >
        <Button disabled={busy} onClick={() => setEditing(null)}>
          新增映射
        </Button>
      </ListToolbar>
      <ListPanel
        loading={loading}
        error={error}
        empty={!items.length}
        emptyText="暂无模型映射"
        page={page}
        total={total}
        disabled={busy}
        onPageChange={setPage}
      >
        <Table>
          <TableHead>
            <tr>
              <TableHeader>内部模型 ID</TableHeader>
              <TableHeader>供应商</TableHeader>
              <TableHeader>外部模型 ID</TableHeader>
              <TableHeader>积分价格</TableHeader>
              <TableHeader>状态</TableHeader>
              <TableHeader>操作</TableHeader>
            </tr>
          </TableHead>
          <TableBody>
            {items.map((m) => (
              <tr key={m.id}>
                <TableCell>
                  <code>{m.internalId}</code>
                </TableCell>
                <TableCell>{m.resourceName}</TableCell>
                <TableCell>
                  <code>{m.externalId}</code>
                </TableCell>
                <TableCell>
                  {m.pricing && m.pricing.reservePoints > 0
                    ? "已定价"
                    : "未定价"}
                </TableCell>
                <TableCell>
                  <Badge>
                    {!m.enabled
                      ? "已停用"
                      : !m.resourceEnabled
                        ? "供应商已停用"
                        : "已启用"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex gap-3">
                    <Button variant="text" onClick={() => setEditing(m)}>
                      编辑
                    </Button>
                    <Button
                      variant="dangerText"
                      onClick={() => {
                        setDeleteError("");
                        setRemoving(m);
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
        <Button variant="text" onClick={() => setRevision((v) => v + 1)}>
          重新加载
        </Button>
      )}
      {editing !== undefined && (
        <MappingDialog
          mapping={editing}
          onClose={() => setEditing(undefined)}
          onSaved={() => {
            setEditing(undefined);
            setRevision((v) => v + 1);
          }}
        />
      )}
      {removing && (
        <ConfirmDialog
          title="删除模型映射？"
          description={`删除后内部模型 ${removing.internalId} 将无法调用，密钥仍可使用其他模型。`}
          busy={busy}
          error={deleteError}
          onClose={() => setRemoving(null)}
          onConfirm={() => void remove()}
        />
      )}
    </>
  );
}
