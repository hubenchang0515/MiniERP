import { useEffect, useId, useState, type FormEvent } from "react";
import Button from "../../components/Button";
import Card from "../../components/Card";
import { Dialog } from "../../components/Dialog";
import FormActions from "../../components/FormActions";
import { FormField, Input, Textarea } from "../../components/FormControls";
import ListPanel from "../../components/ListPanel";
import ListToolbar from "../../components/ListToolbar";
import { ErrorBox, Loading } from "../../components/Shared";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
} from "../../components/Table";
import {
  quotaApi,
  type AIQuotaBalance,
  type AIQuotaUsage,
  type AIQuotaUser,
} from "./quotaApi";

const beijingDate = new Intl.DateTimeFormat("zh-CN", {
  timeZone: "Asia/Shanghai",
  dateStyle: "short",
  timeStyle: "short",
});

function formatPoints(micros: string) {
  const amount = BigInt(micros);
  const magnitude = amount < 0n ? -amount : amount;
  const whole = magnitude / 1_000_000n;
  const fractional = (magnitude % 1_000_000n)
    .toString()
    .padStart(6, "0")
    .replace(/0+$/, "");
  return `${amount < 0n ? "−" : ""}${whole.toLocaleString("zh-CN")}${fractional ? `.${fractional}` : ""}`;
}

function formatTokens(tokens: string | bigint) {
  return BigInt(tokens).toLocaleString("zh-CN");
}

function QuotaTotals({ balance }: { balance: AIQuotaBalance }) {
  const items = [
    [
      "周额度",
      balance.configured
        ? balance.weeklyPoints.toLocaleString("zh-CN")
        : "未分配",
    ],
    ["本周补发", formatPoints(balance.addedMicros)],
    ["本周已用", formatPoints(balance.usedMicros)],
    ["本周预留", formatPoints(balance.reservedMicros)],
    ["本周可用", formatPoints(balance.availableMicros)],
  ];
  return (
    <>
      <dl className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        {items.map(([label, value]) => (
          <div key={label} className="rounded-md bg-subtle px-3 py-2">
            <dt className="text-[12px] text-muted">{label}</dt>
            <dd className="mt-1 break-all text-[15px] font-semibold tabular-nums">
              {value}
            </dd>
          </div>
        ))}
      </dl>
      <p className="text-[12px] text-muted">
        本周开始于 {balance.week}，下次重置为{" "}
        {beijingDate.format(new Date(balance.resetsAt))}（北京时间）。
      </p>
    </>
  );
}

