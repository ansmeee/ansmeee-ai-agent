# 通用 Agent 智能体 — 研发实施计划

> 关联文档：[产品需求](proposal.md) | [技术方案](design.md)

---

## 1. 总览

- **工期**：4 周（20 个工作日）
- **里程碑**：4 个 Phase，每周交付一个可验证的增量
- **开发人数**：1 后端 + 1 前端（Phase 4 加入）

### 里程碑一览

| Phase | 周期 | 交付物 | 验收标志 |
|-------|------|--------|---------|
| P1 — 底层基础 | 第 1 周 (D1–D5) | 数据模型 + Memory 适配 + 配置 | `go build ./...` 通过，迁移脚本可执行，单元测试通过 |
| P2 — 核心循环 | 第 2 周 (D6–D10) | ReAct 引擎 + 工具 Schema + 并行执行 | mock LLM 下完整 ReAct 循环跑通，16 项单元测试全绿 |
| P3 — API 串联 | 第 3 周 (D11–D15) | Handler + Store + 启动流程 + 集成测试 | 真实 LLM 下端到端 SSE 流式工具调用成功 |
| P4 — 前端 + 收尾 | 第 4 周 (D16–D20) | 前端渲染 + Agent 编辑 + 文档 + 上线 | 浏览器中完整使用 Agent 工具调用功能 |

---

## 2. Phase 1 — 底层基础（D1–D5）

> 目标：完成所有数据层和配置层变更，为 Phase 2 的核心循环提供基础设施。

### D1：数据模型扩展

**任务 1.1 — models/agent.go 新增字段** ✅ 已完成
- 文件：`internal/models/agent.go`
- 内容：
  - 新增 `AgentModelConfig` 结构体（Model / Temperature / MaxTokens / TopP）
  - 新增 `JSONStringSlice` 类型，实现 `driver.Valuer` 和 `sql.Scanner`
  - Agent 结构体新增字段：`Tools JSONStringSlice`、`ModelConfig *AgentModelConfig`、`MaxIterations int8`、`Status int8`
  - 新增 `AgentStatusEnabled int8 = 1` 和 `AgentStatusDisabled int8 = 2` 常量
- 参考：design.md §3.1
- 验收：`go build ./...` 通过

**任务 1.2 — models/chat_message.go 新增 role 常量** ✅ 已完成
- 文件：`internal/models/chat_message.go`
- 内容：新增 `RoleTool int8 = 3` 和 `RoleAssistantToolCall int8 = 4`
- 参考：design.md §3.3
- 验收：编译通过

### D2：数据库迁移

**任务 1.3 — 编写迁移脚本**
- 文件：`migrations/001_agent_tools_support.sql`（新建）
- 内容：
  - 正向迁移：`ALTER TABLE ai_agent ADD COLUMN tools / model_config / max_iterations / status`
  - 回滚脚本：`DROP COLUMN` 四个字段
  - ~~存量数据更新~~：UPDATE 语句已注释为可选操作，避免给不需要工具能力的存量 Agent 误加默认工具
- 参考：design.md §8
- 验收：在开发数据库执行迁移 + 回滚 + 再迁移均无报错

### D3：Memory 接口与 MySQL 适配

**任务 1.4 — memory/interface.go 确认 Message 结构**
- 文件：`internal/memory/interface.go`
- 内容：Message 结构体保持 `Role` + `Content` 两个字段不变。role=tool/assistant_tool_call 的工具元信息统一存储在 content JSON 中（唯一权威数据源），无需新增字段
- 新增辅助函数（可放在 `internal/agent/engine.go` 或独立文件中）：
  - `buildToolCallJSON(toolCalls)` — 构建 role=4 的 content JSON
  - `buildToolResultJSON(toolCallID, name, result)` — 构建 role=3 的 content JSON
- 参考：design.md §4.4.1
- 验收：编译通过

