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

首版权限固定为 `users:read`、`users:manage`、`roles:read`、`roles:manage`。具有某项 `manage` 权限的用户也能执行对应的 `read` 操作。普通用户可以拥有多个角色，有效权限是这些角色权限的合并。初始化创建的系统管理员账号不可调整角色、不可停用；新建和编辑其他账号时均不可分配系统管理员角色。编辑用户时省略 `roleIds` 表示保留原有角色，传入空数组表示清空普通用户的角色。用户停用后保留记录，不提供硬删除。登录失败时不区分账号不存在、密码错误或账号停用。

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

## 配置

每个配置项都有内置默认值。配置来源按“命令行参数 > 环境变量 > TOML 文件 > 内置默认值”覆盖；未提供配置文件时，程序尝试读取当前工作目录下的 `./config.toml`，文件不存在就继续使用默认值。若显式指定了一个不存在的配置文件，程序会报错。相对数据目录按程序启动时的当前工作目录解析。

| 配置项 | 命令行参数 | 环境变量 | TOML 字段 | 默认值 |
|---|---|---|---|---|
| 配置文件路径 | `-c`、`--config` | `MINIERP_CONFIG_FILE` | 无 | `./config.toml`；文件可不存在 |
| HTTP 监听地址 | `-a`、`--addr` | `MINIERP_ADDR` | `server.addr` | `0.0.0.0:8080` |
| 数据目录 | `-d`、`--data-dir` | `MINIERP_DATA_DIR` | `data.dir` | `./data`；数据库文件为其中的 `mini-erp.db` |
| JWT 密钥 | `-s`、`--jwt-secret` | `MINIERP_JWT_SECRET` | `auth.jwt_secret` | `minierp-insecure-default-secret-change-me` |
| JWT 有效期 | `-t`、`--jwt-ttl` | `MINIERP_JWT_TTL` | `auth.jwt_ttl` | `168h`（7 天） |

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
```

Vite 开发服务器默认监听 `5173`，并把 `/api` 转发给 `http://127.0.0.1:8080`；可用 `VITE_API_TARGET` 修改转发目标。开发脚本会启动 Go 和 Vite 两个进程，便于热更新；发布后的单文件程序只启动 Go 服务。

## 数据库、测试和构建

启动时 GORM `AutoMigrate` 会创建模型所需的表并补充常规字段，权限清单也会在启动时初始化。首版不另写迁移脚本，因为当前只需从空库建立表结构。`AutoMigrate` 不会自动删除模型中已经移除的列；将来需要数据转换或有破坏性表结构调整时，再编写受控迁移。

SQLite 使用纯 Go 驱动 `github.com/libtnb/sqlite`，避免 CGO 和额外的 C 编译器要求。构建脚本设置 `CGO_ENABLED=0`，每次只生成一个目标的程序，默认使用当前主机的系统和架构。通过目标参数可交叉编译 Windows x64（`windows/amd64`）、Linux x64（`linux/amd64`）、macOS Intel（`darwin/amd64`）或 macOS Apple Silicon（`darwin/arm64`）等目标；具体支持范围还取决于所用驱动和系统资源采集库。

开发环境需要 Go 1.27 或更新版本、Node.js 22 或更新版本和 npm。常用命令：

```powershell
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

Linux/macOS 使用 `./scripts/dev.sh` 和 `./scripts/build.sh`；前端测试命令为 `cd frontend && npm test`，后端测试命令为 `cd backend && go test ./...`。发布产物位于 `release/`，目标机器运行程序时不需要安装 Node.js、CGO 或 SQLite。

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
