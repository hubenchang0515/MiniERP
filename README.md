# MiniERP

小型企业内部管理系统。前端使用 Vite、React 和 Tailwind CSS；后端使用 Go、Gin、GORM 和 SQLite。发布时，前端文件会先复制到 Go 的嵌入目录，再通过 `go:embed` 编进程序。每个平台分发一个可执行文件；SQLite 数据库是程序运行后生成的独立文件，不会被打进程序。

## 目录和文件说明

```text
MiniERP/
├── README.md                                      # 本文件：开发、构建、配置和接口说明
├── .gitignore                                     # 忽略依赖、数据库、配置和构建产物
├── config.example.toml                            # TOML 配置样例；复制为 config.toml 后修改
├── frontend/                                      # 浏览器端项目
│   ├── package.json                               # 前端依赖和开发、构建、测试命令
│   ├── package-lock.json                          # 锁定依赖版本，供 npm ci 安装
│   ├── index.html                                 # Vite 使用的网页 HTML 外壳
│   ├── tsconfig.json                              # TypeScript 编译和类型检查规则
│   ├── vite.config.ts                              # React、Tailwind 插件和开发时 API 代理
│   ├── dist/                                      # 自动生成的前端构建文件；不手动编辑
│   └── src/                                       # 前端源代码
│       ├── main.tsx                               # 浏览器端启动入口，加载 React 应用和样式
│       ├── app/                                   # 应用入口、登录状态和页面路由
│       │   ├── App.tsx                            # React 根组件，创建浏览器路由环境
│       │   ├── router.tsx                         # 前端 URL、页面和访问权限的对应关系
│       │   ├── RouteGuards.tsx                    # 按初始化、登录和权限状态拦截页面
│       │   ├── AuthContext.tsx                    # 在各页面之间共享当前用户、登录和退出状态
│       │   └── router.test.tsx                    # 测试回跳地址和路由访问限制
│       ├── components/                            # 多个功能页面共用的界面部件
│       │   └── AppLayout.tsx                      # 登录后的侧栏、顶栏、导航和内容区域
│       ├── features/                              # 按业务功能放置页面和后端请求
│       │   ├── auth/                              # 首次初始化和登录功能
│       │   │   ├── api.ts                         # 初始化、登录和查询当前账号的请求
│       │   │   ├── LoginPage.tsx                  # 登录页面
│       │   │   ├── SetupPage.tsx                  # 空数据库中创建首位管理员的页面
│       │   │   └── AuthPages.test.tsx             # 测试初始化和登录页面
│       │   ├── dashboard/                         # 登录后的系统首页
│       │   │   └── DashboardPage.tsx              # 显示当前账号和可进入的功能
│       │   ├── users/                             # 用户管理功能
│       │   │   ├── api.ts                         # 查询、创建、编辑、停用用户和重设密码
│       │   │   ├── UsersPage.tsx                  # 用户列表页面
│       │   │   ├── UserFormPage.tsx               # 创建或编辑用户、分配角色的表单
│       │   │   └── UserPages.test.tsx             # 测试用户列表和表单
│       │   └── roles/                             # 角色和权限管理功能
│       │       ├── api.ts                         # 查询和维护角色、读取可分配权限
│       │       ├── RolesPage.tsx                  # 角色列表页面
│       │       ├── RoleFormPage.tsx               # 创建或编辑角色并勾选权限
│       │       └── RolePages.test.tsx             # 测试角色列表和表单
│       ├── lib/                                   # 不属于某一个业务页面的基础工具
│       │   ├── apiClient.ts                       # 统一发送请求、带上登录凭证并整理错误
│       │   └── authStorage.ts                     # 在浏览器 localStorage 中保存登录凭证
│       ├── styles/                                # 全站共用样式
│       │   └── index.css                          # Tailwind CSS 入口和少量全局样式
│       └── test/                                  # 前端测试共用准备代码
│           └── setup.ts                           # 初始化模拟浏览器和测试断言工具
├── backend/                                       # Go 后端项目
│   ├── go.mod                                     # Go 模块名、Go 版本和直接依赖
│   ├── go.sum                                     # Go 依赖校验信息
│   ├── cmd/                                       # 可执行程序的启动入口
│   │   └── minierp/                               # MiniERP 服务程序
│   │       └── main.go                            # 读取配置、连接数据库并启动 HTTP 服务
│   └── internal/                                  # 后端内部实现，不供其他 Go 模块直接导入
│       ├── config/                                # 读取默认值、TOML、环境变量和命令行参数
│       │   ├── config.go                          # 配置字段、来源合并、校验和帮助信息
│       │   └── config_test.go                     # 测试默认值、错误输入和配置覆盖顺序
│       ├── database/                              # SQLite 连接和数据库表结构
│       │   ├── sqlite.go                          # 建立 SQLite 文件和 GORM 数据库连接
│       │   ├── schema.go                          # 用 AutoMigrate 建表并初始化权限清单
│       │   └── schema_test.go                     # 测试首次建表和重复启动
│       ├── model/                                 # 描述数据库里保存的数据
│       │   ├── user.go                            # 用户、密码哈希和账号状态字段
│       │   ├── role.go                            # 角色及用户、权限之间的关联
│       │   └── permission.go                      # 首版支持的权限名称和说明
│       ├── repository/                            # 只负责把数据读写到数据库
│       │   ├── user_repository.go                 # 查询、新增和更新用户记录
│       │   ├── user_repository_test.go            # 测试用户记录读写
│       │   ├── role_repository.go                 # 查询和更新角色及关联关系
│       │   └── role_repository_test.go             # 测试角色、权限和用户关联读写
│       ├── service/                               # 放业务规则，不处理 HTTP 请求格式
│       │   ├── auth_service.go                    # 首次初始化、密码验证和 Token 规则
│       │   ├── auth_service_test.go                # 测试初始化、登录和 Token
│       │   ├── user_service.go                    # 用户校验、管理和管理员保护规则
│       │   ├── user_service_test.go                # 测试用户操作和管理员保护规则
│       │   ├── role_service.go                    # 角色、权限及删除限制规则
│       │   ├── role_service_test.go                # 测试角色权限和删除限制
│       │   ├── dto.go                              # 业务层输入和输出的数据结构
│       │   ├── errors.go                           # 可转换成 HTTP 错误响应的业务错误
│       │   └── validation.go                       # 用户名、密码和权限等字段校验
│       ├── transport/                             # 把外部请求接到内部业务逻辑的入口层
│       │   └── http/                              # 用 HTTP/JSON 提供浏览器调用的接口
│       │       ├── router.go                      # 将 API 地址、权限检查和处理函数连起来
│       │       ├── middleware/                    # 每个请求执行的通用检查
│       │       │   └── auth.go                    # 验证登录 Token 和接口权限
│       │       └── handler/                       # 接收请求、调用业务层并返回 HTTP 结果
│       │           ├── auth_handler.go            # 初始化、登录、状态和当前用户接口
│       │           ├── auth_handler_test.go       # 测试认证接口输入和响应
│       │           ├── user_handler.go            # 用户管理接口的 HTTP 处理
│       │           ├── user_handler_test.go       # 测试用户接口和权限检查
│       │           ├── role_handler.go            # 角色及权限接口的 HTTP 处理
│       │           ├── role_handler_test.go       # 测试角色接口和权限检查
│       │           └── common.go                  # 多个处理函数共用的错误和响应方法
│       ├── testutil/                              # 只供后端测试复用的工具
│       │   └── sqlite.go                          # 创建隔离的临时 SQLite 测试数据库
│       └── web/                                   # 让 Gin 返回前端文件
│           ├── embed.go                           # 声明把 static 文件嵌入程序
│           ├── handler.go                         # 返回静态文件并为前端路由回退到首页
│           └── static/                            # 构建脚本生成并嵌入的前端文件
├── scripts/                                       # 开发和发布脚本
│   ├── install-deps.ps1                           # Windows 上单独安装前端和后端依赖
│   ├── install-deps.sh                            # Linux/macOS 上执行相同的依赖安装
│   ├── build.ps1                                  # Windows 上构建前端并编译指定目标的程序
│   ├── build.sh                                   # Linux/macOS 上执行相同的发布构建
│   ├── dev.ps1                                    # Windows 上构建前端并启动前后端开发服务
│   └── dev.sh                                     # Linux/macOS 上启动前后端开发服务
└── release/                                       # 自动生成的跨平台程序，不手动编辑
```

