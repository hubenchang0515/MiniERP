import { act, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import DashboardPage from "./DashboardPage";
import { dashboardApi, type SystemResources } from "./api";

const auth = vi.hoisted(() => ({
  user: {
    username: "admin",
    displayName: "管理员",
    roles: [{ id: 1, name: "系统管理员", system: true }],
    permissions: ["users:read"],
  },
  can: () => true,
}));
vi.mock("../../app/AuthContext", () => ({ useAuth: () => auth }));
vi.mock("./api", () => ({ dashboardApi: { resources: vi.fn() } }));
const sample: SystemResources = {
  cpu: { percent: 37.5 },
  memory: {
    percent: 37.5,
    used: 3 * 1024 ** 3,
    total: 8 * 1024 ** 3,
    available: 5 * 1024 ** 3,
  },
  disk: {
    percent: 25,
    used: 128 * 1024 ** 3,
    total: 512 * 1024 ** 3,
    free: 384 * 1024 ** 3,
    path: "D:\\data",
  },
  sampledAt: "2026-11-15T10:00:00Z",
};
const mount = () =>
  render(
    <MemoryRouter>
      <DashboardPage />
    </MemoryRouter>,
  );

describe("dashboard system resources", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    auth.user.roles[0].system = true;
    vi.spyOn(document, "hidden", "get").mockReturnValue(false);
    vi.mocked(dashboardApi.resources).mockResolvedValue(sample);
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it("shows real percentages and capacities to a system administrator", async () => {
    mount();
    await screen.findByText("3.0 GiB / 8.0 GiB");
    expect(
      screen.getByRole("img", { name: "CPU使用率折线图" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("img", { name: "内存使用率折线图" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("meter", { name: "硬盘使用率" })).toHaveAttribute(
      "aria-valuenow",
      "25",
    );
    expect(screen.getByText(/128\.0 GiB \/ 512\.0 GiB/)).toBeInTheDocument();
    expect(screen.getByTitle(/可用 384\.0 GiB/)).toBeInTheDocument();
    expect(screen.queryByText("当前账号")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /刷新|采集中/ }),
    ).not.toBeInTheDocument();
  });
  it("does not fetch host resources for a non-system administrator", () => {
    auth.user.roles[0].system = false;
    mount();
    expect(screen.getByText("当前账号")).toBeInTheDocument();
    expect(screen.queryByText("系统资源")).not.toBeInTheDocument();
    expect(dashboardApi.resources).not.toHaveBeenCalled();
  });
  it("refreshes every five seconds and stops after leaving the page", async () => {
    vi.useFakeTimers();
    const view = mount();
    await act(async () => {});
    expect(dashboardApi.resources).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(dashboardApi.resources).toHaveBeenCalledTimes(2);
    view.unmount();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(dashboardApi.resources).toHaveBeenCalledTimes(2);
  });
  it("marks unavailable counters rather than presenting fake zeroes", async () => {
    vi.mocked(dashboardApi.resources).mockResolvedValue({
      ...sample,
      cpu: { percent: 0, error: "CPU 使用率采集失败" },
    });
    mount();
    await screen.findByText("CPU 使用率采集失败");
    expect(
      screen.queryByRole("meter", { name: "CPU使用率" }),
    ).not.toBeInTheDocument();
    expect(screen.getByText(/128\.0 GiB \/ 512\.0 GiB/)).toBeInTheDocument();
  });
  it("automatically retries a failed request", async () => {
    vi.useFakeTimers();
    vi.mocked(dashboardApi.resources).mockRejectedValueOnce(
      new Error("服务器不可用"),
    );
    mount();
    await act(async () => {});
    expect(screen.getByRole("alert")).toHaveTextContent("服务器不可用");
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(screen.getByText("3.0 GiB / 8.0 GiB")).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });
  it("keeps the graph unchanged during failed requests and reconnects after recovery", async () => {
    vi.useFakeTimers();
    const time = Date.parse(sample.sampledAt);
    vi.setSystemTime(time);
    const request = vi.mocked(dashboardApi.resources);
    request.mockResolvedValueOnce(sample);
    for (let i = 0; i < 4; i++)
      request.mockRejectedValueOnce(new Error("网络连接失败"));
    request.mockResolvedValueOnce({
      ...sample,
      sampledAt: new Date(time + 25000).toISOString(),
      cpu: { percent: 60 },
    });
    mount();
    await act(async () => {});
    const graph = screen.getByRole("img", { name: "CPU使用率折线图" });
    const originalGraph = graph.innerHTML;
    const card = within(screen.getByRole("region", { name: "CPU" }));
    for (let i = 0; i < 4; i++) {
      await act(async () => {
        await vi.advanceTimersByTimeAsync(5000);
      });
      expect(graph.innerHTML).toBe(originalGraph);
      expect(card.getByRole("meter", { name: "CPU使用率" })).toHaveAttribute(
        "aria-valuenow",
        "37.5",
      );
      expect(card.getByText("近 5 分钟 · 1/60")).toBeInTheDocument();
    }
    await act(async () => {
      await vi.advanceTimersByTimeAsync(5000);
    });
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(graph.querySelectorAll("polyline")).toHaveLength(1);
    expect(
      graph.querySelector("polyline")!.getAttribute("points")!.split(" "),
    ).toHaveLength(3);
    expect(graph.querySelectorAll(".chart-point")).toHaveLength(1);
    expect(card.getByText("近 5 分钟 · 2/60")).toBeInTheDocument();
    expect(card.getByRole("meter", { name: "CPU使用率" })).toHaveAttribute(
      "aria-valuenow",
      "60",
    );
  });
  it("accumulates real samples as a line and leaves a gap for unavailable memory", async () => {
    vi.useFakeTimers();
    const time = Date.parse(sample.sampledAt);
    vi.mocked(dashboardApi.resources)
      .mockResolvedValueOnce(sample)
      .mockResolvedValueOnce({
        ...sample,
        sampledAt: new Date(time + 5000).toISOString(),
        cpu: { percent: 60 },
        memory: { ...sample.memory, error: "内存使用量采集失败" },
      })
      .mockResolvedValueOnce({
        ...sample,
        sampledAt: new Date(time + 10000).toISOString(),
        cpu: { percent: 42 },
      });
    mount();
    await act(async () => {});
    // The first reading fills the chart horizontally without adding statistical samples.
    const cpu = screen.getByRole("img", { name: "CPU使用率折线图" });
    expect(cpu.querySelectorAll(".chart-point")).toHaveLength(1);
    const initialPoints = cpu
      .querySelector("polyline")!
      .getAttribute("points")!
      .split(" ")
      .map((point) => point.split(",").map(Number));
    expect(initialPoints).toHaveLength(2);
    expect(initialPoints[0][0]).toBe(0);
    expect(initialPoints[0][0]).toBeLessThan(initialPoints[1][0]);
    expect(initialPoints[0][1]).toBe(initialPoints[1][1]);
    expect(initialPoints.at(-1)![0]).toBe(400);
    expect(cpu.querySelector("polygon")!.getAttribute("points")).toMatch(
      /400,84$/,
    );
    const gradient = cpu.querySelector("linearGradient")!;
    expect(gradient).toHaveAttribute("x1", "0");
    expect(gradient).toHaveAttribute("x2", "400");
    expect(gradient).toHaveAttribute("y1", "0");
    expect(gradient).toHaveAttribute("y2", "0");
    expect(cpu.querySelector("polygon")).toHaveAttribute(
      "fill",
      `url(#${gradient.id})`,
    );
    const lineGradient = cpu.querySelectorAll("linearGradient")[1];
    expect(cpu.querySelector("polyline")).toHaveAttribute(
      "stroke",
      `url(#${lineGradient.id})`,
    );
    expect(lineGradient).toHaveAttribute("x2", "400");
    expect(lineGradient).toHaveAttribute("y2", "0");
    expect(lineGradient.querySelector("stop:first-child")).toHaveAttribute(
      "stop-opacity",
      "0.1",
    );
    expect(lineGradient.querySelector("stop:last-child")).toHaveAttribute(
      "stop-opacity",
      "1",
    );
    expect(gradient.querySelector("stop:first-child")).toHaveAttribute(
      "stop-opacity",
      "0.1",
    );
    expect(gradient.querySelector("stop:last-child")).toHaveAttribute(
      "stop-opacity",
      "0.3",
    );
    expect(
      within(screen.getByRole("region", { name: "CPU" })).getByText(
        "近 5 分钟 · 1/60",
      ),
    ).toBeInTheDocument();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(
      cpu.querySelector("polyline")?.getAttribute("points")?.split(" "),
    ).toHaveLength(4);
    expect(cpu.querySelectorAll(".chart-point")).toHaveLength(1);
    expect(cpu.querySelector(".chart-point")).toHaveAttribute(
      "aria-hidden",
      "true",
    );
    expect(cpu.querySelector(".chart-point")).not.toHaveAttribute("title");
    const memory = screen.getByRole("img", { name: "内存使用率折线图" });
    expect(memory.querySelectorAll(".chart-point")).toHaveLength(1);
    expect(memory.querySelectorAll("polyline")).toHaveLength(1);
    // The earlier segment ends before the latest point, leaving the failed sample disconnected.
    const earlierEnd = memory
      .querySelector("polyline")!
      .getAttribute("points")!
      .split(" ")
      .at(-1)!
      .split(",")[0];
    expect(Number(earlierEnd)).toBeLessThan(400);
    const cpuCard = within(screen.getByRole("region", { name: "CPU" }));
    expect(cpuCard.getByText("峰值 60.0% · 均值 46.5%")).toBeInTheDocument();
    expect(cpuCard.getByText("近 5 分钟 · 3/60")).toBeInTheDocument();
    expect(
      screen
        .getByRole("img", { name: "硬盘使用率折线图" })
        .querySelectorAll(".chart-point"),
    ).toHaveLength(1);
  });
});
