# BT 下载提速、边下边播、链接存活检测调研报告

调研日期：2026-04-30

## 1. 结论先行

当前项目已经选择 `github.com/anacrolix/torrent` 作为 Go 后端 BT 引擎，这是合理路线，不建议推倒重写协议栈。真正要“提高下载速度 + 边下边播 + 检测 BT 链接死活”，应当重写的是调度策略、健康检测模型、配置暴露和观测能力，而不是重写 BitTorrent 协议。

推荐方案：

1. 继续保留 `anacrolix/torrent`，用它处理 DHT、tracker、PEX、uTP、metadata、piece 下载和文件读取。
2. 下载提速采用“多来源发现 + 端口可达 + tracker 补强 + piece 调度 + 热点区域优先 + 指标反馈”的组合方案。
3. 边下边播采用“HTTP Range + `File.NewReader()` + 动态 readahead + seek 感知 piece 提权”的方案。
4. 链接死活检测不要做成简单 true/false，应做成分级状态：`Unknown`、`ResolvingMetadata`、`Alive`、`Weak`、`NoPeers`、`MetadataTimeout`、`TrackerFailed`、`DeadLikely`。
5. 首版目标不要追求“100% 判断死链”，BT 网络本身不支持一次性确定。更可靠的是在限定时间窗口内综合 metadata、tracker scrape、DHT get_peers、已连接 peer、piece 可下载性给出置信度。

一句话落地：以当前 Go 后端为核心，增加一个 `HealthProbe + SpeedScheduler + StreamPriorityController` 三件套。

## 2. 当前项目现状

从代码看，当前项目已经具备基础能力：

- `go.mod` 已引入 `github.com/anacrolix/torrent`。
- `internal/service/torrent_runtime_manager.go` 已用 `torrent.NewDefaultClientConfig()` 创建 BT client。
- session 启动流程使用 `client.AddMagnet(...)`。
- metadata 获取通过 `torrent.GotInfo()` 等待。
- 主文件选择后调用 `file.SetPriority(torrent.PiecePriorityHigh)` 和 `file.Download()`。
- 播放流通过 `selectedFile.NewReader()` 获取 reader。
- reader 已启用 `SetResponsive()` 和 `SetReadahead(8 << 20)`。
- `internal/controller/stream_controller.go` 使用 `http.ServeContent`，天然支持 HTTP Range/seek。
- 当前死活判断主要依赖 `stats.TotalPeers > 0 || stats.ActivePeers > 0`，这只能作为粗略信号，不足以做准确健康评级。

这说明项目不是“完全没有边下边播”，而是已经有 MVP 基础。下一步应重点补强：

- peer 来源质量。
- 下载窗口调度。
- player seek 后的动态提权。
- tracker/DHT 探测。
- dead/weak/alive 的分级模型。
- 指标日志，便于判断到底是资源死、网络慢、端口不通、tracker 挂了，还是播放器请求模式导致卡顿。

## 3. 开源项目与技术对比

| 项目/技术 | 类型 | 可借鉴点 | 不建议直接照搬的点 |
| --- | --- | --- | --- |
| `anacrolix/torrent` | Go BT library | Go 原生、适合嵌入、支持流式 reader、DHT/tracker/PEX/uTP 等 BT 能力 | 文档较偏 API，需要自己补产品级调度和诊断 |
| `libtorrent-rasterbar` | C++ BT library | 成熟、高性能、qBittorrent/Deluge 等大量客户端使用，piece priority、deadline、状态指标丰富 | Go/Wails 项目接入成本高，需要 CGO/绑定/分发处理 |
| qBittorrent | 桌面 BT 客户端 | 顺序下载、首尾 piece 优先、Web API、基于 libtorrent 的调度经验 | 不是嵌入式库，直接集成会变成外部进程控制 |
| Transmission | 轻量 BT 客户端/daemon | daemon + RPC 模式、轻量、跨平台、适合参考远程控制模型 | 边播调度能力不是它的主要产品重点 |
| aria2 | 下载工具/daemon | 多协议、JSON-RPC、DHT、PEX、多 tracker、适合参考配置项和 RPC 模型 | BT 边下边播不是核心场景 |
| WebTorrent | JS/WebRTC BT 客户端 | 浏览器/桌面流式播放体验、`streamTo`、web peers、WebRTC 思路 | 浏览器端只能连 WebRTC peers，不能直接覆盖传统 TCP/uTP swarm |
| peerflix / popcornflix | Node torrent streaming | “本地 HTTP server + 外部播放器”的经典边播模型 | 多数项目维护活跃度一般，技术栈与当前 Go 后端不一致 |
| BiglyBT | 全功能 BT 客户端 | Swarm Merging、资源健康、WebTorrent/I2P/插件生态、复杂诊断能力 | 体量大，不适合作为当前项目内核 |