**任务 1.5 — memory/mysql.go role 映射**
- 文件：`internal/memory/mysql.go`
- 内容：
  - 新增 `roleStringToInt()` 和 `roleIntToString()` 函数
  - `AddMessage()` 使用 roleStringToInt 替代现有硬编码 if/else
  - `History()` 使用 roleIntToString，role=3/4 时 content 原样返回（JSON 解析在 engine 层 `buildMessages()` 中处理）
- 参考：design.md §4.4.2
- 验收：编写单元测试 `TestMySQLStore_ToolMessage`，验证 role=3/4 的存取正确性

**任务 1.6 — memory/memory.go InMemoryStore 适配**
- 文件：`internal/memory/memory.go`
- 内容：InMemoryStore 无需变更（Message 结构未变，新 role 值直接存取即可），仅需确认 AddMessage/History 对 role="tool"/"assistant_tool_call" 的透传正确
- 验收：`TestInMemoryStore_ToolMessage`

### D4：配置系统扩展

**任务 1.7 — config/config.go 新增 AgentConfig**
- 文件：`internal/config/config.go`
- 内容：
  - 新增 `AgentConfig` 结构体（MaxIterations / ToolTimeout / MaxOutputLength / ParallelToolCalls / MaxContextMessages）
  - Config 结构体新增 `Agent AgentConfig` 字段
  - `applyDefaults()` 中为 Agent 配置设置默认值（max_iterations=5, tool_timeout=30s, max_output_length=4096, parallel_tool_calls=true, max_context_messages=50）
- 参考：design.md §4.7

**任务 1.8 — configs/config.yaml 新增配置段**
- 文件：`configs/config.yaml`、`configs/config.example.yaml`
- 内容：新增 `agent:` 配置段，包含 max_iterations / tool_timeout / max_output_length / parallel_tool_calls / max_context_messages
- 验收：`make build` 通过，配置正确加载

### D5：Phase 1 自测 + 修复

- 运行 `go build ./...`，确认无编译错误
- 运行 `go test ./...`，确认新增单元测试全部通过
- 运行 `go vet ./...` + `go fmt ./...`
- 在开发数据库执行迁移脚本，确认字段生效
- 修复过程中发现的问题

**Phase 1 交付检查清单：**
- [ ] `go build ./...` 无错误
- [ ] `go test ./internal/memory/...` 全部通过（含 ToolMessage 测试）
- [ ] 数据库迁移执行成功，`DESCRIBE ai_agent` 可见新字段
- [ ] `configs/config.yaml` 新增 agent 段，Load 无报错

---

## 3. Phase 2 — 核心循环（D6–D10）

> 目标：实现 ReAct 推理循环和工具执行，这是整个功能的核心引擎。

### D6：工具系统增强

**任务 2.1 — 各工具实现 Parameters()**
- 文件：`internal/tool/calculator.go`、`internal/tool/datetime.go`、`internal/tool/web_search.go`、`internal/tool/weather.go`
- 内容：每个工具新增 `Parameters() map[string]any` 方法，返回 JSON Schema
- 参考：design.md §4.3.2
- 验收：编译通过

**任务 2.2 — llm/provider.go 新增 ToolWithSchema 接口**
- 文件：`internal/llm/provider.go`
- 内容：
  - 定义 `ToolWithSchema` 接口（扩展 `tools.Tool`，新增 `Parameters()`）
  - `toLLMTools()` 检测 ToolWithSchema，将 Parameters 填入 `FunctionDefinition.Parameters`
- 参考：design.md §4.2.1
- 验收：单元测试 `TestToLLMTools_WithParameters` 验证 Schema 正确传递

**任务 2.3 — tool/registry.go 新增方法**
- 文件：`internal/tool/registry.go`
- 内容：
  - 新增 `GetByNames(names []string) []tools.Tool`
  - 新增 `GetSchema(name string) (ToolSchema, bool)`
  - 新增 `ToolSchema` 结构体
- 参考：design.md §4.3.1
- 验收：`TestGetByNames` + `TestGetSchema`

### D7：LLM Provider Chat 增强