`frontend/dist/`、`backend/internal/web/static/` 和 `release/` 都是构建产物。开发时运行构建脚本会先生成前端文件，再复制到 `static/`，最后编译 Go 程序。`static/.keep` 和 `static/placeholder.html` 是空构建时供 Go 编译器使用的占位文件；构建后的服务不会把它们提供给浏览器。

## 前端页面路由

所有页面都由 React Router 管理。页面上的权限检查用于改善导航体验；后端也会独立检查权限，不能只依赖前端保护。

| 路由 | 访问条件 | 页面与行为 |
|---|---|---|
| `/` | 所有人 | 查看初始化和登录状态，转到 `/setup`、`/login` 或 `/dashboard` |
| `/setup` | 数据库尚未初始化 | 创建首位管理员；成功后直接登录并进入首页 |
| `/login` | 已初始化且当前未登录 | 登录后回到原目标地址；没有目标时进入首页 |
| `/dashboard` | 已登录 | 系统管理员以紧凑卡片显示 CPU、内存及硬盘占用：环形使用率、容量、背景趋势，以及近 5 分钟的峰值、均值和有效采样数。每 5 秒自动采样，最多保留 60 点，离开页面后清空；无手动刷新按钮。趋势初始按首个采样值水平延展，之后仅显示最新采样点标记，延展部分不计入统计。硬盘统计数据目录所在分区；其他用户显示账号信息。保留可用功能入口 |
| `/account/password` | 已登录，无管理权限要求 | 验证当前密码后修改本人密码，成功后重新登录 |
| `/ai` | 已登录 | AI 资源入口，管理员默认进入供应商，普通用户默认进入我的资源 |
| `/ai/providers` | 系统管理员 | 供应商列表和配置 |
| `/ai/provider-keys` | 系统管理员 | 供应商密钥管理 |
| `/ai/models` | 系统管理员 | 跨供应商模型映射 |
| `/ai/allocations` | 系统管理员 | 资源分配，查询用户额度，分别设置周额度和补发本周积分，以及启停额度管理。普通用户打开旧地址会转到我的资源 |
| `/ai/my-resources` | 已登录 | 我的资源，查看本人周额度、用量和密钥。有“使用 AI”权限时可生成和重新生成本人密钥 |
| `/users` | `users:read`；`users:manage` 也可读取 | 用户列表 |
| `/users/new` | `users:manage` | 创建用户并分配角色 |
| `/users/:id/edit` | `users:manage` | 编辑用户、分配角色或重设密码 |
| `/roles` | `roles:read`；`roles:manage` 也可读取 | 角色列表，支持按名称或说明搜索，每页 10 条 |
| `/roles/new` | `roles:manage` | 创建角色并勾选权限 |
| `/roles/:id/edit` | `roles:manage` | 编辑角色及权限 |
| `/403` | 已登录 | 提示当前账号没有访问权限 |
| 其他地址 | 所有人 | 显示 404 页面 |