对当前项目的最优选择是：核心仍用 `anacrolix/torrent`，参考 `libtorrent/qBittorrent` 的 piece 调度思想，参考 `WebTorrent/peerflix` 的播放链路，参考 `aria2/Transmission` 的配置和 RPC 观测方式，参考 BiglyBT 的健康诊断思路。

## 4. 下载速度提升方案

BT 下载速度不是单一参数能解决的，通常由这些因素共同决定：

- swarm 里有没有足够 peer/seed。
- tracker、DHT、PEX、LSD 是否能找到 peer。
- 本机监听端口是否可从公网或局域网访问。
- 防火墙/NAT 是否阻断 TCP/UDP/uTP。
- 上传带宽是否过低，导致被其他 peer 限制。
- piece 选择是否过度顺序化，影响稀有 piece 获取。
- 磁盘写入和缓存是否跟得上。
- 播放器 seek 请求是否频繁打断下载窗口。

### 4.1 peer 来源补强

建议启用并观测这些来源：

- Tracker announce：从 magnet 自带 tracker 获取 peer。
- DHT get_peers：tracker 不可用时仍可发现 peer。
- PEX：连接到 peer 后继续交换更多 peer。
- LSD：局域网场景可发现本地 peer。
- Web seed：如果 torrent 带 `ws` 或 URL seed，可作为 HTTP 补充来源。
- 手动 tracker 增补：对公开资源可允许用户配置公共 tracker 列表。

落地建议：

- 启动 session 后立即记录 tracker 数量、DHT 状态、peer 来源。
- 对无 tracker 或 tracker 极少的 magnet，自动追加用户配置的公共 tracker 列表。
- 定期 `force announce` 或等价重试，不要只在创建时请求一次。
- 对 private torrent 必须尊重 private flag，不应强行 DHT/PEX。

### 4.2 端口与网络可达性

很多“下载慢”不是 BT 算法问题，而是入站连接不可达。

建议加入：

- 展示当前 BT 监听端口。
- 检测 TCP/UDP 监听是否成功。
- 支持 UPnP/NAT-PMP/PCP 端口映射状态展示。
- 防火墙拦截时给用户明确提示。
- 区分 TCP peer、uTP peer、WebRTC peer、tracker peer、DHT peer。

提速判断逻辑：

- 如果 peer 很多但 active peer 很少，优先怀疑连接质量、NAT、被 choking 或本地限速。
- 如果 tracker 返回 peer 但连接不上，优先怀疑端口、防火墙、协议阻断。
- 如果 DHT 长时间无节点，优先怀疑 UDP 被阻断或 bootstrap 不足。

### 4.3 上传策略

BT 是互惠协议，上传完全关死通常会影响下载表现。

建议：

- 不要默认上传为 0。
- 给出“保守上传上限”，例如用户上行带宽的 20%-40%。
- 保留最小上传能力，让 peer wire 的互惠机制正常工作。
- 下载完成后可按策略短时间做种，提高未来同资源可用性。

### 4.4 piece 调度策略

纯“顺序下载”不一定最快，因为它会放弃 rarest-first 的效率。但边下边播又必须优先当前位置附近。因此推荐混合策略：

- 对播放窗口：高优先级，保障当前播放和 readahead。
- 对首部：高优先级，保障快速开播。
- 对尾部：中高优先级，兼容 MP4/MKV/AVI 等容器索引或播放器探测。
- 对非播放区域：保持普通优先级，继续 rarest-first，避免长期下载效率下降。
- 对 seek 目标：瞬时最高优先级，然后随播放位置滑动。

推荐窗口：

- 首次开播预热：首部 16-64 MiB，尾部 4-16 MiB。
- 播放 readahead：默认 32-128 MiB，根据下载速度和码率动态调整。
- seek 后热区：seek 点前 2-8 MiB，seek 点后 32-128 MiB。
- 弱网模式：优先保证当前播放窗口，不追求全文件并行。