**任务 2.4 — llm/provider.go Chat 方法 ChatOption**
- 文件：`internal/llm/provider.go`
- 内容：
  - 新增 `ChatOption` 类型和 `WithTemperature()`、`WithChatMaxTokens()`、`WithTopP()` 函数
  - `Chat()` 签名新增 `opts ...ChatOption`，内部合并参数覆盖（含 TopP）
- 参考：design.md §4.2.2
- 验收：`TestChat_WithOptions` 验证参数覆盖生效

### D8：Engine ReAct 循环

**任务 2.5 — agent/engine.go 核心改造**
- 文件：`internal/agent/engine.go`、`internal/agent/callback.go`（确认 OnToolStart/OnToolEnd 签名匹配 design.md §4.1.6）
- 内容（按 design.md §4.1 逐步实现）：
  - StreamEvent 结构体扩展（新增 ToolCallID / ToolName / Arguments / Result / Success / Iteration）
  - 新增 `AgentConfig` 结构体（含 `ParallelToolCalls *bool` 字段）
  - `ProcessStream()` 签名变更（接收 `*AgentConfig` 替代 `promptOverride string`）
  - 新增 `resolveTools()` — 按 Agent 配置过滤工具
  - 新增 `resolveLLMProvider()` — 三级模型配置合并
  - 新增 `buildChatOptions()` — 根据 AgentConfig 构建 Chat 调用的 temperature/maxTokens/topP 选项
  - 新增 `executeToolWithTimeout()` — 带超时 + panic recover + 输出截断 + trace_id 传播（调用 callback.OnToolStart/OnToolEnd）
  - 新增 `emitContentAsChunks()` — 将 Chat 返回的完整文本按 4 字符拆分为 chunk 事件发送（替代原 streamFinalAnswer 的双重调用）
  - 新增 `executeAndEmitTool()` — 单个工具的执行 + SSE 事件发送 + 持久化
  - 改造 `buildMessages(prompt, history, maxContextMessages)` — 支持 role=tool / assistant_tool_call，从 content JSON 解析工具元信息
  - 新增 `trimHistory()` — 上下文窗口管理，保持 tool_call/tool 配对
  - 实现 ReAct 主循环（for iter < maxIter），循环中根据 `parallelToolCalls` 选择串行或并行路径
  - 超过 maxIter 时：调用 Chat(messages, nil) 不带工具强制生成，然后 emitContentAsChunks 发送
- 参考：design.md §4.1.1–4.1.7
- **这是最大的单个任务，预留 2 天**
- 验收：通过 mock LLM 的单元测试（见 D10）

### D9：并行工具调用

**任务 2.6 — agent/engine.go 并行执行**
- 文件：`internal/agent/engine.go`
- 内容：
  - 新增 `executeToolsConcurrently()` — errgroup 并发执行 + SetLimit(5) + sync.Mutex 保序
  - ReAct 循环中根据 `parallelToolCalls` 配置 + ToolCalls > 1 时调用并行逻辑，否则串行调用 `executeAndEmitTool()`
- 参考：design.md §6
- 依赖：go.mod 中添加 `golang.org/x/sync/errgroup`（如未存在）
- 验收：`TestProcessStream_ParallelTools` + `TestProcessStream_SerialTools`

**任务 2.7 — Engine 新增 Option**
- 文件：`internal/agent/engine.go`
- 内容：
  - 新增 `WithToolTimeout(d time.Duration)` Option
  - 新增 `WithMaxOutputLength(n int)` Option
  - 新增 `WithParallelToolCalls(b bool)` Option
  - 新增 `WithMaxContextMessages(n int)` Option
  - Engine 结构体新增 `toolTimeout`、`maxOutputLen`、`parallelToolCalls`、`maxContextMessages` 字段
- 验收：`TestEngineOptions`

### D10：单元测试全覆盖

**任务 2.8 — 编写 Engine 单元测试**
- 文件：`internal/agent/engine_test.go`（新建）
- Mock 策略：
  - Mock LLM Provider：按序返回预设 ChatResult（含/不含 ToolCalls）
  - Mock Tool：可控返回值、延迟、panic
  - Memory：使用 InMemoryStore