`returnTo` 只接受站内地址，并拒绝跳回登录页或初始化页，避免把用户带到外部网站。登录时会检查当前账号能否访问回跳目标，无权访问时进入工作台。即使没有任何角色或权限，也可以进入工作台，并通过侧栏的“修改密码”入口修改本人密码。

界面首次使用时默认跟随系统主题，并实时响应系统主题变化。侧栏底部及登录、初始化页右上角的主题按钮仅显示图标，点击按“跟随系统 → 深色 → 浅色 → 跟随系统”循环切换；选择保存在当前浏览器中，刷新页面、重新登录后仍会保留。

主题按钮旁的调色盘按钮用于切换主题色，按“绿色 → 蓝色 → 紫色 → 橙色 → 玫红色 → 绿色”循环切换。主题色同时适配浅色和深色模式，独立保存在当前浏览器中，刷新页面后保持所选颜色。

## 后端 HTTP 接口

接口使用 JSON。需要登录的接口通过 `Authorization: Bearer <token>` 传递凭证。错误响应统一为 `{ "code": "...", "message": "..." }`。

| 方法和路径 | 登录与权限要求 | 用途 |
|---|---|---|
| `GET /api/auth/status` | 无 | 查看数据库是否已经完成首次初始化 |
| `POST /api/auth/bootstrap` | 数据库必须为空 | 创建首位管理员并返回登录凭证 |
| `POST /api/auth/login` | 无 | 验证账号密码并返回 Token 和用户信息 |
| `GET /api/auth/me` | 需要登录 | 返回当前用户、角色和有效权限 |
| `POST /api/auth/password` | 需要登录，无管理权限要求 | 仅修改本人密码，请求包含 `currentPassword` 和 `newPassword`；成功返回 204，并撤销该账号的全部登录凭证 |
| `GET /api/ai/resources` | 系统管理员 | 分页列出供应商；`q` 搜索名称或 API 地址，支持 `page`、`pageSize`，仅返回密钥名称、启用和健康状态、恢复时间及使用次数，不返回外部 KEY |
| `GET /api/ai/resources/options` | 系统管理员 | 分页搜索供应商，返回 ID、名称和启用状态，供独立模型映射选择 |
| `POST /api/ai/resources` | 系统管理员 | 新增供应商：`name`、`baseURL`、`enabled`，不包含密钥和模型映射；创建后在“密钥管理”中添加供应商密钥 |
| `POST /api/ai/resources/models` | 系统管理员 | 通过 `resourceId` 使用后端保存的地址和密钥获取上游模型；也支持临时 `baseURL`、`apiKey`，仅返回 `modelIds`，不保存表单 |
| `PATCH /api/ai/resources/:id` | 系统管理员 | 仅修改供应商名称、地址或启用状态，不修改密钥池 |
| `GET /api/ai/provider-keys` | 系统管理员 | 分页搜索供应商密钥，`q` 匹配密钥名称或供应商；返回供应商、健康状态、恢复时间和使用记录，不返回密钥值 |
| `POST /api/ai/provider-keys` | 系统管理员 | 独立添加供应商密钥：`resourceId`、`name`、`apiKey`、`enabled`；每个供应商最多 100 个 |
| `PATCH /api/ai/provider-keys/:id` | 系统管理员 | 修改供应商密钥名称、密钥值或启用状态；`apiKey` 留空保留原值；不允许更换所属供应商 |
| `DELETE /api/ai/provider-keys/:id` | 系统管理员 | 独立删除供应商密钥，可以删除最后一个；没有可用密钥时该供应商无法处理调用 |
| `GET /api/ai/models` | 系统管理员 | 分页搜索全局模型映射，支持 `q`、`page`、`pageSize`，包括供应商名称及启用状态 |
| `POST /api/ai/models` | 系统管理员 | 新增全局映射：`internalId`、`resourceId`、`externalId`、`enabled`；内部 ID 全局唯一 |
| `PATCH /api/ai/models/:id` | 系统管理员 | 修改全局映射，可在保留内部 ID 的情况下切换供应商及外部模型 |
| `DELETE /api/ai/models/:id` | 系统管理员 | 删除模型映射，不撤销任何访问密钥 |
| `GET /api/ai/keys` | 已登录 | 分页查看本人 KEY，返回独立于搜索和分页的 `activeCount` 与 `limit`。仅系统管理员可使用 `all=true` 查看全部 |
| `POST /api/ai/keys` | `ai:use` | 通过 `name` 为本人生成通用 KEY，完整值仅返回一次。有效密钥达到 10 个时返回 `409 ai_key_limit` |
| `POST /api/ai/keys/:id/rotate` | 本人且拥有 `ai:use` | 重新生成本人 KEY，旧值立即失效；管理员不能替其他用户重新生成 |
| `DELETE /api/ai/keys/:id` | 本人或系统管理员 | 永久撤销访问密钥 |
| `GET /api/ai/quota` | 已登录 | 本人本周额度、补发、已用、预留、可用、重置时间、额度开关状态和历史累计用量 |
| `GET /api/ai/quota/usage` | 已登录 | 分页查询本人的请求用量和结算状态 |
| `GET /api/ai/quota/settings` | 系统管理员 | 查询全局额度管理开关 |
| `PATCH /api/ai/quota/settings` | 系统管理员 | 通过 `enabled` 启用或停用全局额度管理 |
| `GET /api/ai/quota/users` | 系统管理员 | 分页查看用户本周额度和历史累计用量，包括未分配的账号。支持按用户名或姓名搜索 |
| `GET /api/ai/quota/users/:id` | 系统管理员 | 查询指定用户的周额度和本周余额 |
| `PATCH /api/ai/quota/users/:id` | 系统管理员 | 通过 `weeklyPoints` 设置指定用户的常规周额度，立即生效 |
| `POST /api/ai/quota/users/:id/grants` | 系统管理员 | 通过 `points`、`reason` 向指定用户补发本周积分 |
| `GET /api/system/resources` | 系统管理员角色 | 直接返回后台采样缓存：服务器整机 CPU 使用率、物理内存使用率及容量、数据目录所在分区的硬盘使用率及容量、采集时间；单项采集失败时返回该项的 `error`；首次采样未完成时返回 503 |
| `GET /api/permissions` | `roles:read` 或 `roles:manage` | 返回可分配的权限清单 |
| `GET /api/users` | `users:read` | 搜索和分页列出用户；支持 `q`、`page`、`pageSize` |
| `POST /api/users` | `users:manage` | 创建用户 |
| `GET /api/users/:id` | `users:read` | 查询单个用户 |
| `PATCH /api/users/:id` | `users:manage` | 修改姓名和角色；用户名创建后不可修改 |
| `PATCH /api/users/:id/status` | `users:manage` | 启用或停用用户 |
| `POST /api/users/:id/password-reset` | `users:manage` | 重设用户密码 |
| `GET /api/roles` | `roles:read` 或 `users:manage` | 分页搜索角色，包含权限和成员数；成员数按当前页批量统计 |
| `GET /api/roles/options` | `users:manage` | 分配角色使用的精简分页搜索接口，仅返回 `id`、`name`、`description`，排除系统管理员角色 |
| `POST /api/roles` | `roles:manage` | 创建角色 |
| `GET /api/roles/:id` | `roles:read` 或 `roles:manage` | 查看单个角色 |
| `PATCH /api/roles/:id` | `roles:manage` | 修改角色名称、说明和权限 |
| `DELETE /api/roles/:id` | `roles:manage` | 删除没有分配给用户的角色 |