当前代码的 `streamReaderReadahead = 8 << 20` 偏保守。8 MiB 对低码率视频足够，但对高码率 4K、频繁 seek 或 peer 抖动不稳。建议改为动态值。

### 4.5 动态 readahead

建议根据下载速度和媒体码率估算：

```text
目标缓存秒数 = 60-180 秒
目标 readahead = max(32 MiB, min(256 MiB, 平均播放码率 * 目标缓存秒数 / 8))
```

如果无法解析码率，使用文件大小和时长估算；如果也无法获取时长，按文件大小分档：

- 小于 1 GiB：32 MiB。
- 1-8 GiB：64-128 MiB。
- 大于 8 GiB：128-256 MiB。

## 5. 边下边播方案

当前项目的播放链路已经接近正确形态：

```text
播放器/VLC/mpv/浏览器
  -> HTTP Range 请求
  -> Go http.ServeContent
  -> anacrolix File.NewReader
  -> torrent piece 下载
```

这个方案比“先下载完整文件再播放”更适合当前产品，也比“前端自己解码”更稳。

### 5.1 首帧开播流程

推荐流程：

1. 用户提交 magnet。
2. 后端 AddMagnet。
3. 后端等待 metadata。
4. 获取文件列表，选择最大视频文件或用户指定文件。
5. 优先下载首部 piece。
6. 同时下载尾部 piece。
7. 达到开播阈值后暴露 stream URL。
8. 前端启动外部播放器或内嵌播放器访问本地 HTTP URL。

开播阈值不能只看 metadata ready，应综合：

- selected file 已确定。
- 首部至少完成若干连续 piece。
- 当前 active peer > 0。
- 最近 N 秒有 useful bytes 增长。
- 当前下载速度大于估算播放码率的 1.2-2.0 倍，或缓存量足够。

### 5.2 Range 和 seek 处理

播放器会发很多 Range 请求，尤其是：

- 探测文件头。
- 探测文件尾。
- 获取容器索引。
- 用户拖动进度条。
- 播放器内部预读。

建议在 `StreamController` 或 reader 包装层记录：

- `Range` header。
- 请求 offset。
- 响应状态码。
- 响应耗时。
- 该 Range 触发的 piece 区间。

然后把 Range 请求反馈给 `StreamPriorityController`：

```text
Range: bytes=start-end
  -> 计算 pieceStart/pieceEnd
  -> 提升 [pieceStart, pieceEnd + readaheadWindow] 优先级
  -> 对旧窗口降级，但不要取消全局下载
```

### 5.3 容器格式注意事项

不同视频容器对边播友好程度不同：

- MP4 faststart：最适合边下边播，moov 在文件头。
- 普通 MP4：可能需要尾部 moov，必须预热尾部。
- MKV：通常可边播，但 seek 和索引行为依播放器而异。
- AVI：部分场景需要尾部索引，首尾 piece 优先非常重要。
- TS：天然流式友好。

因此报告建议：不要只做“顺序下载”，而要做“首部 + 尾部 + 当前窗口”的混合策略。

## 6. BT 链接死活检测方案

### 6.1 不能做简单 true/false

BT/Magnet 的存活性是概率问题：

- DHT 查询可能暂时找不到 peer，但过几分钟能找到。
- tracker 可能挂了，但 DHT 有 peer。
- tracker scrape 返回 seed/peer，但实际连接失败。
- 有 peer 不代表有完整文件。
- 有 metadata 不代表可下载完整文件。
- private torrent 可能禁止 DHT/PEX。
- 网络环境可能阻断 UDP，导致误判死链。

所以建议输出“状态 + 置信度 + 证据”，例如：

```json
{
  "state": "Weak",
  "confidence": 0.72,
  "evidence": {
    "metadataResolved": true,
    "trackerPeers": 0,
    "dhtPeers": 3,
    "activePeers": 1,
    "bytesReadUsefulDelta": 1048576,
    "pieceCompletedRecently": true
  }
}
```

### 6.2 分级状态模型

| 状态 | 含义 | 触发条件 |
| --- | --- | --- |
| `Unknown` | 还没有足够信息 | 刚创建任务 |
| `ResolvingMetadata` | 正在解析 magnet metadata | 已 AddMagnet，`GotInfo()` 未完成 |
| `Alive` | 高概率可下载 | metadata ready，active peer > 0，且近期有有效数据或 piece 完成 |
| `Weak` | 可用但风险高 | 有 metadata 或少量 peer，但速度低/连接不稳定 |
| `NoPeers` | 暂未发现 peer | tracker/DHT 均未发现 peer，仍在检测窗口内 |
| `MetadataTimeout` | metadata 超时 | 例如 2-5 分钟内未拿到 metadata |
| `TrackerFailed` | tracker 层失败明显 | 大部分 tracker announce/scrape 失败 |
| `DeadLikely` | 高概率死链 | 多轮 tracker + DHT + 连接尝试均失败 |

