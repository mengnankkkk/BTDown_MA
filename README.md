# BTDown_MA

`BTDown_MA` 是一个面向 Windows 桌面场景的 BT 边下边播项目。

- `Go` 负责 torrent runtime、调度、流媒体与系统集成
- `Wails + React` 负责桌面壳与前端交互
- 本地 `HTTP Range` stream 负责向外部播放器提供媒体数据

当前项目已经完成流式 MVP，正在从“能播”继续推进到“播得稳、seek 快、可高质量诊断”。

## 快速入口

1. [todolist.md](/E:/github/BTDown_MA/todolist.md:1)
当前权威状态页，只回答“做到哪了、最缺什么、下一步先做什么”。

2. [docs/README.md](/E:/github/BTDown_MA/docs/README.md:1)
文档总索引，统一说明各文档职责、阅读路径和维护规则。

## 文档原则

- `README.md` 只做仓库入口，不重复维护详细进度和深度分析
- `todolist.md` 是唯一的当前状态权威页
- `docs/README.md` 是文档导航与维护规则入口
- 深度分析、路线图、历史资料分别放在各自文档中，避免多处同步同一信息