- 测试用例（16 项，参考 design.md §10.1）：

| # | 测试函数 | 场景 |
|---|---------|------|
| 1 | `TestProcessStream_NoTools` | 未配置工具 → 纯对话 |
| 2 | `TestProcessStream_SingleToolCall` | 单次工具调用 → 最终回答 |
| 3 | `TestProcessStream_MultiRound` | 多轮工具调用 → 最终回答 |
| 4 | `TestProcessStream_ParallelTools` | parallel_tool_calls=true 时并行工具调用 |
| 5 | `TestProcessStream_SerialTools` | parallel_tool_calls=false 时串行工具调用 |
| 6 | `TestProcessStream_ToolTimeout` | 工具超时 → 错误 result |
| 7 | `TestProcessStream_ToolPanic` | 工具 panic → recover |
| 8 | `TestProcessStream_MaxIter` | 达到 maxIter → 强制结束（Chat 不带工具 + emitContentAsChunks） |
| 9 | `TestProcessStream_ToolNotFound` | 工具名无效 → 错误 result |
| 10 | `TestProcessStream_ContextCancel` | 中途取消 |
| 11 | `TestResolveTools_FilterByNames` | 工具过滤 |
| 12 | `TestResolveLLMProvider_Priority` | 模型配置合并优先级 |
| 13 | `TestBuildMessages_WithToolHistory` | 含工具历史的消息构建（从 content JSON 解析） |
| 14 | `TestTrimHistory_KeepPairs` | 截断保持配对完整 |
| 15 | `TestEmitContentAsChunks` | 文本按 4 字符拆分为 chunk |
| 16 | `TestBuildChatOptions` | 从 AgentConfig 构建 ChatOption |

- 验收：`go test -v ./internal/agent/...` 全部通过

**Phase 2 交付检查清单：**
- [ ] `go test ./...` 全部通过（含 16 项 Engine 测试）
- [ ] Mock LLM 下 ReAct 循环可跑通：thinking → tool_start → tool_end → chunk → done
- [ ] 并行工具调用正确保序（parallel_tool_calls=true）
- [ ] 串行工具调用正确执行（parallel_tool_calls=false）
- [ ] 工具超时 / panic / 未找到均正确处理
- [ ] emitContentAsChunks 正确拆分文本并发送 chunk 事件

---

## 4. Phase 3 — API 串联（D11–D15）

> 目标：将底层能力接入 HTTP 层，实现端到端可用。

### D11：AgentStore 改造

**任务 3.1 — agent/store.go 支持新字段**
- 文件：`internal/agent/store.go`
- 内容：
  - `Create()` 签名扩展，接收 tools / modelConfig / maxIterations
  - `Update()` 改为接收 `map[string]interface{}`，白名单过滤字段
  - `EnsureDefault()` 创建默认 Agent 时设置默认工具列表
- 参考：design.md §4.6
- 验收：`TestAgentStore_CreateWithTools`、`TestAgentStore_UpdateFields`

### D12：Handler 层适配

**任务 3.2 — handler/agent.go CRUD 适配**
- 文件：`internal/handler/agent.go`
- 内容：
  - Create 接口解析新字段（tools / model_config / max_iterations）
  - Update 接口解析新字段
  - Response 返回完整 Agent 对象（含新字段）
- 验收：curl 测试 Create/Update/Get 接口，确认新字段正确读写