### 6.3 检测信号优先级

强证据：

- 成功下载并校验 piece。
- 最近持续收到 useful data。
- metadata 已解析，且有多个 active peer。
- tracker scrape 显示 seed/complete 数量明显大于 0，并且能连接到 peer。

中等证据：

- DHT get_peers 返回 peer 地址。
- tracker announce 返回 peer。
- metadata 解析成功，但还未下载有效 piece。
- peer handshake 成功。

弱证据：

- magnet 格式合法。
- tracker URL 可访问。
- DHT 有节点但未返回该 infohash peer。

反证：

- magnet 格式非法或缺失 infohash。
- 多轮 metadata 超时。
- 所有 tracker 均失败，DHT 也无 peer。
- peer 全部连接失败或全部无所需 piece。
- 长时间 useful bytes 为 0。

### 6.4 检测流程建议

```text
输入 magnet
  -> 解析 infohash 和 tracker 列表
  -> 添加到 torrent client，但可先不全量下载
  -> 并行执行：
       1. 等待 metadata
       2. tracker announce/scrape
       3. DHT get_peers
       4. peer handshake 统计
       5. 小窗口 piece 可下载性探测
  -> 汇总证据
  -> 输出状态、置信度、原因、下一步建议
```

建议时间窗口：

- 快速检测：10-20 秒，给 UI 初步反馈。
- 标准检测：60-120 秒，适合普通用户判断。
- 深度检测：3-5 分钟，适合低热度资源。

### 6.5 置信度建议

```text
Alive >= 0.90:
  metadata ready + active peer >= 1 + useful bytes 增长或 piece 完成

Weak 0.55-0.89:
  metadata ready 或发现 peer，但有效下载不足

NoPeers 0.40-0.70:
  多来源暂未发现 peer，但检测时间还短

DeadLikely >= 0.85:
  标准检测窗口内 metadata 未解析，tracker/DHT/连接均失败
```

注意：`DeadLikely` 也不是 100% 死。更严谨的 UI 文案应是“高概率不可用”，不要写“绝对死链”。

## 7. 推荐重构模块

### 7.1 `HealthProbe`

职责：

- 解析 magnet。
- 记录 tracker 列表和 infohash。
- 跟踪 metadata 状态。
- 跟踪 tracker/DHT/PEX peer 来源。
- 跟踪 active peer、handshake、piece 完成、useful bytes。
- 输出健康状态和置信度。

建议接口：

```go
type TorrentHealthState string

type TorrentHealthReport struct {
    State      TorrentHealthState
    Confidence float64
    Reasons    []string
    CheckedAt  time.Time
}
```

### 7.2 `SpeedScheduler`

职责：

- 管理全局下载策略。
- 控制上传/下载限制。
- 追加公共 tracker。
- 根据速度、peer、piece 稀缺度调整策略。
- 避免播放窗口调度完全破坏 rarest-first。

### 7.3 `StreamPriorityController`

职责：

- 根据 HTTP Range 请求提升 piece 优先级。
- 动态调整 readahead。
- seek 后切换热点窗口。
- 首尾 piece 预热。
- 记录卡顿和等待时间。

### 7.4 `PlaybackReadiness`

职责：

- 判断是否可以展示“播放”按钮。
- 判断是否建议等待缓存。
- 判断当前速度是否足以流畅播放。

建议状态：

- `PreparingMetadata`
- `Buffering`
- `ReadyToPlay`
- `Playing`
- `LikelyToStall`
- `Stalled`
- `Completed`

## 8. 推荐实施路线

### 第 1 阶段：观测与诊断

目标：先知道为什么慢。

任务：

- 在 session 中记录 tracker count、active peers、total peers、download speed、useful bytes delta。
- 将 `DeadState` 从简单枚举扩展为健康报告。
- 记录每次 HTTP Range 请求和响应耗时。
- UI 展示“metadata 中 / peer 数 / 下载速度 / 健康状态 / 最近错误”。

验收标准：

