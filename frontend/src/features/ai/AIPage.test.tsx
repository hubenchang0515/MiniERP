import {
  render,
  screen,
  fireEvent,
  waitFor,
  within,
} from "@testing-library/react";
import {
  MemoryRouter,
  Routes,
  Route,
  useLocation,
  useNavigate,
} from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { aiRoutes } from "./routes";
import { aiApi } from "./api";
import { quotaApi } from "./quotaApi";
const auth = vi.hoisted(() => ({
  user: { id: 1, roles: [{ system: true }], permissions: ["ai:use"] },
}));
vi.mock("../../app/AuthContext", () => ({
  useAuth: () => ({
    ...auth,
    can: (code: string) => auth.user.permissions.includes(code),
  }),
}));
vi.mock("./api", () => ({
  aiApi: {
    resources: vi.fn(),
    resourceOptions: vi.fn(),
    keys: vi.fn(),
    createKey: vi.fn(),
    rotate: vi.fn(),
    revoke: vi.fn(),
    saveResource: vi.fn(),
    upstreamModels: vi.fn(),
    models: vi.fn(),
    saveModel: vi.fn(),
    deleteModel: vi.fn(),
    providerKeys: vi.fn(),
    saveProviderKey: vi.fn(),
    deleteProviderKey: vi.fn(),
  },
}));
vi.mock("./quotaApi", () => ({
  quotaApi: {
    mine: vi.fn(),
    usage: vi.fn(),
    settings: vi.fn(),
    setEnabled: vi.fn(),
    setWeekly: vi.fn(),
    grant: vi.fn(),
    users: vi.fn(),
  },
}));
const quotaBalance = {
  enabled: false,
  configured: false,
  week: "2026-11-16",
  resetsAt: "2026-11-23T00:00:00+08:00",
  weeklyPoints: 0,
  addedMicros: "0",
  usedMicros: "0",
  reservedMicros: "0",
  availableMicros: "0",
  totalUsage: { inputTokens: "0", outputTokens: "0", chargedMicros: "0" },
};
const quotaUser = {
  ...quotaBalance,
  userId: 2,
  username: "member",
  displayName: "张三",
};
const quotaPage = { items: [quotaUser], total: 1, page: 1, pageSize: 10 };
const resource = {
  id: 1,
  name: "外部服务",
  baseURL: "https://example.com/v1",
  keys: [
    {
      id: 1,
      name: "primary",
      enabled: true,
      state: "available" as const,
      retryAt: null,
      requests: 0,
      lastUsedAt: null,
    },
  ],
  enabled: true,
};
const key = {
  id: 1,
  name: "工作 KEY",
  userId: 2,
  username: "member",
  enabled: true,
  requests: 0,
  lastUsedAt: null,
};
function NavigationControls() {
  const location = useLocation(),
    navigate = useNavigate();
  return (
    <>
      <output data-testid="current-path">{location.pathname}</output>
      <button onClick={() => navigate(-1)}>浏览器后退</button>
      <button onClick={() => navigate(1)}>浏览器前进</button>
    </>
  );
}
function renderAI(path = "/ai") {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <NavigationControls />
      <Routes>
        {aiRoutes}
        <Route path="/403" element={<div>没有访问权限</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("AI resources and keys", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    auth.user.roles = [{ system: true }];
    auth.user.id = 1;
    auth.user.permissions = ["ai:use"];
    vi.mocked(quotaApi.mine).mockResolvedValue(quotaBalance);
    vi.mocked(quotaApi.usage).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    });
    vi.mocked(quotaApi.settings).mockResolvedValue({ enabled: false });
    vi.mocked(quotaApi.users).mockResolvedValue(quotaPage);
    vi.mocked(aiApi.models).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    });
    vi.mocked(aiApi.resourceOptions).mockResolvedValue({
      items: [resource],
      total: 1,
      page: 1,
      pageSize: 10,
    });
    vi.mocked(aiApi.resources).mockResolvedValue({
      items: [resource],
      total: 1,
      page: 1,
      pageSize: 10,
    });
    vi.mocked(aiApi.providerKeys).mockResolvedValue({
      items: [
        {
          ...resource.keys[0],
          resourceId: resource.id,
          resourceName: resource.name,
          resourceEnabled: true,
        },
      ],
      total: 1,
      page: 1,
      pageSize: 10,
    });
    vi.mocked(aiApi.keys).mockResolvedValue({
      items: [key],
      total: 1,
      page: 1,
      pageSize: 10,
    });
  });
  it.each([
    ["/ai/providers", "供应商", "外部服务"],
    ["/ai/provider-keys", "密钥管理", "primary"],
    ["/ai/models", "模型映射", "暂无模型映射"],
    ["/ai/allocations", "资源分配", "张三"],
    ["/ai/my-resources", "我的资源", "我的 AI 周额度"],
  ])("opens and reloads the same child at %s", async (path, label, text) => {
    const view = renderAI(path);
    await screen.findByText(text);
    expect(screen.getByRole("link", { name: label })).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(screen.getByTestId("current-path")).toHaveTextContent(path);
    view.unmount();
    renderAI(path);
    await screen.findByText(text);
    expect(screen.getByTestId("current-path")).toHaveTextContent(path);
  });
  it("uses browser history to return to the correct AI child", async () => {
    renderAI("/ai/providers");
    await screen.findByText("外部服务");
    fireEvent.click(screen.getByRole("link", { name: "密钥管理" }));
    await screen.findByText("primary");
    fireEvent.click(screen.getByRole("link", { name: "模型映射" }));
    await screen.findByText("暂无模型映射");
    fireEvent.click(screen.getByRole("button", { name: "浏览器后退" }));
    await screen.findByText("primary");
    expect(screen.getByTestId("current-path")).toHaveTextContent(
      "/ai/provider-keys",
    );
    fireEvent.click(screen.getByRole("button", { name: "浏览器前进" }));
    await screen.findByText("暂无模型映射");
    expect(screen.getByTestId("current-path")).toHaveTextContent("/ai/models");
  });
  it.each(["/ai/providers", "/ai/provider-keys", "/ai/models"])(
    "blocks direct access to %s for ordinary users",
    async (path) => {
      auth.user.roles = [];
      renderAI(path);
      await screen.findByText("没有访问权限");
      expect(aiApi.resources).not.toHaveBeenCalled();
      expect(aiApi.providerKeys).not.toHaveBeenCalled();
      expect(aiApi.models).not.toHaveBeenCalled();
    },
  );
  it("generates a key for the current account without a user selector", async () => {
    vi.mocked(aiApi.createKey).mockResolvedValue({
      secret: "mioa_test_one_time_secret",
    });
    vi.mocked(aiApi.keys).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    });
    renderAI();
    await screen.findByText("外部服务");
    fireEvent.click(screen.getByRole("link", { name: "我的资源" }));
    await screen.findByText("暂无密钥");
    expect(
      screen.queryByRole("button", { name: "重新生成" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "生成密钥" }));
    fireEvent.change(screen.getByRole("textbox", { name: "名称" }), {
      target: { value: "new key" },
    });
    expect(
      screen.queryByRole("combobox", { name: "用户" }),
    ).not.toBeInTheDocument();
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
    expect(aiApi.resourceOptions).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    expect(await screen.findByLabelText("新生成的密钥")).toHaveValue(
      "mioa_test_one_time_secret",
    );
    expect(aiApi.createKey).toHaveBeenCalledWith({
      name: "new key",
    });
    fireEvent.click(screen.getByRole("button", { name: "我已保存" }));
    expect(screen.queryByLabelText("新生成的密钥")).not.toBeInTheDocument();
  });
  it("lets an AI-authorized user rotate only their own keys", async () => {
    auth.user.roles = [{ system: false }];
    auth.user.id = 2;
    vi.mocked(aiApi.rotate).mockResolvedValue({ secret: "mioa_rotated" });
    renderAI();
    await screen.findByText("工作 KEY");
    expect(aiApi.keys).toHaveBeenCalledWith(false, 1, "");
    expect(aiApi.resources).not.toHaveBeenCalled();
    expect(
      screen.queryByRole("button", { name: "新增供应商" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "重新生成" }));
    fireEvent.click(screen.getByRole("button", { name: "确认操作" }));
    expect(await screen.findByLabelText("新生成的密钥")).toHaveValue(
      "mioa_rotated",
    );
    expect(aiApi.rotate).toHaveBeenCalledWith(1);
  });
  it("disables key creation at capacity and re-enables it after revocation", async () => {
    auth.user.id = 2;
    vi.mocked(aiApi.keys)
      .mockResolvedValueOnce({
        items: [key],
        total: 10,
        page: 1,
        pageSize: 10,
        activeCount: 10,
        limit: 10,
      })
      .mockResolvedValue({
        items: [{ ...key, enabled: false }],
        total: 10,
        page: 1,
        pageSize: 10,
        activeCount: 9,
        limit: 10,
      });
    vi.mocked(aiApi.revoke).mockResolvedValue(undefined);
    renderAI("/ai/my-resources");
    await screen.findByText("有效密钥 10 / 10");
    expect(screen.getByRole("button", { name: "生成密钥" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "撤销" }));
    fireEvent.click(screen.getByRole("button", { name: "确认操作" }));
    await screen.findByText("有效密钥 9 / 10");
    expect(screen.getByRole("button", { name: "生成密钥" })).toBeEnabled();
  });
  it("edits supplier metadata without including credentials", async () => {
    vi.mocked(aiApi.saveResource).mockResolvedValue(resource);
    renderAI();
    await screen.findByText("外部服务");
    fireEvent.click(screen.getByRole("button", { name: "编辑" }));
    expect(screen.queryByLabelText("外部 API KEY 1")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    await waitFor(() =>
      expect(aiApi.saveResource).toHaveBeenCalledWith(1, {
        name: resource.name,
        baseURL: resource.baseURL,
        enabled: true,
      }),
    );
  });
  it("does not allow an unauthorized user to generate or rotate keys", async () => {
    auth.user.roles = [];
    auth.user.id = 2;
    auth.user.permissions = [];
    vi.mocked(aiApi.revoke).mockResolvedValue(undefined);
    renderAI();
    await screen.findByText("工作 KEY");
    expect(
      screen.queryByRole("button", { name: "生成密钥" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "重新生成" }),
    ).not.toBeInTheDocument();
    expect(aiApi.resourceOptions).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "撤销" }));
    fireEvent.click(screen.getByRole("button", { name: "确认操作" }));
    await waitFor(() => expect(aiApi.revoke).toHaveBeenCalledWith(1));
  });
  it("manages mappings separately and changes suppliers without changing the internal ID", async () => {
    const mapping = {
      id: 4,
      internalId: "chat-default",
      externalId: "provider-chat-v2",
      resourceId: 1,
      resourceName: resource.name,
      resourceEnabled: true,
      enabled: true,
    };
    vi.mocked(aiApi.models).mockResolvedValue({
      items: [mapping],
      total: 1,
      page: 1,
      pageSize: 10,
    });
    vi.mocked(aiApi.resourceOptions).mockResolvedValue({
      items: [resource, { ...resource, id: 2, name: "第二供应商" }],
      total: 2,
      page: 1,
      pageSize: 10,
    });
    vi.mocked(aiApi.upstreamModels).mockResolvedValue({
      modelIds: ["provider-new"],
    });
    vi.mocked(aiApi.saveModel).mockResolvedValue(mapping);
    renderAI();
    await screen.findByText("外部服务");
    fireEvent.click(screen.getByRole("link", { name: "模型映射" }));
    await screen.findByText("chat-default");
    fireEvent.click(screen.getByRole("button", { name: "编辑" }));
    expect(screen.getByRole("textbox", { name: "内部模型 ID" })).toHaveValue(
      "chat-default",
    );
    const supplier = screen.getByRole("combobox", { name: "供应商" });
    expect(supplier).toHaveValue("外部服务");
    const inputPrice = screen.getByRole("spinbutton", {
      name: "每百万输入 Token（积分）",
    });
    const outputPrice = screen.getByRole("spinbutton", {
      name: "每百万输出 Token（积分）",
    });
    const reserve = screen.getByRole("spinbutton", {
      name: "单次预留（积分）",
    });
    expect(inputPrice).toHaveValue(0);
    expect(outputPrice).toHaveValue(0);
    expect(reserve).toHaveValue(0);
    fireEvent.click(screen.getByRole("button", { name: "获取上游模型" }));
    await screen.findByText(/已获取 1 个模型/);
    expect(aiApi.upstreamModels).toHaveBeenCalledWith({ resourceId: 1 });
    expect(screen.queryByLabelText("外部 API KEY")).not.toBeInTheDocument();
    const externalModel = screen.getByRole("combobox", { name: "外部模型 ID" });
    expect(externalModel).toHaveAttribute("autocomplete", "off");
    expect(externalModel).not.toHaveAttribute("list");
    fireEvent.click(externalModel);
    fireEvent.click(await screen.findByRole("option", { name: "provider-new" }));
    expect(externalModel).toHaveValue("provider-new");
    fireEvent.change(supplier, {
      target: { value: "第二" },
    });
    expect(screen.queryByText(/已获取 1 个模型/)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "获取上游模型" })).toBeDisabled();
    fireEvent.click(externalModel);
    expect(screen.queryByRole("option", { name: "provider-new" })).toBeNull();
    fireEvent.change(externalModel, {
      target: { value: "second-model" },
    });
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    expect(aiApi.saveModel).not.toHaveBeenCalled();
    expect(await screen.findByText("请选择供应商")).toBeInTheDocument();
    fireEvent.click(supplier);
    fireEvent.click(await screen.findByRole("option", { name: "第二供应商" }));
    expect(supplier).toHaveValue("第二供应商");
    fireEvent.change(inputPrice, { target: { value: "1000" } });
    fireEvent.change(outputPrice, { target: { value: "4000" } });
    fireEvent.change(reserve, { target: { value: "5" } });
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    await waitFor(() =>
      expect(aiApi.saveModel).toHaveBeenCalledWith(4, {
        internalId: "chat-default",
        externalId: "second-model",
        resourceId: 2,
        enabled: true,
        pricing: {
          inputPerMillion: 1000,
          outputPerMillion: 4000,
          reservePoints: 5,
          skipStreamUsage: false,
        },
      }),
    );
    expect(aiApi.saveResource).not.toHaveBeenCalled();
  });
  it("keeps supplier editing separate from mapping configuration", async () => {
    renderAI();
    await screen.findByText("外部服务");
    fireEvent.click(screen.getByRole("button", { name: "编辑" }));
    expect(
      screen.queryByRole("textbox", { name: "内部模型 ID" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "获取上游模型" }),
    ).not.toBeInTheDocument();
    expect(screen.queryByLabelText("外部 API KEY 1")).not.toBeInTheDocument();
  });
  it("keeps manual mapping available if fetching upstream models fails", async () => {
    vi.mocked(aiApi.models).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    });
    vi.mocked(aiApi.upstreamModels).mockRejectedValue(
      new Error("上游不支持模型列表"),
    );
    renderAI();
    await screen.findByText("外部服务");
    fireEvent.click(screen.getByRole("link", { name: "模型映射" }));
    await screen.findByText("暂无模型映射");
    fireEvent.click(screen.getByRole("button", { name: "新增映射" }));
    fireEvent.click(screen.getByRole("combobox", { name: "供应商" }));
    fireEvent.click(await screen.findByRole("option", { name: "外部服务" }));
    fireEvent.click(screen.getByRole("button", { name: "获取上游模型" }));
    await screen.findByText("上游不支持模型列表");
    expect(
      screen.getByRole("combobox", { name: "外部模型 ID" }),
    ).not.toBeDisabled();
    expect(screen.getByRole("button", { name: "保存" })).not.toBeDisabled();
  });
  it("searches each list and preserves the supplier query when switching tabs", async () => {
    vi.mocked(aiApi.models).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    });
    renderAI();
    await screen.findByText("外部服务");
    expect(
      screen.getByRole("button", { name: "新增供应商" }),
    ).toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "搜索供应商" }), {
      target: { value: "  example  " },
    });
    fireEvent.keyDown(screen.getByRole("textbox", { name: "搜索供应商" }), {
      key: "Enter",
    });
    await waitFor(() =>
      expect(aiApi.resources).toHaveBeenLastCalledWith("example", 1),
    );
    fireEvent.click(screen.getByRole("link", { name: "资源分配" }));
    await screen.findByText("张三");
    fireEvent.change(screen.getByRole("textbox", { name: "搜索用户" }), {
      target: { value: "member" },
    });
    fireEvent.click(screen.getByRole("button", { name: "搜索" }));
    await waitFor(() =>
      expect(quotaApi.users).toHaveBeenLastCalledWith("member", 1),
    );
    expect(aiApi.keys).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("link", { name: "我的资源" }));
    await screen.findByText("我的 AI 周额度");
    await waitFor(() =>
      expect(aiApi.keys).toHaveBeenLastCalledWith(false, 1, ""),
    );
    fireEvent.click(screen.getByRole("link", { name: "模型映射" }));
    await screen.findByText("暂无模型映射");
    expect(
      screen.getByRole("button", { name: "新增映射" }),
    ).toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "搜索模型映射" }), {
      target: { value: "chat" },
    });
    fireEvent.click(screen.getByRole("button", { name: "搜索" }));
    await waitFor(() =>
      expect(aiApi.models).toHaveBeenLastCalledWith("chat", 1),
    );
    fireEvent.click(screen.getByRole("link", { name: "供应商" }));
    await screen.findByText("外部服务");
    expect(screen.getByRole("textbox", { name: "搜索供应商" })).toHaveValue(
      "  example  ",
    );
    expect(aiApi.resources).toHaveBeenLastCalledWith("example", 1);
  });
  it("creates supplier credentials on the independent key management tab", async () => {
    vi.mocked(aiApi.saveProviderKey).mockResolvedValue(resource.keys[0]);
    vi.mocked(aiApi.resourceOptions)
      .mockResolvedValueOnce({
        items: [{ ...resource, id: 2, name: "外部测试", enabled: false }],
        total: 11,
        page: 1,
        pageSize: 10,
      })
      .mockResolvedValueOnce({
        items: [resource],
        total: 11,
        page: 2,
        pageSize: 10,
      });
    renderAI();
    await screen.findByText("外部服务");
    fireEvent.click(screen.getByRole("link", { name: "密钥管理" }));
    await screen.findByText("primary");
    expect(
      screen.queryByRole("button", { name: "生成密钥" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "新增密钥" }));
    fireEvent.change(screen.getByRole("textbox", { name: "密钥名称" }), {
      target: { value: "second" },
    });
    const supplier = screen.getByRole("combobox", { name: "供应商" });
    fireEvent.change(supplier, { target: { value: "外部" } });
    expect(
      await screen.findByRole("option", { name: /外部测试.*已停用/ }),
    ).toBeInTheDocument();
    expect(aiApi.resourceOptions).toHaveBeenLastCalledWith("外部", 1);
    fireEvent.click(screen.getByRole("button", { name: "加载更多" }));
    await screen.findByRole("option", { name: "外部服务" });
    expect(aiApi.resourceOptions).toHaveBeenLastCalledWith("外部", 2);
    fireEvent.keyDown(supplier, { key: "ArrowDown" });
    fireEvent.keyDown(supplier, { key: "ArrowDown" });
    fireEvent.keyDown(supplier, { key: "Enter" });
    expect(supplier).toHaveValue("外部服务");
    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("外部 API KEY"), {
      target: { value: "new-secret" },
    });
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    await waitFor(() =>
      expect(aiApi.saveProviderKey).toHaveBeenCalledWith(null, {
        resourceId: 1,
        name: "second",
        apiKey: "new-secret",
        enabled: true,
      }),
    );
    expect(aiApi.saveResource).not.toHaveBeenCalled();
    expect(aiApi.createKey).not.toHaveBeenCalled();
  });
  it("sets and supplements a selected user's weekly allowance before enabling enforcement", async () => {
    vi.mocked(quotaApi.setEnabled).mockResolvedValue(undefined);
    vi.mocked(quotaApi.setWeekly).mockResolvedValue(undefined);
    vi.mocked(quotaApi.grant).mockResolvedValue(undefined);
    vi.mocked(quotaApi.users)
      .mockResolvedValueOnce(quotaPage)
      .mockResolvedValue({
        ...quotaPage,
        items: [
          {
            ...quotaUser,
            configured: true,
            weeklyPoints: 50,
            availableMicros: "50000000",
          },
        ],
      });
    renderAI("/ai/allocations");
    await screen.findByText("额度管理未启用");
    expect(quotaApi.mine).not.toHaveBeenCalled();
    expect(aiApi.keys).not.toHaveBeenCalled();
    expect(
      screen.queryByRole("region", { name: "用量明细" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "生成密钥" }),
    ).not.toBeInTheDocument();
    const row = await screen.findByRole("row", { name: /张三/ });
    expect(within(row).getByText("未分配")).toBeInTheDocument();
    expect(
      within(row).getByRole("button", { name: "补发额度" }),
    ).toBeDisabled();
    fireEvent.click(within(row).getByRole("button", { name: "设置周额度" }));
    const weekly = within(screen.getByRole("dialog", { name: "设置周额度" }));
    expect(weekly.queryByLabelText("补发原因")).not.toBeInTheDocument();
    fireEvent.change(
      weekly.getByRole("spinbutton", { name: "每周额度（积分）" }),
      {
        target: { value: "50" },
      },
    );
    fireEvent.click(weekly.getByRole("button", { name: "保存周额度" }));
    await screen.findByText("已保存周额度");
    const updated = await screen.findByRole("row", { name: /张三.*50/ });
    fireEvent.click(within(updated).getByRole("button", { name: "补发额度" }));
    const grant = within(screen.getByRole("dialog", { name: "补发本周额度" }));
    expect(grant.queryByLabelText("每周额度（积分）")).not.toBeInTheDocument();
    fireEvent.change(
      grant.getByRole("spinbutton", { name: "本周补发（积分）" }),
      {
        target: { value: "10" },
      },
    );
    fireEvent.change(screen.getByRole("textbox", { name: "补发原因" }), {
      target: { value: "项目试用" },
    });
    fireEvent.click(screen.getByRole("button", { name: "补发积分" }));
    await screen.findByText("已补发本周积分");
    expect(quotaApi.setWeekly).toHaveBeenCalledWith(2, 50);
    expect(quotaApi.grant).toHaveBeenCalledWith(2, 10, "项目试用");
    fireEvent.click(screen.getByRole("button", { name: "启用额度管理" }));
    await waitFor(() => expect(quotaApi.setEnabled).toHaveBeenCalledWith(true));
  });
  it("shows personal usage and hides allocation controls from ordinary users", async () => {
    auth.user.roles = [{ system: false }];
    vi.mocked(quotaApi.mine).mockResolvedValue({
      ...quotaBalance,
      totalUsage: {
        inputTokens: "3040",
        outputTokens: "520",
        chargedMicros: "7800000",
      },
    });
    vi.mocked(quotaApi.usage).mockResolvedValue({
      items: [
        {
          requestId: "request-1",
          model: "chat-default",
          week: quotaBalance.week,
          status: "estimated",
          inputTokens: null,
          outputTokens: null,
          reservedMicros: "1200000",
          chargedMicros: "1200000",
          createdAt: "2026-11-18T12:00:00+08:00",
        },
      ],
      page: 1,
      pageSize: 10,
      total: 1,
    });
    renderAI("/ai/allocations");
    await screen.findByText("本周可用");
    expect(screen.getByText("3,560")).toBeInTheDocument();
    expect(screen.getByText("7.8")).toBeInTheDocument();
    expect(screen.getByTestId("current-path")).toHaveTextContent(
      "/ai/my-resources",
    );
    expect(quotaApi.settings).not.toHaveBeenCalled();
    expect(aiApi.keys).toHaveBeenCalledWith(false, 1, "");
    expect(
      screen.queryByRole("button", { name: "设置周额度" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "启用额度管理" }),
    ).not.toBeInTheDocument();
    const usage = screen.getByRole("region", { name: "用量明细" });
    expect(await within(usage).findByText("按预留结算")).toBeInTheDocument();
    expect(within(usage).getByText("1.2")).toBeInTheDocument();
    const keyRow = await screen.findByRole("row", { name: /工作 KEY/ });
    expect(
      keyRow.compareDocumentPosition(usage) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
  it("shows supplier-key health and preserves an existing key when editing without a secret", async () => {
    vi.mocked(aiApi.providerKeys).mockResolvedValue({
      items: [
        {
          ...resource.keys[0],
          state: "quota_exhausted",
          retryAt: "2026-11-18T01:00:00Z",
          resourceId: 1,
          resourceName: resource.name,
          resourceEnabled: true,
        },
      ],
      total: 1,
      page: 1,
      pageSize: 10,
    });
    vi.mocked(aiApi.saveProviderKey).mockResolvedValue(resource.keys[0]);
    vi.mocked(aiApi.deleteProviderKey).mockResolvedValue(undefined);
    renderAI();
    await screen.findByText("外部服务");
    fireEvent.click(screen.getByRole("link", { name: "密钥管理" }));
    await screen.findByText("额度耗尽");
    expect(
      screen.getByRole("columnheader", { name: "下次试用" }),
    ).toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "搜索供应商密钥" }), {
      target: { value: "primary" },
    });
    fireEvent.click(screen.getByRole("button", { name: "搜索" }));
    await waitFor(() =>
      expect(aiApi.providerKeys).toHaveBeenLastCalledWith("primary", 1),
    );
    await screen.findByText("primary");
    fireEvent.click(screen.getByRole("button", { name: "编辑" }));
    expect(screen.getByLabelText("外部 API KEY")).toHaveValue("");
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    await waitFor(() =>
      expect(aiApi.saveProviderKey).toHaveBeenCalledWith(1, {
        resourceId: 1,
        name: "primary",
        apiKey: "",
        enabled: true,
      }),
    );
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    fireEvent.click(screen.getByRole("button", { name: "删除" }));
    expect(aiApi.deleteProviderKey).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "确认操作" }));
    await waitFor(() =>
      expect(aiApi.deleteProviderKey).toHaveBeenCalledWith(1),
    );
  });
});
