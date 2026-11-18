import { useEffect, useState } from "react";
import { dashboardApi, type SystemResources, type Usage } from "./api";
import { ErrorBox, Loading } from "../../components/Shared";
import {
  HISTORY_LIMIT,
  HISTORY_WINDOW,
  UsageChart,
  type ResourceSample,
} from "./UsageChart";
import { Cpu, MemoryStick, HardDrive, type LucideIcon } from "lucide-react";

const tones = {
  normal:
    "[--resource-color:var(--brand-chart)] [--resource-text:var(--accent)]",
  warning:
    "[--resource-color:#ffb900] [--resource-text:#df7800] dark:[--resource-text:#ffd477]",
  high: "[--resource-color:#ef655e] [--resource-text:#cf443e] dark:[--resource-text:#ffaaa6]",
  unavailable: "[--resource-color:#a2adb9] [--resource-text:var(--muted)]",
};

function bytes(value: number) {
  const units = ["B", "KiB", "MiB", "GiB"];
  const index =
    value > 0
      ? Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
      : 0;
  return `${(value / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}

function Resource({
  title,
  usage,
  detail,
  detailTitle,
  icon: Icon,
  metric,
  history,
}: {
  title: string;
  usage: Usage;
  detail: string;
  detailTitle?: string;
  icon: LucideIcon;
  metric: "cpu" | "memory" | "disk";
  history: ResourceSample[];
}) {
  const values = history.flatMap((sample) =>
    sample[metric] === null ? [] : [sample[metric]!],
  );
  const peak = values.length ? Math.max(...values) : null;
  const average = values.length
    ? values.reduce((sum, value) => sum + value, 0) / values.length
    : null;
  const level = usage.error
    ? "unavailable"
    : usage.percent >= 85
      ? "high"
      : usage.percent >= 60
        ? "warning"
        : "normal";
  const percent = Math.max(0, Math.min(100, usage.percent));
  return (
    <section
      className={`relative isolate grid min-h-[122px] grid-cols-[76px_minmax(0,1fr)] items-center gap-[18px] overflow-hidden rounded-xl border border-resource-border bg-resource-surface px-[18px] py-[15px] before:absolute before:inset-y-3 before:left-0 before:w-1 before:rounded-r before:bg-[color:var(--resource-color)] before:content-[''] max-[760px]:gap-[14px] max-[760px]:px-4 max-[760px]:py-[14px] ${tones[level]}`}
      aria-label={title}
    >
      <UsageChart title={title} metric={metric} samples={history} />
      <div
        className="relative grid size-[76px] place-items-center text-resource-ink [&>svg:first-child]:absolute [&>svg:first-child]:inset-0 [&>svg:first-child]:size-[76px] [&>svg:last-child]:relative"
        role={usage.error ? undefined : "meter"}
        aria-label={`${title}使用率`}
        aria-valuemin={usage.error ? undefined : 0}
        aria-valuemax={usage.error ? undefined : 100}
        aria-valuenow={usage.error ? undefined : percent}
      >
        <svg viewBox="0 0 76 76" aria-hidden="true">
          <circle
            className="fill-none stroke-resource-track [stroke-width:6]"
            cx="38"
            cy="38"
            r="32"
          />
          <circle
            className="origin-[38px_38px] -rotate-90 fill-none stroke-[color:var(--resource-color)] [stroke-width:6] [stroke-linecap:round] transition-[stroke-dasharray,stroke] duration-350 motion-reduce:transition-none"
            cx="38"
            cy="38"
            r="32"
            pathLength="100"
            strokeDasharray={`${usage.error ? 0 : percent} 100`}
          />
        </svg>
        <Icon size={29} strokeWidth={1.6} aria-hidden="true" />
      </div>
      <div className="min-w-0 text-resource-ink">
        <h3 className="mb-[2px] text-[13px] leading-5 font-semibold">
          {title}占用
        </h3>
        {usage.error ? (
          <p className="min-h-[47px] pt-2 text-[12px] text-muted" role="status">
            {usage.error}
          </p>
        ) : (
          <>
            <strong className="block font-mono text-[23px] leading-[29px] font-bold text-[color:var(--resource-text)] tabular-nums">
              {usage.percent.toFixed(1)}%
            </strong>
            <p
              className="truncate text-[11px] leading-[18px]"
              title={detailTitle}
            >
              {detail}
            </p>
          </>
        )}
        <div className="mt-1 flex flex-wrap items-center justify-between gap-x-3 gap-y-[2px] font-mono text-[10px] leading-4 tabular-nums">
          <span>
            峰值 {peak === null ? "—" : peak.toFixed(1) + "%"} · 均值{" "}
            {average === null ? "—" : average.toFixed(1) + "%"}
          </span>
          <span title="近 5 分钟有效采样数">
            近 5 分钟 · {values.length}/{HISTORY_LIMIT}
          </span>
        </div>
      </div>
    </section>
  );
}

export default function SystemResourcesPanel() {
  const [data, setData] = useState<SystemResources | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [history, setHistory] = useState<ResourceSample[]>([]);
  useEffect(() => {
    let active = true,
      pending = false;
    async function load() {
      if (pending || document.hidden) return;
      pending = true;
      setLoading(true);
      try {
        const next = await dashboardApi.resources();
        if (active) {
          setData(next);
          const time = Date.parse(next.sampledAt);
          const valid = (usage: Usage) =>
            !usage.error &&
            Number.isFinite(usage.percent) &&
            usage.percent >= 0 &&
            usage.percent <= 100
              ? usage.percent
              : null;
          const sample: ResourceSample = {
            time: Number.isFinite(time) ? time : Date.now(),
            cpu: valid(next.cpu),
            memory: valid(next.memory),
            disk: valid(next.disk),
          };
          setHistory((previous) =>
            [
              ...previous.filter(
                (p) =>
                  p.time > sample.time - HISTORY_WINDOW && p.time < sample.time,
              ),
              sample,
            ].slice(-HISTORY_LIMIT),
          );
          setError("");
        }
      } catch (e) {
        if (active) {
          setError((e as Error).message);
        }
      } finally {
        pending = false;
        if (active) setLoading(false);
      }
    }
    void load();
    const timer = window.setInterval(() => {
      void load();
    }, 5000);
    const visible = () => {
      if (!document.hidden) void load();
    };
    document.addEventListener("visibilitychange", visible);
    return () => {
      active = false;
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", visible);
    };
  }, []);

  return (
    <section className="" aria-labelledby="system-resources-title">
      <div className="mb-4 flex items-center justify-between gap-4">
        <h2
          className="mb-2 flex items-center gap-2 text-[15px] font-semibold"
          id="system-resources-title"
        >
          系统资源
        </h2>
      </div>
      <ErrorBox
        message={
          error ? `${error}${data ? "；以下保留上次采集结果。" : ""}` : ""
        }
      />
      {!data && loading && <Loading />}
      {data && (
        <>
          <div className="grid grid-cols-3 gap-4 max-[1250px]:grid-cols-1 max-[760px]:gap-3">
            <Resource
              title="CPU"
              usage={data.cpu}
              detail="总体占用率"
              icon={Cpu}
              metric="cpu"
              history={history}
            />
            <Resource
              title="内存"
              usage={data.memory}
              detail={`${bytes(data.memory.used)} / ${bytes(data.memory.total)}`}
              detailTitle={`可用 ${bytes(data.memory.available)}`}
              icon={MemoryStick}
              metric="memory"
              history={history}
            />
            <Resource
              title="硬盘"
              usage={data.disk}
              detail={`${bytes(data.disk.used)} / ${bytes(data.disk.total)} · 数据目录所在分区`}
              detailTitle={`${data.disk.path} · 可用 ${bytes(data.disk.free)}`}
              icon={HardDrive}
              metric="disk"
              history={history}
            />
          </div>
          <div className="mt-3 flex flex-wrap justify-between gap-x-5 gap-y-2 text-[11px] text-muted">
            <span>每 5 秒采样；趋势记录进入本页后的最近 5 分钟。</span>
            <span>
              采集时间：{new Date(data.sampledAt).toLocaleTimeString("zh-CN")}
            </span>
          </div>
        </>
      )}
    </section>
  );
}
