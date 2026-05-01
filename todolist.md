# BTDown_MA 当前计划与进度（2026-04-30）

## 一、当前总体阶段判断

项目已经完成「桌面应用骨架 + 后端主链路雏形 + 基础流媒体通路」，当前整体处于 **Phase 2 与 Phase 3 之间**：
- Phase 0（骨架）大部分已完成
- Phase 1（下载引擎 MVP）主流程已接通，但会话控制能力未闭环
- Phase 2（HTTP Range 流媒体壳子）核心能力已落地
- Phase 3（P2P 数据源替换）核心能力已落地
- Phase 4+（播放器联动、调度增强、观测与产品化）尚未进入实做

---

## 二、对照原计划的阶段进度

## Phase 0：项目骨架
- [x] Go 后端工程骨架可运行（`cmd/desktop/main.go` + `internal/bootstrap/application.go`）
- [x] React 前端骨架与页面结构已建立（`frontend/src/pages/desktop/DesktopHomePage.tsx`）
- [x] Wails Binding 接口雏形已建立（`internal/wails/app_bindings.go` + `frontend/src/shared/runtime/wailsSessionBridge.ts`）
- [x] 基础配置模型已建立（`internal/config/application_config.go`）
- [x] 日志与配置管理已完整（播放器路径、自动清理与设置读写已落地）
- [ ] AI CLI 扩展位未建立独立模块（仅计划层提及）

状态：**80%**

## Phase 1：底层下载引擎 MVP
- [x] `AddMagnet -> Wait GotInfo -> 选主文件 -> Download` 主流程已实现（`internal/service/torrent_runtime_manager.go`）
- [x] metadata 超时控制已实现（`metadataResolveTimeout`）
- [x] 指标刷新机制已实现（peer、速度、下载字节）
- [x] 会话创建/查询能力已通过 HTTP + Binding 暴露（`session_controller.go` / `app_bindings.go`）
- [x] 会话停止、清理等生命周期能力已实现
- [ ] “完整落盘验证链路”未看到自动化验证

状态：**75%**

## Phase 2：本地 HTTP 流媒体壳子
- [x] 基于 HTTP 服务暴露 `/api/v1/streams/:sessionId`（`internal/stream/http_stream_server.go`）
- [x] 使用 `http.ServeContent` 支持 Range/206（`internal/controller/stream_controller.go`）
- [x] 多播放器 Range 行为日志验证已落地
- [ ] UI 对 stream URL 与播放状态联动仍偏展示层

状态：**60%**

## Phase 3：P2P 数据源替换本地文件
- [x] Stream 数据源已使用 `selectedFile.NewReader()`（`torrent_runtime_manager.go`）
- [x] 主文件识别策略已实现（扩展名优先 + 最大文件兜底，`session_file_selection.go`）
- [x] 未完整下载即可读取的链路已实现（reader 阻塞读取 + responsive + readahead）
- [x] 状态机已同步到会话模型（`session_state_machine.go` + `session.go`）
- [ ] 多文件复杂选择策略与 UI 选择能力未实现（当前默认自动选择）

状态：**85%**

## Phase 4：外部播放器联动
- [x] 播放器路径配置与持久化已实现（设置页已接真实配置读写）
- [ ] 一键启动外部播放器未实现（后端仅有 URL 构建）
- [ ] 启动失败诊断与兜底策略未实现
- [ ] 复制 stream URL 快捷能力未实现

状态：**10%（仅结构预留）**

## Phase 5：优先级调度器
- [x] 文件高优先级 + 头尾预热已实现（`SetPriority` + `DownloadPieces`）
- [ ] 滑动窗口策略未实现
- [ ] Seek 抢占策略未实现
- [ ] `SubscribePieceStateChanges()` 事件驱动未实现

状态：**30%**

## Phase 6：观测与性能
- [ ] pprof 未接入
- [x] metadata 慢因诊断、peer 计数、下载速度、useful bytes 增量与 Range 响应耗时已打点并展示
- [x] BT 网络基础诊断已接入（原始/追加 tracker、torrent public/private、监听端口、DHT/UDP/入站连接）
- [x] 深度健康检测已接入（快速/标准/深度窗口、置信度、证据列表、四档可用性）
- [x] 第一帧时间 / seek 恢复时间 / 缓冲命中率已打点
- [ ] 慢节点治理与 deadline 策略未实现

状态：**70%**

## Phase 7：Windows 产品化与扩展预留
- [x] 设置页已接真实后端配置
- [ ] 会话清理策略（释放 torrent/reader/订阅）未完整实现
- [ ] 集成/回归测试体系未建立
- [ ] 打包分发方案未落地
- [ ] 诊断导出与 AI CLI 扩展未实现

状态：**10%（页面占位）**

---

## 三、当前代码已具备的关键能力

- [x] 会话创建与列表查询
- [x] Magnet 加载、metadata 等待、主文件选择
- [x] 流媒体 URL 构建与 HTTP Range 拉流
- [x] 基础会话状态机与指标刷新
- [x] 前后端基础连接（HTTP + Wails Bridge 双路径）

---

## 四、当前最关键缺口

- [ ] 会话生命周期闭环：停止、清理、资源释放策略
- [ ] 外部播放器真实联动（路径配置、启动、失败可观测）
- [ ] 调度器从“预热”升级到“请求驱动窗口 + seek 抢占”
- [ ] 观测体系（pprof + 首播/seek 指标）
- [x] 前端设置页从占位数据切换为真实配置读写

---

## 五、下一阶段执行 Todo（建议按顺序）

- [x] 1. 完成会话控制闭环：`StopSession`、`CleanupSession`、资源安全释放
- [ ] 2. 完成外部播放器最小可用链路：路径配置 + 一键启动 + 错误提示
- [ ] 3. 实现复制 stream URL 能力，作为播放器兼容兜底
- [ ] 4. 实现请求驱动调度 V1：滑动窗口 + seek 抢占
- [x] 5. 增加关键观测指标：首播时间、seek恢复时间、metadata耗时
- [x] 6. 将设置页接入后端真实配置接口，替换当前占位值
- [ ] 7. 为主流程补最小集成测试（创建会话 -> 获取流 -> 状态变化）

---

- [x] 可观测总览已接入管理界面（会话总数、状态分布、吞吐与最近流访问）

本文件依据当前仓库代码与 `docs/bt-streaming-mvp-todolist.md` 交叉整理，属于“当前实现快照”。
后续每完成一个里程碑，建议同步更新本文件，避免计划与代码状态脱节。