权限包括 `users:read`、`users:manage`、`roles:read`、`roles:manage`、`ai:use`。具有某项 `manage` 权限的用户也能执行对应的 `read` 操作。普通用户可以拥有多个角色，有效权限是这些角色权限的合并。初始化创建的系统管理员账号不可调整角色、不可停用；新建和编辑其他账号时均不可分配系统管理员角色。编辑用户时省略 `roleIds` 表示保留原有角色，传入空数组表示清空普通用户的角色。用户停用后保留记录，不提供硬删除。登录失败时不区分账号不存在、密码错误或账号停用。

两个角色列表接口均接受 `q`、`page`、`pageSize`。`q` 匹配角色名称或说明；默认页码为 1、每页 20 条，`pageSize` 可选 1–100，页码范围为 1–1000000。响应统一为 `{ "items": [], "total": 0, "page": 1, "pageSize": 20 }`，不再直接返回完整数组。前端角色管理及角色选择器均按每页 10 条读取，翻页或搜索不会清除已选角色；编辑用户时，当前页之外的原有角色也会保留。删除某页最后一条角色后，角色列表自动回到上一页并重新加载。

服务启动时创建唯一的资源采样任务，立即采集一次，之后每 1 秒更新一次共享快照。前端仍每 5 秒读取一次快照。CPU 的 500ms 采样窗口仅在后台执行，接口直接读取快照，不等待采样，也不会因客户端数量增加而重复采集。首次快照尚未就绪时立即返回 503；服务退出时取消后台任务。

系统资源请求失败时，页面保留上次成功的数值、曲线和统计，不插入空采样点；页面继续每 5 秒重试，成功后再更新并连续绘制曲线。

前端布局、间距、颜色和响应式样式直接使用组件中的 Tailwind 工具类。`src/styles/index.css` 仅保留 Tailwind 入口、主题颜色变量和光晕阴影令牌；不再维护独立的组件 CSS 或深色覆盖样式表。

通用图标按钮位于 `frontend/src/components/IconButton.tsx`。传入 `icon` 和 `label` 即可统一图标、尺寸、样式、悬停提示及无障碍标签；使用 `onClick` 执行操作，或传入 `to` 渲染站内导航链接。普通按钮默认 `type="button"`，并支持原生按钮属性。主题、主题色、修改密码和退出登录均使用此组件。

其余通用界面组件也放在 `frontend/src/components/`，页面通过组合这些组件呈现业务内容：