**任务 3.3 — handler/stream.go 全面改造**
- 文件：`internal/handler/stream.go`
- 内容：
  - 新增 SSE 类型结构体：`sseChunkData`、`sseThinkingData`、`sseToolStartData`、`sseToolEndData`、`sseSessionData`、`sseErrorData`
  - 新增 `writeSSEJSON()` — 替代现有 `writeSSE()`，使用 `json.Marshal` 确保所有 SSE data 为合法 JSON（防注入）
  - 新增 `ensureJSON()` — 确保字符串是合法 JSON，非法则包装为 JSON 字符串
  - `Handle()` 变更：
    - 调用 `resolveAgentConfig()` 获取完整 Agent 配置（含状态检查）
    - 在 resolveAgentConfig 之后立即发送 `session` SSE 事件（PRD 4.2.1 要求）
    - 传入 `agent.AgentConfig` 调用 `engine.ProcessStream()`
    - 所有事件使用 `writeSSEJSON()` 发送
    - `chunk` 事件格式从裸字符串改为 `{"content":"..."}`
    - `error` 事件格式从裸字符串改为 `{"message":"..."}`
  - `resolveAgentConfig()` 改为返回 `(*agent.AgentConfig, error)`，检查 Agent 状态（`AgentStatusDisabled` 返回错误）
- 参考：design.md §4.5
- 验收：curl 发送 SSE 请求，收到 session + thinking + tool_start + tool_end + chunk(JSON) + done 完整事件流

**任务 3.4 — handler/tool.go Schema 接口**
- 文件：`internal/handler/tool.go`
- 内容：新增 `Schema()` 方法，处理 `GET /api/v1/tools/:name/schema`
- 参考：design.md §4.8

**任务 3.5 — router/router.go 注册新路由**
- 文件：`internal/router/router.go`
- 内容：注册 `GET /api/v1/tools/:name/schema` 路由
- 验收：curl 请求返回工具 Schema JSON

### D13：启动流程适配

**任务 3.6 — cmd/server/main.go**
- 文件：`cmd/server/main.go`
- 内容：
  - Engine 构造时传入所有新 Option：`WithToolTimeout`、`WithMaxOutputLength`、`WithParallelToolCalls`、`WithMaxContextMessages`
  - 从 cfg.Agent 读取配置
- 参考：design.md §9
- 验收：`make run` 启动无报错

### D14：集成测试

**任务 3.7 — 端到端集成测试**
- 文件：`internal/handler/stream_test.go`（新建或扩展）
- 测试方式：启动真实 HTTP Server（gin TestMode），使用真实 InMemoryStore + 真实 Tool Registry + Mock LLM
- 测试用例（5 项，参考 design.md §10.3）：

| # | 测试场景 | 验证点 |
|---|---------|--------|
| 1 | 数学计算端到端 | SSE 流含 session + thinking + tool_start + tool_end + chunk + done，所有 data 为合法 JSON |
| 2 | 多轮工具调用 | 多个 tool_start/tool_end 对 + 最终答案 |
| 3 | Agent 配置生效 | 限定工具的 Agent → 只调用配置的工具 |
| 4 | 历史回放 | GET /chat/:sessionId 返回 role=3/4 消息 |
| 5 | 禁用 Agent | status=2 的 Agent → 返回 error 事件 |

- 验收：`go test -v -tags integration ./internal/handler/...` 全部通过

### D15：Phase 3 联调 + Bug 修复

- 使用真实 LLM（Deepseek）进行端到端测试
- 测试场景：
  1. "今天是星期几" → 触发 datetime 工具
  2. "计算 2^10 + 365 除以 7 的余数" → 触发 calculator 多轮调用
  3. 创建只有 calculator 的 Agent → 确认不会调用 datetime
  4. 查看历史 → 确认工具调用消息完整（role=3/4 content 为合法 JSON）
  5. 禁用 Agent 后发起对话 → 确认返回 error 事件
  6. 验证 SSE 流首事件为 session，所有 data 字段为合法 JSON
- 修复联调中发现的问题
- LangChainGo v0.1.13 ToolCall 兼容性验证（如有问题则升级或自行适配）

**Phase 3 交付检查清单：**
- [ ] `go test ./...` 全部通过（含集成测试）
- [ ] curl 手动测试 Agent CRUD 新字段正确
- [ ] curl SSE 流包含完整事件类型（session → thinking → tool_start → tool_end → chunk → done）
- [ ] SSE 所有 data 字段为合法 JSON（chunk 为 `{"content":"..."}`，error 为 `{"message":"..."}`）
- [ ] 禁用 Agent 返回 error 事件
- [ ] 真实 LLM 下工具调用成功
- [ ] GET /api/v1/tools/:name/schema 返回正确 Schema
- [ ] 历史接口包含 tool 类型消息

