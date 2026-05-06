# BT 边下边播 Windows 桌面应用实现计划

> 文档定位：
> 本文件保留项目最初的总体方案、架构判断和历史 Todo 基线。
> 它仍有参考价值，但**不再作为当前进度权威文档**。
>
> 当前状态请优先阅读：
> - `../../todolist.md`
> - `../streaming-execution-roadmap.md`
> - `../download-speed-analysis-plan.md`

## 1. 结论先行

当前产品目标已经明确，不再是“Go 流媒体原型服务”或“CLI 优先工具”，而是一个以 **Windows 桌面应用** 为目标形态的 BT 边下边播产品。

推荐技术路线：

1. 使用 `github.com/anacrolix/torrent` 作为 P2P 引擎，而不是自研 BT/DHT/Tracker/Peer Wire 协议。
2. 使用 **Wails + React + Go** 作为应用基础框架：
   - Wails 负责桌面应用壳与前后端桥接
   - React 负责图形化操作界面
   - Go 负责 torrent、调度、流媒体、状态管理与系统集成
3. 控制通信统一走 **Wails JS Bindings**，用于：
   - 发起 magnet 任务
   - 查询会话状态
   - 接收下载进度与错误信息
   - 触发播放、暂停、清理等操作
4. 媒体通信保留 **本地 HTTP Range Stream**，由 Go 提供标准 `Range`/`206 Partial Content` 能力，供外部播放器消费。
5. 首版播放器形态固定为 **外部播放器**，应用负责启动播放器并向其传入本地 stream URL。
6. 预留 **AI CLI 调用扩展位**，用于后续接入智能诊断、批量操作、脚本化控制或自动化分析，但当前阶段不实现具体能力。

一句话判断：
首版不要同时做“自研 BT 协议栈 + 应用内视频播放器 + AI 自动控制”。当前最优路径是：**先把 Wails 桌面壳、Go 下载核心、外部播放器播放链路打通，再逐步补调度和扩展能力。**

## 2. 为什么这样实现

### 2.1 已确认的库能力

- `anacrolix/torrent` 官方文档明确支持“按需下载 / streaming”。
- `Torrent.NewReader()` 返回阻塞式 `ReadSeekCloser`，读取请求的数据未到齐时会等待。
- `Reader.SetReadahead(...)` 和 `Reader.SetResponsive()` 可调流式读取行为。
- `File.SetPriority(...)`、`Piece.SetPriority(...)`、`Torrent.DownloadPieces(...)` 可做优先级提升。
- `Torrent.SubscribePieceStateChanges()` 可订阅 piece 状态变化，用于事件驱动唤醒。
- 官方相关项目 `anacrolix/confluence` 本身就是“Torrent client as a HTTP service”，并明确写明：
  - 支持 HTTP byte range requests
  - response 会阻塞直到数据可用

### 2.2 已确认的桌面集成路径

- Wails 适合构建 Windows 桌面应用，并提供 Go 与前端之间的双向调用能力。
- React 适合快速构建会话列表、下载状态、播放器启动按钮、日志视图等操作界面。
- Wails JS Bindings 适合承担控制面通信，避免为了 UI 控制额外引入本地 REST API。
- 外部播放器天然更适合消费标准 HTTP Range 流，能减少应用内解码、格式兼容、拖拽处理、播放器状态同步等复杂度。

### 2.3 已确认的 HTTP 媒体能力

- Go 官方 `net/http.ServeContent` 原生支持：
  - `Range`
  - `If-Range`
  - `206 Partial Content`
  - 基于 `io.ReadSeeker` 的 seek/长度探测
- 因此媒体面首版不必手写完整 Range 协议栈，仍可优先复用标准库。

### 2.4 关键现实约束

- 真正困难不在“返回 206”，而在“播放器请求的字节还没下完时，如何尽快把正确区域提权并减少卡顿”。
- 不同外部播放器的 Range 请求行为并不完全相同，常见行为是：
  - 先探测头部
  - seek 到尾部读取索引/容器信息
  - 再回到中间或当前位置附近
- 如果视频文件不是 `faststart`，仅优先下载首段通常不够。
- 桌面产品还需要额外考虑：
  - Windows 路径与权限处理
  - 外部播放器启动方式与参数拼接
  - 后台服务生命周期与 UI 生命周期的协调

## 3. 方案对比

### 方案 A：直接自研完整 BT + 桌面 GUI + 播放桥接

优点：

- 完全可控
- 理论上可以做极限优化