function TotalUsage({ usage }: { usage: AIQuotaBalance["totalUsage"] }) {
  const items = [
    [
      "累计 Token",
      formatTokens(BigInt(usage.inputTokens) + BigInt(usage.outputTokens)),
    ],
    ["累计输入 Token", formatTokens(usage.inputTokens)],
    ["累计输出 Token", formatTokens(usage.outputTokens)],
    ["累计已用积分", formatPoints(usage.chargedMicros)],
  ];
  return (
    <div className="border-t border-line pt-4">
      <h3 className="mb-3 text-[13px] font-medium">累计用量</h3>
      <dl className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {items.map(([label, value]) => (
          <div key={label} className="rounded-md bg-subtle px-3 py-2">
            <dt className="text-[12px] text-muted">{label}</dt>
            <dd className="mt-1 break-all text-[15px] font-semibold tabular-nums">
              {value}
            </dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

const usageStatus: Record<AIQuotaUsage["status"], string> = {
  reserved: "已预留",
  dispatched: "处理中",
  actual: "按实际结算",
  estimated: "按预留结算",
  released: "已释放",
};

export function AIMyUsagePanel() {
  const headingId = useId();
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<AIQuotaUsage[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  useEffect(() => {
    let current = true;
    setLoading(true);
    setError("");
    quotaApi
      .usage(page)
      .then((data) => {
        if (current) {
          setItems(data.items);
          setTotal(data.total);
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
  }, [page]);
  return (
    <section className="mt-6" aria-labelledby={headingId}>
      <h2 id={headingId} className="mb-3 text-[15px] font-semibold">
        用量明细
      </h2>
      <ListPanel
        loading={loading}
        error={error}
        empty={!items.length}
        emptyText="暂无用量记录"
        page={page}
        total={total}
        onPageChange={setPage}
      >
        <Table>
          <TableHead>
            <tr>
              <TableHeader>请求时间</TableHeader>
              <TableHeader>模型</TableHeader>
              <TableHeader>输入 Token</TableHeader>
              <TableHeader>输出 Token</TableHeader>
              <TableHeader>积分</TableHeader>
              <TableHeader>结算状态</TableHeader>
              <TableHeader>请求编号</TableHeader>
            </tr>
          </TableHead>
          <TableBody>
            {items.map((item) => (
              <tr key={item.requestId}>
                <TableCell>
                  <time dateTime={item.createdAt}>
                    {beijingDate.format(new Date(item.createdAt))}
                  </time>
                  <p className="text-[11px] text-muted">周次 {item.week}</p>
                </TableCell>
                <TableCell>
                  <code className="whitespace-normal wrap-anywhere">
                    {item.model}
                  </code>
                </TableCell>
                <TableCell className="tabular-nums">
                  {item.inputTokens ?? "—"}
                </TableCell>
                <TableCell className="tabular-nums">
                  {item.outputTokens ?? "—"}
                </TableCell>
                <TableCell className="tabular-nums">
                  {formatPoints(
                    item.status === "reserved" || item.status === "dispatched"
                      ? item.reservedMicros
                      : item.chargedMicros,
                  )}
                </TableCell>
                <TableCell>{usageStatus[item.status]}</TableCell>
                <TableCell>
                  <code className="text-[12px] text-muted">
                    {item.requestId}
                  </code>
                </TableCell>
              </tr>
            ))}
          </TableBody>
        </Table>
      </ListPanel>
    </section>
  );
}

type Adjustment = { user: AIQuotaUser; kind: "weekly" | "grant" };

function QuotaAdjustmentDialog({
  adjustment,
  onClose,
  onSaved,
}: {
  adjustment: Adjustment;
  onClose: () => void;
  onSaved: (message: string) => void;
}) {
  const { user, kind } = adjustment;
  const grant = kind === "grant";
  const [points, setPoints] = useState(
    !grant && user.configured ? String(user.weeklyPoints) : "",
  );
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function save(event: FormEvent) {
    event.preventDefault();
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      if (grant) await quotaApi.grant(user.userId, Number(points), reason);
      else await quotaApi.setWeekly(user.userId, Number(points));
      onSaved(grant ? "已补发本周积分" : "已保存周额度");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      title={grant ? "补发本周额度" : "设置周额度"}
      description={
        user.displayName
          ? `${user.displayName}（${user.username}）`
          : user.username
      }
      onClose={onClose}
      busy={busy}
    >
      <ErrorBox message={error} />
      <form onSubmit={(event) => void save(event)} className="grid gap-4">
        <FormField
          label={grant ? "本周补发（积分）" : "每周额度（积分）"}
          required
        >
          <Input
            type="number"
            min={grant ? 1 : 0}
            max={1_000_000_000}
            step={1}
            required
            disabled={busy}
            value={points}
            onChange={(event) => setPoints(event.target.value)}
          />
        </FormField>
        {grant && (
          <FormField label="补发原因" required>
            <Textarea
              rows={2}
              maxLength={256}
              required
              disabled={busy}
              value={reason}
              onChange={(event) => setReason(event.target.value)}
            />
          </FormField>
        )}
        <FormActions>
          <Button variant="secondary" disabled={busy} onClick={onClose}>
            取消
          </Button>
          <Button type="submit" loading={busy}>
            {grant ? "补发积分" : "保存周额度"}
          </Button>
        </FormActions>
      </form>
    </Dialog>
  );
}

export function AIMyQuotaPanel() {
  const [balance, setBalance] = useState<AIQuotaBalance>();
  const [error, setError] = useState("");
  useEffect(() => {
    let current = true;
    quotaApi
      .mine()
      .then((data) => {
        if (current) setBalance(data);
      })
      .catch((e: Error) => {
        if (current) setError(e.message);
      });
    return () => {
      current = false;
    };
  }, []);
  return (
    <Card className="mb-4 grid gap-4">
      <div>
        <h2 className="text-[15px] font-semibold">我的 AI 周额度</h2>
        {balance && (
          <p className="mt-1 text-[12px] text-muted">
            {balance.enabled ? "额度管理已启用" : "额度管理未启用"}
          </p>
        )}
      </div>
      <ErrorBox message={error} />
      {balance ? (
        <>
          <QuotaTotals balance={balance} />
          <TotalUsage usage={balance.totalUsage} />
        </>
      ) : (
        !error && <Loading />
      )}
    </Card>
  );
}

export function AIQuotaAllocationPanel() {
  const [enabled, setEnabled] = useState<boolean>();
  const [settingsError, setSettingsError] = useState("");
  const [busy, setBusy] = useState(false);
  const [adjustment, setAdjustment] = useState<Adjustment | null>(null);
  const [notice, setNotice] = useState("");
  const [text, setText] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [items, setItems] = useState<AIQuotaUser[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [revision, setRevision] = useState(0);

  useEffect(() => {
    let current = true;
    quotaApi
      .settings()
      .then((settings) => {
        if (current) setEnabled(settings.enabled);
      })
      .catch((e: Error) => {
        if (current) setSettingsError(e.message);
      });
    return () => {
      current = false;
    };
  }, []);

  useEffect(() => {
    let current = true;
    setLoading(true);
    setError("");
    quotaApi
      .users(query, page)
      .then((result) => {
        if (current) {
          setItems(result.items);
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
  }, [query, page, revision]);

  async function toggle() {
    if (enabled === undefined || busy) return;
    setBusy(true);
    setSettingsError("");
    try {
      await quotaApi.setEnabled(!enabled);
      setEnabled(!enabled);
    } catch (e) {
      setSettingsError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <Card className="mb-4 grid gap-3">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-[15px] font-semibold">用户周额度</h2>
            {enabled !== undefined && (
              <p className="mt-1 text-[12px] text-muted">
                {enabled ? "额度管理已启用" : "额度管理未启用"}
              </p>
            )}
          </div>
          <Button
            variant="secondary"
            disabled={enabled === undefined}
            loading={busy}
            onClick={() => void toggle()}
          >
            {enabled ? "停用额度管理" : "启用额度管理"}
          </Button>
        </div>
        <ErrorBox message={settingsError} />
        {enabled === false && (
          <p className="text-[12px] text-muted">
            启用前请先为用户分配周额度并为模型定价。
          </p>
        )}
      </Card>
      <ListToolbar
        label="搜索用户"
        placeholder="搜索用户名或姓名…"
        value={text}
        onChange={setText}
        onSearch={() => {
          setQuery(text.trim());
          setPage(1);
          setRevision((value) => value + 1);
        }}
      />
      {notice && (
        <p role="status" className="mb-3 text-[13px] text-accent">
          {notice}
        </p>
      )}
      <ListPanel
        loading={loading}
        error={error}
        empty={!items.length}
        emptyText="暂无匹配用户"
        page={page}
        total={total}
        onPageChange={setPage}
      >
        <Table>
          <TableHead>
            <tr>
              <TableHeader>用户</TableHeader>
              <TableHeader>周额度</TableHeader>
              <TableHeader>本周补发</TableHeader>
              <TableHeader>本周已用</TableHeader>
              <TableHeader>本周预留</TableHeader>
              <TableHeader>本周可用</TableHeader>
              <TableHeader>累计 Token</TableHeader>
              <TableHeader>累计已用积分</TableHeader>
              <TableHeader>操作</TableHeader>
            </tr>
          </TableHead>
          <TableBody>
            {items.map((user) => (
              <tr key={user.userId}>
                <TableCell>
                  <strong>{user.displayName || user.username}</strong>
                  {user.displayName && user.displayName !== user.username && (
                    <span className="ml-2 text-[12px] text-muted">
                      {user.username}
                    </span>
                  )}
                </TableCell>
                <TableCell>
                  {user.configured
                    ? user.weeklyPoints.toLocaleString("zh-CN")
                    : "未分配"}
                </TableCell>
                <TableCell>{formatPoints(user.addedMicros)}</TableCell>
                <TableCell>{formatPoints(user.usedMicros)}</TableCell>
                <TableCell>{formatPoints(user.reservedMicros)}</TableCell>
                <TableCell>{formatPoints(user.availableMicros)}</TableCell>
                <TableCell className="tabular-nums">
                  <p>
                    {formatTokens(
                      BigInt(user.totalUsage.inputTokens) +
                        BigInt(user.totalUsage.outputTokens),
                    )}
                  </p>
                  <p className="text-[12px] text-muted">
                    输入 {formatTokens(user.totalUsage.inputTokens)} · 输出{" "}
                    {formatTokens(user.totalUsage.outputTokens)}
                  </p>
                </TableCell>
                <TableCell className="tabular-nums">
                  {formatPoints(user.totalUsage.chargedMicros)}
                </TableCell>
                <TableCell>
                  <div className="flex gap-3">
                    <Button
                      variant="text"
                      onClick={() => setAdjustment({ user, kind: "weekly" })}
                    >
                      设置周额度
                    </Button>
                    <Button
                      variant="text"
                      disabled={!user.configured}
                      title={user.configured ? undefined : "请先设置周额度"}
                      onClick={() => setAdjustment({ user, kind: "grant" })}
                    >
                      补发额度
                    </Button>
                  </div>
                </TableCell>
              </tr>
            ))}
          </TableBody>
        </Table>
      </ListPanel>
      {adjustment && (
        <QuotaAdjustmentDialog
          adjustment={adjustment}
          onClose={() => setAdjustment(null)}
          onSaved={(message) => {
            setAdjustment(null);
            setNotice(message);
            setRevision((value) => value + 1);
          }}
        />
      )}
    </>
  );
}
