import { useId } from "react";

export type ResourceSample = {
  time: number;
  cpu: number | null;
  memory: number | null;
  disk: number | null;
};

export const HISTORY_WINDOW = 5 * 60 * 1000;
export const HISTORY_LIMIT = 60;

export function UsageChart({
  title,
  metric,
  samples,
}: {
  title: string;
  metric: "cpu" | "memory" | "disk";
  samples: ResourceSample[];
}) {
  const gradientId = `usage-fill-${useId().replace(/:/g, "")}`;
  const lineGradientId = `${gradientId}-line`;
  const end = samples.at(-1)?.time ?? Date.now();
  const start = end - HISTORY_WINDOW;
  const x = (time: number) => ((time - start) / HISTORY_WINDOW) * 400;
  const y = (percent: number) => 5 + ((100 - percent) / 100) * 70;
  const segments: ResourceSample[][] = [];
  let segment: ResourceSample[] = [];
  for (const sample of samples) {
    if (sample[metric] === null) {
      if (segment.length) segments.push(segment);
      segment = [];
    }
    if (sample[metric] !== null) segment.push(sample);
  }
  if (segment.length) segments.push(segment);

  // Extend the first observed value visually; statistics still use only real samples.
  const first = samples[0];
  if (first && segments[0]?.[0] === first && first.time > start) {
    segments[0].unshift({ ...first, time: start });
  }
  const latest = samples.at(-1);

  return (
    <div
      className="pointer-events-none absolute top-[30px] left-0 -z-10 h-[calc(100%-30px)] w-full text-[color:var(--resource-color)]"
      role="img"
      aria-label={`${title}使用率折线图`}
    >
      <svg
        className="block size-full"
        viewBox="0 0 400 84"
        preserveAspectRatio="none"
        aria-hidden="true"
      >
        <title>{title}使用率折线图</title>
        <defs>
          <linearGradient
            id={gradientId}
            gradientUnits="userSpaceOnUse"
            x1="0"
            y1="0"
            x2="400"
            y2="0"
          >
            <stop offset="0%" stopColor="currentColor" stopOpacity="0.1" />
            <stop offset="100%" stopColor="currentColor" stopOpacity="0.3" />
          </linearGradient>
          <linearGradient
            id={lineGradientId}
            gradientUnits="userSpaceOnUse"
            x1="0"
            y1="0"
            x2="400"
            y2="0"
          >
            <stop
              offset="0%"
              stopColor="var(--resource-text)"
              stopOpacity="0.1"
            />
            <stop
              offset="100%"
              stopColor="var(--resource-text)"
              stopOpacity="1"
            />
          </linearGradient>
        </defs>
        <desc>
          最近五分钟的真实采样，纵轴为 0% 至
          100%。初始曲线以首个采样值水平延展；缺失的采样以断线表示，仅标记最新采样点。
        </desc>
        <line
          className="stroke-current [stroke-width:1] [stroke-opacity:0.18] [stroke-dasharray:1_4]"
          x1={0}
          x2={400}
          y1={y(50)}
          y2={y(50)}
        />
        <g className="">
          {segments.map(
            (points, index) =>
              points.length > 1 && (
                <polygon
                  key={index}
                  className=""
                  fill={`url(#${gradientId})`}
                  points={`${x(points[0].time)},84 ${points.map((p) => `${x(p.time)},${y(p[metric]!)}`).join(" ")} ${x(points[points.length - 1].time)},84`}
                />
              ),
          )}
        </g>
        <g className="">
          {segments.map(
            (points, index) =>
              points.length > 1 && (
                <polyline
                  key={index}
                  className="fill-none [stroke-width:2.25] [stroke-linejoin:round] [stroke-linecap:round] [vector-effect:non-scaling-stroke]"
                  stroke={`url(#${lineGradientId})`}
                  points={points
                    .map((p) => `${x(p.time)},${y(p[metric]!)}`)
                    .join(" ")}
                />
              ),
          )}
        </g>
      </svg>
      {latest && latest[metric] !== null && (
        <span
          className="chart-point pointer-events-none absolute right-0 size-[6px] translate-x-1/2 -translate-y-1/2 rounded-full bg-current shadow-point"
          style={{ top: `${(y(latest[metric]!) / 84) * 100}%` }}
          aria-hidden="true"
        />
      )}
    </div>
  );
}