| 文件 | 组件与约定 |
|---|---|
| `Button.tsx` | 普通按钮及站内链接，支持 `primary`、`secondary`、`danger`、`text`、`dangerText`；`loading` 显示加载状态并禁用按钮。默认不提交表单，提交时显式传入 `type="submit"` |
| `FormControls.tsx` | `FormField` 统一标签、必填标记、说明和错误关联；`Input`、`Textarea` 保留原生属性；`PasswordInput` 可通过 `reveal` 启用显隐；`CheckboxItem` 用于角色和权限选项 |
| `FormSection.tsx` | 统一表单分区的标题、说明、边框和间距 |
| `Dialog.tsx` | `Dialog` 提供弹窗容器、焦点限制、焦点恢复、Esc 与遮罩关闭、背景滚动锁定；`busy` 时禁止关闭。`ConfirmDialog` 组合确认按钮与错误提示，业务请求由页面处理 |
| `Alert.tsx` | 错误、成功和说明提示；现有 `ErrorBox` 委托此组件渲染错误 |
| `Table.tsx` | 表格、表头、表体和单元格基础组件，统一滚动和响应式样式；列定义、权限和行操作仍保留在业务页面 |
| `Badge.tsx` | 用户角色与角色权限的标签样式；`onRemove`、`removeLabel` 和 `disabled` 支持角色选择器中的可移除标签 |
| `SearchField.tsx`、`Pagination.tsx` | 用户管理、角色管理与角色选择器共用搜索和分页组件。搜索支持 Enter，并避免触发外层用户表单提交 |
| `ListPanel.tsx` | 用户和角色列表共用容器，统一加载、失败、空数据、表格及分页布局；加载或失败时禁止翻页。请求和列内容仍由页面提供 |
| `FormActions.tsx` | 用户表单、角色表单、修改密码与弹窗共用操作区，统一按钮排列、间距和窄屏换行 |
| `ResultState.tsx` | 应用连接失败、403 和 404 共用结果布局，支持图标、标题、说明及操作区 |
| `Logo.tsx` | 侧栏、移动端页头及登录／初始化页共用项目标识，颜色跟随主题；图形源文件为 `frontend/public/logo.svg`，浏览器标签页也使用此文件 |

用户、角色、登录、初始化、修改密码及工作台页面已使用以上组件。输入校验、请求、权限判断和业务状态由页面及业务层负责，不放入这些通用界面组件。

用户模块的 `features/users/AccountActionRow.tsx` 统一账号状态与重设密码操作行，保留在业务模块内复用。系统管理员账号的角色和停用限制继续由业务页面与后端执行。

## AI 供应商、密钥管理和资源分配

系统管理员进入侧栏“AI 资源”，在“供应商”页签维护名称、API 基础地址和启用状态。基础地址应包含版本路径，例如 `https://api.example.com/v1`，也可接入企业内网 HTTP 兼容服务。供应商可以先创建而不填写密钥，停用供应商会阻止发往它的所有新代理请求。

AI 资源的子页面使用独立路由。管理员通过顶部导航进入供应商、密钥管理、模型映射、资源分配和我的资源。普通用户只可进入我的资源，原有的资源分配地址会跳转到个人页。刷新、收藏和浏览器前进或后退均可直接恢复有权限的页面。

各列表在表格前显示搜索和当前页面可用的操作。“资源分配”展示用户的周额度、已补发、已用、预留和可用积分。周额度与补发使用各自的表单。个人密钥集中在“我的资源”。

供应商密钥在独立的“密钥管理”页签中逐条新增、编辑、停用和删除，支持按密钥名称或供应商搜索、分页显示，以及查看额度状态、使用次数、最近使用和下次试用时间。新增时选择供应商、填写名称和外部 API KEY；每个供应商最多 100 个密钥。编辑时所属供应商只读，API KEY 留空保留原值，填写新值会清除原不可用状态。密钥原值和片段均不会回显。删除最后一个密钥后，供应商配置仍保留，但暂时无法处理调用。原管理用户访问密钥的“密钥管理”已改名为“资源分配”，管理员和普通用户的侧栏入口统一显示“AI 资源”。

转发时优先选择并发请求较少的可用密钥，负载相同时按轮询顺序分配。上游明确返回额度耗尽、临时限流或认证无效时，持久化该密钥的不可用状态，并在尚未向客户端输出内容前尝试同一供应商的其他可用密钥，每个密钥在同一请求中最多尝试一次。网络断开、普通服务错误或已经开始输出的流式响应不会自动重放请求。所有密钥不可用时返回 503；本次尝试用尽所有候选密钥时保留最后一次上游错误状态码，正文统一脱敏。