- 一个 magnet 慢时，能区分是 metadata 卡住、无 peer、peer 有但不传、还是播放器等待 Range。

### 第 2 阶段：提速基础配置

目标：提升 peer 发现能力。

任务：

- 支持用户配置公共 tracker 列表。
- 对公开 torrent 自动追加 tracker。
- 暴露 BT 监听端口和网络状态。
- 检测 UDP/DHT 是否工作。
- 增加限速配置，避免上传完全关闭。

验收标准：

- 对 tracker 较少的公开 magnet，peer 发现数量明显增加。

### 第 3 阶段：边播调度

目标：减少首播等待和 seek 卡顿。

任务：

- 将固定 8 MiB readahead 改为动态 readahead。
- 首尾 piece 预热窗口扩大并参数化。
- Range 请求反馈到 piece 优先级调度。
- seek 后提升新窗口，旧窗口降级。

验收标准：

- VLC/mpv 打开本地 stream URL 后可更快开始播放。
- 拖动进度条后能优先下载目标区间。

### 第 4 阶段：深度健康检测

目标：准确识别高概率死链。

任务：

- 实现快速/标准/深度检测窗口。
- 将 metadata timeout、tracker failed、DHT no peers、no useful data 分开记录。
- 输出置信度和证据列表。
- UI 不再只显示“死/活”，而显示“高概率可用/弱可用/暂未发现资源/高概率不可用”。

验收标准：

- 对无 peer magnet，能在 1-2 分钟内给出合理解释。
- 对低热度但可下载资源，不会过早误判死链。

## 9. 风险与边界

### 9.1 技术风险

- BT 网络不可控，热门资源和冷门资源体验差异很大。
- NAT/防火墙会严重影响 peer 连接。
- 某些 tracker 不支持 scrape，只能 announce。
- DHT scrape 是扩展协议，不是所有节点都支持。
- WebTorrent browser peers 与传统 BT peers 不是完全互通，除非使用 hybrid 或支持 WebRTC 的客户端。
- 边下边播对视频容器很敏感，普通 MP4/MKV/AVI 的表现不同。

### 9.2 产品风险

- 如果 UI 写“死链”，用户会认为 100% 不可用；建议写“高概率不可用”。
- 如果默认追加公共 tracker，要允许用户关闭，避免 private torrent 合规问题。
- 如果上传默认过高，会影响用户网络；如果过低，又影响下载速度。

### 9.3 合规边界

该能力应定位为通用 BT 技术与合法内容播放工具。文档和产品不要鼓励下载未授权内容。默认示例应使用公开授权资源，例如 Linux ISO、Internet Archive、Creative Commons 视频。

## 10. 最终推荐架构

```text
React/Wails UI
  -> Session API
      -> TorrentRuntimeManager
          -> anacrolix/torrent Client
          -> HealthProbe
          -> SpeedScheduler
          -> StreamPriorityController
          -> PlaybackReadiness
      -> HTTP Stream Server
          -> ServeContent
          -> Range Observer
          -> File.NewReader
```

核心原则：

- 不重写 BT 协议。
- 不把“顺序下载”当成万能提速。
- 不把“无 peer”立即等同于死链。
- 不让播放器等待逻辑和 torrent 调度脱节。
- 所有健康结论都要附带证据和置信度。

## 11. 资料来源

- anacrolix/torrent GitHub：https://github.com/anacrolix/torrent
- anacrolix/torrent Go package docs：https://pkg.go.dev/github.com/anacrolix/torrent
- libtorrent core/reference docs：https://www.libtorrent.org/reference-Core.html
- libtorrent torrent handle docs：https://libtorrent.org/reference-Torrent_Handle.html
- libtorrent torrent status docs：https://www.libtorrent.org/reference-Torrent_Status.html
- WebTorrent docs：https://webtorrent.io/docs
- WebTorrent FAQ：https://webtorrent.io/faq
- qBittorrent official site：https://www.qbittorrent.org/
- qBittorrent WebUI API wiki：https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)
- aria2 manual：https://aria2.github.io/manual/en/html/README.html
- Transmission official site：https://transmissionbt.com/
- BiglyBT features：https://www.biglybt.com/features.php
- BEP 5 DHT Protocol：https://www.bittorrent.org/beps/bep_0005.html
- BEP 33 DHT Scrapes：https://www.bittorrent.org/beps/bep_0033.html
- BEP 48 Tracker Scrape：https://www.bittorrent.org/beps/bep_0048.html