---

## 5. Phase 4 — 前端 + 收尾（D16–D20）

> 目标：前端适配新 SSE 事件，完成 Agent 编辑增强，上线。

### D16–D17：对话界面工具调用渲染

**任务 4.1 — SSE 事件解析扩展**
- 文件：`web/index.html`（JS 部分）
- 内容：
  - EventSource 新增 `session` / `thinking` / `tool_start` / `tool_end` 事件监听
  - 所有事件 data 按 JSON 解析（chunk 为 `{"content":"..."}`，error 为 `{"message":"..."}`）
  - 维护当前 Agent 的工具调用状态（pending / executing / done）

**任务 4.2 — 工具调用 UI 组件**
- 文件：`web/index.html`（HTML + CSS + JS）
- 内容：
  - 工具调用折叠块（默认收起，点击展开）
  - 显示工具名、输入参数、返回结果
  - 执行中 loading 动画
  - 成功（绿色）/ 失败（红色）样式区分
  - thinking 状态指示器（"思考中..."动画）
- 参考：proposal.md §4.5.1
- 验收：浏览器中可见工具调用过程

**任务 4.3 — 历史回放工具消息渲染**
- 文件：`web/index.html`
- 内容：
  - 历史消息列表支持 role=3（tool result）和 role=4（tool call）的渲染
  - 复用工具调用折叠块组件
- 验收：刷新页面后历史对话中工具调用过程可正确显示

### D18：Agent 编辑页增强 + 模板系统

**任务 4.4 — 工具多选组件**
- 文件：`web/agent.html`
- 内容：
  - 从 `GET /api/v1/tools` 获取工具列表
  - 多选 checkbox 列表（工具名 + 描述）
  - 保存时提交 tools 数组
- 验收：创建/编辑 Agent 可选择工具

**任务 4.5 — 模型参数配置表单**
- 文件：`web/agent.html`
- 内容：
  - Temperature 滑块（0–2，步长 0.1）
  - Max Tokens 数字输入框
  - 最大迭代次数下拉（1–10）
  - 保存时提交 model_config JSON
- 验收：配置保存后 GET 返回正确值

**任务 4.6 — Agent 模板系统**（对应 PRD US-8）
- 文件：
  - `internal/models/agent_template.go`（新建）— AgentTemplate 数据模型
  - `internal/handler/agent.go`（扩展）— 新增 `GET /api/v1/agent-templates` 接口
  - `web/agent.html`（扩展）— 创建 Agent 时可选择模板预填配置
- 内容：
  - AgentTemplate 模型含 Name / Category / Description / Prompt / Tools / ModelConfig / MaxIterations / SortOrder
  - 通过 seed 数据插入预置模板（通用助手、数学助手、信息查询）
  - 前端创建 Agent 页面显示模板选择下拉，选择后预填 Prompt / Tools / ModelConfig
- 参考：design.md §3.2
- 验收：选择模板后，表单自动填充推荐配置

### D19：联调 + Bug 修复

- 浏览器端到端完整测试：
  1. 创建 Agent（选择工具 + 设置参数）→ 对话 → 看到工具调用过程 → 看到最终回答
  2. 切换 Agent → 工具集不同 → 行为不同
  3. 历史回放 → 工具调用完整展示
  4. 纯对话 Agent（无工具）→ 退化为正常对话
  5. 工具失败场景 → 优雅显示错误信息
- 修复过程中发现的问题

### D20：文档 + 上线

**任务 4.7 — 文档更新**
- 更新 CLAUDE.md 中的 API 文档表格（新增接口 + 变更接口）
- 更新 CLAUDE.md 中的 Core Components 描述