额度耗尽和临时限流优先采用 `Retry-After` 的秒数或 HTTP 日期；临时限流也支持 `x-ratelimit-reset-requests`、`x-ratelimit-reset-tokens` 的时长，取较晚时间。额度错误不使用请求速率的重置时间；无可靠提示时等待 5 分钟。到期后只有一个实际调用可以试用该密钥，收到成功响应后恢复，否则继续暂停。恢复时间不代表已确认额度到账，不会在空闲时主动发起计费探测，也不会因为模型列表读取成功就恢复调用额度。认证无效的密钥需管理员更换；手动停用的密钥不会自动启用。健康状态在程序重启后保留。识别基于 HTTP 401/402/429 及常见 OpenAI 兼容错误类型，供应商自定义错误、HTTP 200 流中的后续错误无法通用识别。具体限流和额度差异见 [OpenAI 错误说明](https://developers.openai.com/api/docs/guides/error-codes) 与 [限流响应头说明](https://developers.openai.com/api/docs/guides/rate-limits)。

模型映射在独立的“模型映射”页签统一管理，不放在供应商内部。每条记录包含 `internalId`、`resourceId`、`externalId`、`enabled`，例如 `chat-default → 供应商 A → provider-chat-v2`。内部 ID 在整个系统中唯一，两类模型 ID 均为 1–256 字节，不允许空白和控制字符。编辑映射可以保留内部 ID，将目标切换到另一家供应商；用户无需更换 KEY 或模型配置。映射支持搜索、分页、新增、编辑、停用和删除。

新增或编辑模型映射时，选择供应商后点击“获取上游模型”。后端使用已保存的地址与密钥池中的可用 KEY 请求 `<baseURL>/models`，仅返回去重排序后的模型 ID；在“外部模型 ID”输入框中选择，也可手动填写。获取操作不会保存映射或返回外部密钥。上游不支持列表或读取失败时仍可手动配置。请求最多等待 15 秒、响应上限 2 MiB，不跟随重定向，不回传上游错误正文。

管理员在“角色与权限”中为允许使用 AI 的角色勾选“使用 AI”（`ai:use`），再为用户分配该角色。有权限的用户进入“我的资源”，只填写名称即可生成本人 KEY。访问密钥只绑定用户，可访问所有已启用供应商下已启用的全局模型映射。每个用户最多持有 10 个有效密钥，撤销可释放名额，重新生成沿用原名额。

完整访问密钥仅在本人生成或重新生成时返回一次，之后列表不返回密钥片段，仅显示名称、所属用户和使用信息。重新生成会使旧值失效；撤销不可恢复；停用用户也会阻止其访问密钥的新请求。已经进入转发流程的请求可能继续完成。资源或用户重新启用后，其尚未撤销的 KEY 恢复可用。

客户端将 API 基础地址设置为本系统的 `/ai/v1`，使用访问密钥作为 `Authorization: Bearer <访问密钥>`。`GET /models` 由系统本地返回所有已启用映射及其已启用供应商对应的内部模型 ID，不请求或暴露上游模型列表。`POST /chat/completions`、`POST /embeddings`、`POST /responses` 的 JSON 请求必须包含内部模型 ID `model`；系统根据全局映射选择供应商，将此字段替换为对应外部模型 ID，再发送给上游，其他请求字段保持原意，支持 SSE 流式转发。未配置的 ID（包括未映射的外部模型名）返回 400，不转发；停用供应商只阻止发往该供应商的请求，停用或删除映射只影响该模型，KEY 仍可调用其他可用模型。上游响应继续直接转发，实际能力取决于上游服务。访问密钥不能用来登录后台或调用管理接口。

```bash
curl http://localhost:8080/ai/v1/chat/completions \
  -H "Authorization: Bearer $MINIERP_AI_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"chat-default","messages":[{"role":"user","content":"你好"}],"stream":true}'
```

外部 KEY 使用 AES-GCM 加密保存，主密钥首次使用时生成在数据目录的 `.ai-master-key` 文件中，与 JWT 密钥独立。备份或迁移时需一并保留此文件与数据库；丢失主密钥后，原外部 KEY 无法解密，需要重新录入。访问密钥使用随机值，仅保存 SHA-256 摘要。接口不返回加密内容或完整外部 KEY，也不记录提示词、生成内容和原始上游错误正文。

每次代理请求都检查 KEY、所属用户、目标供应商和模型映射的启用状态，以及用户当前角色是否仍包含 `ai:use`。撤回角色权限或移除用户角色后，已有 KEY 的新调用和重新生成均被拒绝；重新授予权限后，未撤销的 KEY 恢复可用。没有 AI 权限的用户仍可查看和撤销自己的 KEY。首次初始化时为系统管理员配置全部权限，普通角色的权限由管理员分配。全局默认最多并发转发 32 个请求，可用 `ai.max_concurrent_requests` 调整；超限返回 429 和 `Retry-After: 1`，不无限排队。默认等待上游首个响应或连续无数据传输超过 2 分钟时中止，可用 `ai.idle_timeout` 调整；持续输出的请求没有固定总时限。客户端停止读取时，下行写入也受此超时约束，避免长期占用并发名额。请求体上限 8 MiB。普通调用次数统计按秒批量落盘，本地模型列表和被拦截的请求不计入。供应商、模型映射和 KEY 列表使用通用分页格式，默认每页 20 条、最大 100 条，前端每页 10 条。

额度耗尽或限流响应会向调用方返回经过校验的 `Retry-After`；后续请求因所有密钥处于冷却期而收到 503 时，也会返回距最早可试用时间的等待秒数。客户端应遵守等待提示并采用退避重试。流式响应设置 `X-Accel-Buffering: no`，供支持此响应头的反向代理关闭缓冲；代理不额外生成 SSE 事件。

### AI 周额度

在“模型映射”设置每百万输入与输出 Token 的整数积分单价、单次固定预留。在“资源分配”设置用户周额度，按需填写原因补发本周积分，配置完毕后启用额度管理。功能默认关闭，升级时新增表和默认字段，保留已有用户、密钥和映射。之前的转发次数不补算积分。

周周期从北京时间周一 00:00 开始。补发只在本周有效，修改周额度立即生效并用于后续周期。每个用户的访问密钥共用余额。余额不足返回 HTTP 403 `insufficient_quota`，未定价模型返回 HTTP 403 `ai_model_unpriced`，模型列表查询免费。

“我的资源”和“资源分配”都显示跨周累计用量，汇总同一账号下所有密钥的结算记录。累计 Token 只计上游报告的用量，累计已用积分包含按预留结算的请求。已释放和未结算的请求不计入累计值，未启用额度管理期间的请求没有结算记录可供统计。接口的 `totalUsage` 使用十进制整数字符串返回 `inputTokens`、`outputTokens` 和 `chargedMicros`。

扣除的积分等于“输入 Token × 输入单价 + 输出 Token × 输出单价”除以一百万。嵌入只计输入，缓存输入采用同价。例如输入和输出单价为 1,000 和 4,000，输入 2,000 Token、输出 500 Token 时扣 4 积分。余额和明细的 `*Micros` 字段是十进制整数字符串，一百万微积分等于一积分。模型映射的可选 `pricing` 包含 `inputPerMillion`、`outputPerMillion`、`reservePoints` 和 `skipStreamUsage`。直接填写两项单价和预留积分即可定价。正数预留表示已定价，清零两项单价和预留可移除价格。

代理在转发前预留，完成后按上游用量结算，预留与单价归入发起请求时的周次。实际用量大于预留可使余额变负，后续请求在余额恢复后才可发出。未发出或明确被上游拒绝的请求释放预留，已发出但用量缺失时按预留结算并标记。默认主动向聊天流请求用量，可针对不支持该参数的模型关闭。每次代理请求只结算一次，供应商密钥切换共用一笔预留。账本同步存入 SQLite，每个数据目录运行一个服务实例，重启时恢复未完成记录。

## 配置

每个配置项都有内置默认值。配置来源按“命令行参数 > 环境变量 > TOML 文件 > 内置默认值”覆盖；未提供配置文件时，程序尝试读取当前工作目录下的 `./config.toml`，文件不存在就继续使用默认值。若显式指定了一个不存在的配置文件，程序会报错。相对数据目录按程序启动时的当前工作目录解析。

| 配置项 | 命令行参数 | 环境变量 | TOML 字段 | 默认值 |
|---|---|---|---|---|
| 配置文件路径 | `-c`、`--config` | `MINIERP_CONFIG_FILE` | 无 | `./config.toml`；文件可不存在 |
| HTTP 监听地址 | `-a`、`--addr` | `MINIERP_ADDR` | `server.addr` | `0.0.0.0:8080` |
| 数据目录 | `-d`、`--data-dir` | `MINIERP_DATA_DIR` | `data.dir` | `./data`；数据库文件为其中的 `mini-erp.db` |
| JWT 密钥 | `-s`、`--jwt-secret` | `MINIERP_JWT_SECRET` | `auth.jwt_secret` | `minierp-insecure-default-secret-change-me` |
| JWT 有效期 | `-t`、`--jwt-ttl` | `MINIERP_JWT_TTL` | `auth.jwt_ttl` | `168h`（7 天） |
| AI 无数据超时 | `--ai-idle-timeout` | `MINIERP_AI_IDLE_TIMEOUT` | `ai.idle_timeout` | `2m`；持续输出不会因总时长被中断 |
| AI 并发上限 | `--ai-max-concurrent-requests` | `MINIERP_AI_MAX_CONCURRENT_REQUESTS` | `ai.max_concurrent_requests` | `32`；超限返回 429 |

运行 `minierp --help` 可查看短参数和长参数。未设置 `MINIERP_JWT_SECRET` 时，每次启动都会打印警告和实际密钥来源，但不会打印密钥本身；正式部署建议设置该环境变量。命令行参数可能出现在系统进程列表里，因此密钥最好从环境变量提供。

`config.example.toml` 展示了地址、数据目录和 Token 有效期的写法。把它复制为 `config.toml` 后再编辑；真实配置文件已加入 `.gitignore`。例如：

```toml
[server]
addr = "0.0.0.0:8080"

[data]
dir = "./data"

[auth]
jwt_ttl = "168h"
# jwt_secret = "至少 32 字节的随机密钥；正式环境优先使用环境变量"

[ai]
idle_timeout = "2m"
max_concurrent_requests = 32
```

Vite 开发服务器默认监听 `5173`，并把 `/api` 转发给 `http://127.0.0.1:8080`；可用 `VITE_API_TARGET` 修改转发目标。开发脚本会启动 Go 和 Vite 两个进程，便于热更新；发布后的单文件程序只启动 Go 服务。

## 数据库、测试和构建

当前处于早期开发阶段，仅使用 GORM `AutoMigrate` 从空库建立当前表结构，并初始化权限清单，不提供旧数据迁移或兼容逻辑。表结构发生不兼容变化后，请先停止程序，删除数据目录中的 `mini-erp.db`（以及同名的 `-wal`、`-shm` 文件，如存在），再重启并重新初始化管理员账号。

数据库采用 WAL 模式，并分离读写连接：写连接保持单连接串行提交，API 查询使用最多 8 个独立的只读连接，避免统计更新或账号修改阻塞其他页面读取。JWT 和用户访问密钥的权限检查均合并为一条关联查询，每次请求直接读取已提交数据，不缓存权限，因此停用账号、撤回角色权限和撤销密钥仍立即生效。超过 200ms 的 SQL 会记录警告，日志只包含参数占位符，不输出密钥、密码等参数值。

AI 调用次数和最近使用时间先合并到内存，每秒批量落盘，程序正常退出时会等待正在处理的请求并刷新剩余统计。这些统计不参与计费或权限判断，页面读到的持久化统计通常最多滞后约一秒；数据库忙或异常时保留批次重试，异常退出可能丢失尚未落盘的少量统计。密钥额度、启用状态和权限变更仍同步保存。供应商调度锁按供应商隔离，成功的正常请求不会为统计写入等待数据库。

代理响应的 `Server-Timing` 包含 `gateway`（本地鉴权、查询、选钥等开销）、`upstream_headers`（等待上游响应头及重试错误正文的累计时间）和 `attempts`（上游尝试次数），便于区分本地和上游耗时。它不包含响应头发出后等待流式首个 token 的时间。上游使用共享连接池，每台主机最多保留 32 个空闲连接，减少并发后的重复 TCP/TLS 握手。

AI 代理使用无数据传输超时，持续输出不会被固定的 2 分钟总时限切断。每次请求返回独立的 `X-Request-ID`，响应中断时，日志 `ai proxy stopped` 会记录对应的 `request_id`、阶段、原因和耗时：`client_canceled` 表示客户端取消，`gateway_idle_timeout` 表示等待上游数据超时，`downstream_write_timeout` 表示客户端长时间不读取，`upstream_truncated` / `upstream_read_failed` 表示上游正文传输中断，`response_write_failed` 表示其他下行写入失败。已开始发送的响应会中止连接或流，不追加 JSON、不自动重放请求；中止信号不会再打印整段请求头。响应头发出前触发无数据超时则返回 HTTP 504。

并发回归使用本地模拟供应商，覆盖 HTTP/1 和同一 HTTP/2 连接上的 32 路流式请求、33 路时的限流响应、取消后名额释放、其他请求继续输出、持续输出超过多个空闲超时周期、上游无响应、客户端不读取，以及冷却期等待提示。可运行 `go test ./internal/transport/http/handler -run 'TestAIProxyConcurrent|TestAIProxyIdle|TestAIProxyCooldown|TestAIProxyClientNotReading' -count=1 -v`，不会调用真实供应商。

本机临时数据库和即时响应模拟上游的一次对比中：单客户端请求中位耗时从约 2.7ms 降到 1.0ms；16 个并发客户端从约 41.2ms 降到 5.0ms，P95 从约 54.9ms 降到 9.2ms。持有写事务 300ms 时，用户列表读取从等待约 300.7ms 变为约 1.1ms。每次正常代理请求从 9 次读取、2 次同步统计写入，减少到 3 次读取且无同步统计写入。上述为本地开销对比，不代表真实供应商的生成速度。可运行 `go test ./internal/transport/http/handler -run TestLatencyDiagnostic -count=1 -v` 复测，不会调用真实供应商。

SQLite 使用纯 Go 驱动 `github.com/libtnb/sqlite`，避免 CGO 和额外的 C 编译器要求。构建脚本设置 `CGO_ENABLED=0`，每次只生成一个目标的程序，默认使用当前主机的系统和架构。通过目标参数可交叉编译 Windows x64（`windows/amd64`）、Linux x64（`linux/amd64`）、macOS Intel（`darwin/amd64`）或 macOS Apple Silicon（`darwin/arm64`）等目标；具体支持范围还取决于所用驱动和系统资源采集库。

开发环境需要 Go 1.27 或更新版本、Node.js 22 或更新版本和 npm。常用命令：

```powershell
# Windows：首次使用或依赖变化后，先安装依赖
.\scripts\install-deps.ps1

# Windows：启动开发环境
.\scripts\dev.ps1

# Windows：运行前端测试和后端测试
cd frontend; npm test
cd ..\backend; go test ./...

# Windows：仅构建当前平台
cd ..; .\scripts\build.ps1

# Windows：指定单个目标
.\scripts\build.ps1 -Target linux/amd64
.\scripts\build.ps1 -Target darwin/arm64
```

Linux/macOS 首次使用或依赖变化后先运行 `./scripts/install-deps.sh`，再使用 `./scripts/dev.sh` 或 `./scripts/build.sh`；前端测试命令为 `cd frontend && npm test`，后端测试命令为 `cd backend && go test ./...`。发布产物位于 `release/`，目标机器运行程序时不需要安装 Node.js、CGO 或 SQLite。

依赖安装脚本统一执行前端 `npm ci` 和后端 `go mod download`。构建和开发启动脚本不再安装依赖；缺少前端依赖时会直接提示安装。Go 构建使用 `GOPROXY=off` 和 `-mod=readonly`，缺少缓存模块时也会失败并提示先运行安装脚本，不会自动下载模块或修改依赖声明。更新 `package-lock.json`、`go.mod` 或 `go.sum` 后，请重新执行依赖安装脚本。

Linux/macOS 通过 `./scripts/build.sh --target linux/amd64` 指定目标；不传参数时仅构建当前平台。两种脚本均会在构建前验证目标参数，帮助信息分别使用 `-Help` 和 `--help` 查看。构建不会清理 `release/` 中以前生成的其他目标程序。

构建完成后，在项目根目录运行 Windows 版本：

```powershell
.\release\minierp-windows-amd64.exe
```

访问 `http://localhost:8080`，首次进入会引导创建管理员。Linux 运行 `./release/minierp-linux-amd64`；macOS 按芯片选择 `minierp-darwin-amd64` 或 `minierp-darwin-arm64`。如果脚本或程序没有执行权限，先用 `chmod +x` 设置，或通过 `bash ./scripts/build.sh` 执行构建脚本。

分发时只需复制对应平台的可执行文件；不需要附带 `frontend/dist/` 或 `static/`。运行时会在当前目录下创建 `data/mini-erp.db`，可用 `--data-dir` 指定其他数据目录。真实配置和已有数据请在更新程序时保留。

测试覆盖：配置默认值、错误输入和覆盖顺序；AutoMigrate 和数据库读写；并发初始化、密码哈希、Token 失效、权限合并、最后管理员保护和角色删除限制；HTTP 参数、响应状态和权限校验；前端初始化、登录、页面权限及表单。

## 首次运行和安全说明

首次启动后访问程序地址，页面会引导创建首位系统管理员。此后只有有相应权限的用户才能创建用户和角色。数据库会保存在配置的数据目录中；备份时需要备份这个目录，升级或替换可执行文件不会自动迁移或删除它。

固定 JWT 默认密钥可从源码或程序中提取，掌握它的人可能伪造 Token。正式部署时应通过 `MINIERP_JWT_SECRET` 设置随机密钥。JWT 默认有效期为 7 天，浏览器凭证保存在 `localStorage`；密码重设或停用账号会让该账号已有 Token 失效。