缺点：

- 开发量过大
- GUI、torrent、流媒体、播放器集成、调度器会相互耦合
- 不适合当前空仓启动

结论：不选。

### 方案 B：以 `anacrolix/torrent` 为底座，Wails 做桌面壳，外部播放器消费本地 HTTP 流

优点：

- 最快形成可运行的 Windows 产品原型
- 复用成熟协议栈
- 控制面与媒体面边界清晰
- UI 与核心下载逻辑职责分离
- 外部播放器兼容性和格式支持更成熟

缺点：

- 需要处理播放器发现、启动、路径配置与失败回退
- 某些播放器行为差异要通过实测适配
- 高级播放状态同步能力不如内嵌播放器细粒度

结论：首选。

### 方案 C：直接做应用内播放器

优点：

- 产品观感更完整
- 理论上更容易做统一交互体验

缺点：

- 前端解码、容器兼容、seek 行为、播放器控件、字幕支持都会显著增加复杂度
- 会把当前重点从“边下边播能力验证”转移到“播放器 UI 工程”

结论：暂不采用，后续如有必要再评估。

## 4. 推荐架构

```text
Windows Desktop App (Wails)
   |
   +--> React Frontend
   |      |- Magnet 输入
   |      |- 会话列表
   |      |- 下载进度与速率展示
   |      |- 播放启动与错误提示
   |      |- 日志/状态/设置页面
   |
   +--> Wails JS Bindings
   |      |- StartSession
   |      |- ListSessions
   |      |- GetSessionStatus
   |      |- OpenInPlayer
   |      |- StopSession
   |      |- CleanupSession
   |      |- SubscribeProgress（概念层）
   |
   +--> Go Application Core
          |- Session Manager
          |- Torrent Client(anacrolix/torrent)
          |- Metadata Resolver
          |- Priority Controller
          |- HTTP Stream Server
          |- External Player Launcher
          |- Metrics/Logs
          |
          +--> AI CLI Adapter (预留扩展位)
                 |- 统一命令入口抽象
                 |- 会话诊断/状态导出扩展点
                 |- 自动化脚本调用扩展点
```

### 控制面与媒体面分离

- **控制面**：React 通过 Wails JS Bindings 直接调用 Go 方法。
- **媒体面**：外部播放器通过本地 HTTP URL 拉流。
- **边界原则**：
  - UI 不直接处理 torrent 协议细节
  - 播放器不直接参与控制逻辑
  - Go 核心统一管理 session、调度、流媒体和播放器启动

### 建议模块拆分

- `frontend/`
  - React 前端页面
  - 会话、设置、日志、状态组件
- `app.go` / `wails bindings`
  - Wails 绑定入口
  - UI 调用的应用服务门面
- `internal/app`
  - 生命周期管理、依赖装配、配置加载
- `internal/engine`
  - torrent client、session、metadata、文件识别
- `internal/stream`
  - HTTP handler、reader 包装、range/seek 观测
- `internal/scheduler`
  - 首尾预热、滑动窗口、seek 提权、事件订阅
- `internal/player`
  - 外部播放器发现、启动、参数构建、错误处理
- `internal/metrics`
  - 下载速度、peer 数、buffer 命中、seek 延迟
- `internal/aicli`
  - AI CLI 扩展接口定义与适配层占位

## 5. 核心设计决策

### 5.1 Wails JS Bindings 优先于本地控制 API

首版控制面不额外暴露本地 REST API，而是通过 Wails 绑定直接完成 UI 与 Go 核心通信。

这样做的好处：

- 减少控制面协议设计负担
- 降低桌面应用本地接口暴露面
- 更适合 Windows 单机产品形态
- 便于后续把 UI 操作收敛成强类型方法调用

### 5.2 本地 HTTP Stream 只承担媒体职责

本地 HTTP 服务继续保留，但只用于外部播放器访问媒体流。

这样做的好处：

- 保持对播放器生态的兼容性
- 复用标准 `Range`/`206` 能力
- 避免把控制请求和媒体请求混在一起
- 更容易单独观测播放器 Range 行为

### 5.3 外部播放器优先于内嵌播放器

首版只支持“由应用启动外部播放器”这一条主路径。

这样做的好处：

- 最大化复用 VLC / MPV / PotPlayer 等现成能力
- 降低前端复杂度
- 让研发重点留在 BT 调度和流媒体稳定性

首版至少需要支持：