**任务 4.8 — 上线前检查**
- `go vet ./...` 无警告
- `go fmt ./...` 无变更
- `go test ./...` 全部通过
- 数据库迁移脚本 review
- 配置文件 review（确认 `config.example.yaml` 同步更新）

**任务 4.9 — 部署上线**
- 执行数据库迁移
- 部署新版服务
- 灰度验证核心场景
- 监控日志确认无异常

**Phase 4 交付检查清单：**
- [ ] 浏览器中工具调用过程完整可见
- [ ] Agent 编辑页工具选择和参数配置可用
- [ ] Agent 模板系统可用（创建时可选模板预填配置）
- [ ] 历史回放正确渲染工具消息
- [ ] 纯对话模式正常（向后兼容）
- [ ] CLAUDE.md 已更新
- [ ] `go test ./...` 全绿
- [ ] 线上部署并验证

---

## 6. 依赖与风险

### 6.1 外部依赖

| 依赖项 | 用途 | 风险 | 预案 |
|--------|------|------|------|
| LangChainGo v0.1.13 ToolCall 支持 | LLM 返回 ToolCalls 解析 | 版本可能不完整支持 | D15 联调时验证；不兼容则升级或绕过 |
| `golang.org/x/sync/errgroup` | 并行工具执行 | 低（标准库扩展） | — |
| LLM API（Deepseek） | Function Calling 能力 | API 需支持 function calling | D15 联调验证；不支持则切换模型 |

### 6.2 关键路径

```
任务 1.1 (models) ──→ 任务 1.5 (mysql store) ──→ 任务 2.5 (engine) ──→ 任务 3.3 (stream handler) ──→ 任务 4.1 (前端)
     │                                                   │
     └─→ 任务 1.3 (migration) ───────────────────────→ 任务 3.6 (main.go)
                                                         │
     任务 2.1 (tool params) ──→ 任务 2.2 (schema) ──→ 任务 2.5 (engine)
```

关键路径上的瓶颈是 **任务 2.5（Engine ReAct 循环）**，预留 2 天。

### 6.3 风险清单

| 风险 | 概率 | 影响 | 缓解 | 触发时间 |
|------|------|------|------|---------|
| LangChainGo ToolCall 解析不兼容 | 中 | 高 | D15 联调时验证，不兼容则自行解析 raw response | Phase 3 |
| LLM 不稳定调用工具（幻觉参数） | 中 | 中 | 工具参数校验 + 清晰 Schema 描述 | Phase 3 |
| 工具 content JSON 格式需要后续变更 | 低 | 中 | JSON 中加 version 字段预留 | Phase 1 |
| 前端工期不足 | 低 | 中 | Phase 4 可裁剪模板系统和模型参数配置（仅保留工具选择） | Phase 4 |
| emitContentAsChunks 首字节延迟 | 低 | 低 | 需等 LLM 完整返回才能发送首 chunk；对大多数场景可忽略。如业务要求极低首字节时间，可后续引入真正的流式 function calling（需 LangChainGo 支持） | Phase 2 |

---

## 7. 每日交付看板

