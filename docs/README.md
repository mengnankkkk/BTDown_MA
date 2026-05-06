# docs 文档索引

## 1. 先看哪几份

如果你只想快速进入当前项目语境，按这个顺序读：

1. [../todolist.md](/E:/github/BTDown_MA/todolist.md:1)
当前权威状态页。

2. [streaming-execution-roadmap.md](/E:/github/BTDown_MA/docs/streaming-execution-roadmap.md:1)
当前执行路线图。

3. [download-speed-analysis-plan.md](/E:/github/BTDown_MA/docs/download-speed-analysis-plan.md:1)
当前瓶颈分析与优化优先级判断。

4. [open-source-streaming-performance-research.md](/E:/github/BTDown_MA/docs/open-source-streaming-performance-research.md:1)
外部项目调研与可借鉴思路。

## 2. 文档分层

### 仓库入口

- [../README.md](/E:/github/BTDown_MA/README.md:1)
只负责仓库简介和总入口，不维护详细状态。

### 当前权威

- [../todolist.md](/E:/github/BTDown_MA/todolist.md:1)
唯一的当前状态权威页，维护当前进度、关键缺口、下一步顺序。

### 当前设计与分析

- [streaming-execution-roadmap.md](/E:/github/BTDown_MA/docs/streaming-execution-roadmap.md:1)
把现有分析收敛为阶段路线图。

- [download-speed-analysis-plan.md](/E:/github/BTDown_MA/docs/download-speed-analysis-plan.md:1)
解释当前速度、首播、seek 方面为什么还有优化空间。

- [open-source-streaming-performance-research.md](/E:/github/BTDown_MA/docs/open-source-streaming-performance-research.md:1)
整理同类项目的实现和性能启发。

### 历史基线与归档

- [archive/bt-streaming-mvp-todolist.md](/E:/github/BTDown_MA/docs/archive/bt-streaming-mvp-todolist.md:1)
最初的总体方案与历史基线。

- [archive/find.md](/E:/github/BTDown_MA/docs/archive/find.md:1)
早期调研摘要归档。

## 3. 单一事实来源

- 当前状态只在 `../todolist.md` 维护
- 路线与阶段验收只在 `streaming-execution-roadmap.md` 深化
- 瓶颈归因只在 `download-speed-analysis-plan.md` 深化
- 外部调研结论只在 `open-source-streaming-performance-research.md` 深化
- 历史资料进入 `archive/` 后不再承担当前决策职责

## 4. 维护规则

1. 当前状态变化时，只更新 `../todolist.md`
2. 阶段顺序或验收口径变化时，同时更新 `../todolist.md` 和 `streaming-execution-roadmap.md`
3. 如果某份文档内容已被新文档吸收，优先迁入 `archive/`，而不是继续并行维护
4. 新文档创建前先判断是否已有现成职责承载位置，避免重复开新坑