- 配置播放器可执行文件路径
- 自动拼接 stream URL 与必要启动参数
- 启动失败时在 UI 中给出明确错误

### 5.4 Reader 优先于“手写挂起队列”

首版继续利用 `torrent.Reader` 的阻塞读取特性。

这样做的好处：

- 少写大量并发同步代码
- 天然兼容 `ServeContent`
- 后续只需要在 `Seek`/`Read` 前后做优先级调整

### 5.5 优先级控制要从“固定首尾”升级到“请求驱动”

至少分三层：

1. 启动预热层
   - metadata 到手后，优先下载主视频文件的前若干 MB
   - 同时预热尾部若干 MB，兼容非 faststart MP4
2. 播放窗口层
   - 维护“当前位置之后 N MB”的滑动窗口
3. seek 抢占层
   - 一旦收到新的大跨度 `Range`，立刻重设窗口并提高优先级

### 5.6 AI CLI 只预留扩展接口，不前置实现

当前阶段只做抽象边界，不做真实集成。

建议预留能力：

- 统一命令执行入口
- 会话状态导出
- 日志摘要导出
- 批量任务控制
- 未来接入 AI 助手或脚本工具的适配层

避免事项：

- 不要在首版把 AI CLI 变成关键路径
- 不要把业务逻辑直接耦合到某个具体 CLI 工具

## 6. TodoList

## Phase 0：项目骨架

- [ ] 初始化 Wails 项目，确认 Windows 下开发/构建链路可运行。
- [ ] 建立 React 前端骨架与基础页面布局。
- [ ] 建立 Go 模块，确定目录结构与配置模型。
- [ ] 打通 Wails JS Bindings 最小链路：前端按钮调用 Go 方法并返回结果。
- [ ] 建立最小日志与配置模型：下载目录、数据目录、stream 监听地址、播放器路径、自动清理开关。
- [ ] 设计 session 概念：一个 magnet 对应一个播放会话。
- [ ] 预留 AI CLI 扩展包与接口定义。

验收标准：

- 能启动空白桌面应用。
- 前端可成功调用 Go 方法。
- 配置和日志输出可读。

## Phase 1：底层下载引擎 MVP

- [ ] 实现 `AddMagnet -> Wait GotInfo -> 选择文件 -> 下载` 的最小链路。
- [ ] 打印实时指标：
  - 总下载速度
  - peer 数
  - 已完成字节
  - metadata 获取耗时
- [ ] 支持指定下载目录并验证文件可完整落盘。
- [ ] 加入基础超时与取消控制，避免僵尸会话。
- [ ] 通过 Wails Bindings 暴露会话创建、查询和停止方法。

验收标准：

- 在 UI 中输入磁力链接后能稳定拿到 metadata。
- 能开始下载并持续输出速度。
- 能完整下载一个测试种子。

## Phase 2：本地 HTTP 流媒体壳子

- [ ] 用本地现成 MP4 文件先实现 stream handler。
- [ ] 基于 `http.ServeContent` 跑通 `Range`/`206`/拖拽播放。
- [ ] 用至少两种外部播放器验证（如 VLC、MPV 或 PotPlayer）。
- [ ] 记录不同播放器的实际 Range 行为日志。
- [ ] 在 UI 中展示 stream URL 和播放启动状态。

验收标准：

- 外部播放器可稳定播放本地样本文件。
- 拖拽后不会崩溃或直接断流。
- 服务端能记录每次 Range 请求。

## Phase 3：P2P 数据源替换本地文件

- [ ] 将 stream 数据源切换为 `torrent.File.NewReader()`。
- [ ] 在 metadata 完成后自动识别“主视频文件”。
- [ ] 首版只支持单文件播放，先不处理复杂多文件选择 UI。
- [ ] 验证播放器在未完成下载状态下能开始播放。
- [ ] 将关键状态同步到前端：等待 metadata、准备播放、正在播放、错误。

验收标准：

- 输入 magnet 后可返回一个本地 stream URL。
- 外部播放器连接后，数据未完整下载时仍可开始读取。
- 前端能看到状态流转。

## Phase 4：外部播放器联动

- [ ] 实现外部播放器路径配置与保存。
- [ ] 实现“使用外部播放器播放”按钮。
- [ ] 根据播放器类型拼接启动参数。
- [ ] 处理播放器启动失败、路径无效、URL 无法访问等错误。
- [ ] 增加“复制 stream URL”能力，作为调试和兼容兜底手段。

验收标准：