| 天 | 任务 | 产出物 | 验收 | Check |
|----|------|--------|------|-------|
| D1 | 1.1 + 1.2 | models 变更 | `go build` 通过 | [x] |
| D2 | 1.3 | 迁移脚本 | 数据库执行成功 | [ ] |
| D3 | 1.4 + 1.5 + 1.6 | memory role 映射 + 辅助函数 | 单元测试通过 | [ ] |
| D4 | 1.7 + 1.8 | 配置扩展（含 parallel_tool_calls + max_context_messages） | `make build` 通过 | [ ] |
| D5 | 自测修复 | Phase 1 完整 | 检查清单全勾 | [ ] |
| D6 | 2.1 + 2.2 + 2.3 | 工具 Schema + Registry | 单元测试通过 | [ ] |
| D7 | 2.4 | LLM ChatOption | 单元测试通过 | [ ] |
| D8 | 2.5（前半） | Engine 基本循环 + emitContentAsChunks + executeAndEmitTool | 单工具调用测试通过 | [ ] |
| D9 | 2.5（后半）+ 2.6 + 2.7 | 并行/串行执行 + 全部 Option | 并行+串行测试通过 | [ ] |
| D10 | 2.8 | 16 项单元测试 | 全绿 | [ ] |
| D11 | 3.1 | AgentStore 改造 | 单元测试通过 | [ ] |
| D12 | 3.2 + 3.3 + 3.4 + 3.5 | Handler（含 writeSSEJSON + 状态检查）+ Router | curl 测试通过 | [ ] |
| D13 | 3.6 | main.go 适配 | `make run` 启动成功 | [ ] |
| D14 | 3.7 | 5 项集成测试 | 全部通过 | [ ] |
| D15 | 联调 + 修复 | 真实 LLM 端到端 | 检查清单全勾 | [ ] |
| D16 | 4.1 + 4.2 | 对话界面工具渲染（SSE JSON 解析） | 浏览器可见工具过程 | [ ] |
| D17 | 4.3 | 历史回放渲染 | 刷新后工具消息可见 | [ ] |
| D18 | 4.4 + 4.5 + 4.6 | Agent 编辑页增强 + 模板系统 | 工具选择和参数可配置，模板可用 | [ ] |
| D19 | 联调 + 修复 | 端到端完整 | 浏览器完整测试通过 | [ ] |
| D20 | 4.7 + 4.8 + 4.9 | 文档 + 上线 | 线上验证通过 | [ ] |

---

## 8. 变更文件清单

按修改顺序汇总，方便 Code Review 时对照：

| Phase | 文件 | 变更类型 |
|-------|------|---------|
| P1 | `internal/models/agent.go` | 已完成 ✅ |
| P1 | `internal/models/chat_message.go` | 已完成 ✅ |
| P1 | `migrations/001_agent_tools_support.sql` | 新建 |
| P1 | `internal/memory/mysql.go` | 扩展 |
| P1 | `internal/config/config.go` | 扩展 |
| P1 | `configs/config.yaml` | 扩展 |
| P1 | `configs/config.example.yaml` | 扩展 |
| P2 | `internal/tool/calculator.go` | 扩展 |
| P2 | `internal/tool/datetime.go` | 扩展 |
| P2 | `internal/tool/web_search.go` | 扩展 |
| P2 | `internal/tool/weather.go` | 扩展 |
| P2 | `internal/llm/provider.go` | 扩展 |
| P2 | `internal/tool/registry.go` | 扩展 |
| P2 | `internal/agent/callback.go` | 扩展 |
| P2 | `internal/agent/engine.go` | 重构 |
| P2 | `internal/agent/engine_test.go` | 新建 |
| P3 | `internal/agent/store.go` | 扩展 |
| P3 | `internal/handler/agent.go` | 扩展 |
| P3 | `internal/handler/stream.go` | 重构 |
| P3 | `internal/handler/tool.go` | 扩展 |
| P3 | `internal/router/router.go` | 扩展 |
| P3 | `cmd/server/main.go` | 扩展 |
| P3 | `internal/handler/stream_test.go` | 新建 |
| P4 | `internal/models/agent_template.go` | 新建 |
| P4 | `web/index.html` | 扩展 |
| P4 | `web/agent.html` | 扩展 |
| P4 | `CLAUDE.md` | 更新 |

共计 **27 个文件**（16 个扩展 + 4 个新建 + 2 个重构 + 2 个配置扩展 + 1 个文档更新 + 2 个已完成）。

> **注**：相较旧版本的主要变更：
> - 移除 `internal/memory/interface.go`（Message 结构不变，无需修改）
> - 移除 `internal/memory/memory.go`（InMemoryStore 透传 role 值，无需修改）
> - 新增 `internal/models/agent_template.go`（Phase 4 模板系统）
> - `internal/handler/stream.go` 从"扩展"升级为"重构"（writeSSEJSON + JSON 类型结构体 + session 事件 + 状态检查）