- 用户可在桌面应用中一键启动外部播放器。
- 启动失败时错误可定位。
- 常见播放器可正常消费 stream URL。

## Phase 5：优先级调度器

- [ ] 实现文件级优先级提升：目标视频文件设为 `High`。
- [ ] 实现启动预热：首段 + 尾段。
- [ ] 实现滑动窗口：围绕当前播放位置提权。
- [ ] 实现 seek 检测：收到新的大跨度 `Range` 后重设窗口。
- [ ] 使用 `SubscribePieceStateChanges()` 做事件驱动刷新。
- [ ] 将首播时间、seek 恢复时间等关键指标暴露到前端。

验收标准：

- 首播时间显著短于“全文件自然下载后再播”。
- seek 后能在可接受时间内恢复播放。

## Phase 6：观测与性能

- [ ] 接入 `pprof`，抓取 CPU、heap、alloc、goroutine。
- [ ] 打点关键指标：
  - metadata 耗时
  - 第一帧时间
  - seek 恢复时间
  - 缓冲命中率
  - 当前窗口覆盖字节
- [ ] 加入 peer 慢节点淘汰与 deadline 控制。
- [ ] 基于证据判断是否引入 `sync.Pool`。

验收标准：

- 能回答“卡在哪”：找源慢、调度慢、磁盘慢、还是 GC 慢。

## Phase 7：Windows 产品化与扩展预留

- [ ] 完善设置页：下载目录、播放器路径、日志目录、清理策略。
- [ ] 设计会话清理策略：会话关闭后释放 reader、torrent、订阅与临时状态。
- [ ] 增加集成测试与回归测试。
- [ ] 评估 Wails Windows 打包、安装与分发方式。
- [ ] 预留 AI CLI 扩展配置项和调用入口，但不实现具体功能。
- [ ] 增加诊断导出能力，为后续 CLI/AI 自动分析提供输入材料。

验收标准：

- 会话结束后无明显 goroutine 泄漏。
- 错误信息可定位。
- 桌面应用具备可交付的 Windows 基础形态。

## 7. 重点风险

### 风险 1：视频容器与播放器行为不一致

影响：

- 有的文件必须先拿尾部索引
- 有的播放器会发多个试探性 Range
- 不同播放器参数和容错行为不同

应对：

- 必须记录真实 Range 访问日志
- 不要只靠“首块优先”
- 优先支持少量主流播放器，避免一次性铺太广

### 风险 2：默认调度不足以支撑 seek

影响：

- 虽然能播，但拖拽很慢

应对：

- 及早引入 seek 抢占优先级
- 用 piece 状态事件做联动

### 风险 3：播放器启动与系统环境差异

影响：

- 用户机器上未安装播放器
- 路径错误或启动参数不兼容
- Windows 权限或路径转义问题导致启动失败

应对：

- 提供播放器路径配置
- 提供复制 stream URL 兜底
- 将启动命令与错误输出写入日志

### 风险 4：过早做 AI CLI 集成，拖慢总体进度

影响：

- 增加抽象复杂度
- 让主路径失焦

应对：

- 只保留接口边界与配置占位
- 把 CLI 集成放到后续扩展阶段

## 8. 残余不确定性

以下问题目前不能仅靠文档 100% 证实，必须通过原型实测：

1. `anacrolix/torrent` 默认调度在高 seek 频率场景下的实际恢复时间。
2. `Reader.SetResponsive()` 对不同外部播放器体验的真实影响。
3. 是否真的需要自定义 `sync.Pool` 才能达到目标吞吐平稳性。
4. 多文件种子里“主视频文件识别”规则是否需要更复杂策略。
5. 不同播放器在 Windows 下的 URL 启动参数是否需要差异化适配。
6. Wails 前端事件推送在高频进度更新场景下是否需要节流。

## 9. 建议的下一步

如果继续推进，实现顺序建议固定为：

1. 先建 Wails + React + Go 项目骨架。
2. 先做 Phase 1 下载引擎最小版。
3. 再做 Phase 2 本地 MP4 Range 壳子。
4. 然后做 Phase 3 数据源替换。
5. 再做 Phase 4 外部播放器联动。
6. 最后攻 Phase 5 调度器。

## 10. 参考资料

- anacrolix/torrent: https://pkg.go.dev/github.com/anacrolix/torrent
- Go `net/http.ServeContent`: https://pkg.go.dev/net/http#ServeContent
- anacrolix/confluence: https://github.com/anacrolix/confluence
- Wails: https://wails.io/
