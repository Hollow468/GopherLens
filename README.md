# GopherLens

**基于 Multi-Agent 的侵入式 Go 业务逻辑分析与自动化 Test-Case 闭环生成器**

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

GopherLens 不只是生成简单的测试代码 — 它通过 AI Agent 深度理解代码意图，自动识别外部依赖（SQL、Redis、RPC），并生成能够模拟真实故障的测试用例，实现代码质量的自动化"闭环"。

---

## 痛点

Golang Web 服务涉及大量的 context 传递、嵌套 struct 转换、数据库事务以及对第三方 API 的依赖。当一个开发者接手两三年前的旧接口时，面对几百行业务逻辑和缺失的文档，补齐高覆盖率的单元测试极其耗时。

GopherLens 解决的就是这个问题。

---

## 架构

GopherLens 使用 **5 个协作 Agent** 组成流水线，每个 Agent 负责一个专业任务：

```
┌─────────────┐    ┌──────────────┐    ┌────────────────┐    ┌───────────┐    ┌─────────────┐
│  Architect   │───▶│  Logic Miner  │───▶│ Test Architect  │───▶│   Coder   │───▶│  Validator  │
│  架构分析    │    │  逻辑挖掘     │    │   测试架构      │    │  代码生成  │    │  闭环验证   │
└─────────────┘    └──────────────┘    └────────────────┘    └───────────┘    └─────────────┘
       │                  │                   │                    │                  │
  静态分析 AST      提取 if-else        设计测试矩阵          生成 _test.go       go test -cover
  识别 Mock 目标    追踪 HTTP 状态码    正常/异常/边界          testify/mock      < 80%? 回环 ◀──┐
       │                  │                   │                    │                  │          │
       └──────────────────┴───────────────────┴────────────────────┴──────────────────┘          │
                                              ▲                                                   │
                                              └───────────────────────────────────────────────────┘
                                                   闭环反馈 (覆盖率未达标时自动重试)
```

### Agent 角色

| Agent | 职责 | 核心能力 |
|---|---|---|
| **Architect** | 静态代码分析 | 解析 AST，识别 `gorm.DB`、`*sql.DB`、`*redis.Client`、`*http.Client` 等需要 Mock 的依赖，绘制调用拓扑 |
| **Logic Miner** | 业务路径提取 | 逐行解析函数，识别 if-else 分支、错误处理点、HTTP 状态码，输出 JSON 格式的逻辑路径 |
| **Test Architect** | 测试矩阵设计 | 根据逻辑路径设计测试矩阵（正常流、异常流、边界值），遵循 Go table-driven tests 模式 |
| **Coder** | 测试代码生成 | 使用 testify/mock 和 sqlmock 生成 `_test.go` 文件 |
| **Validator** | 闭环验证 | 执行 `go test -v -cover`，覆盖率低于 80% 自动反馈到 Logic Miner 重新分析 |

---

## 快速开始

### 安装

```bash
git clone https://github.com/Hollow468/GopherLens.git
cd GopherLens
go build -o bin/gopherlens ./cmd/gopherlens/
```

### 使用

```bash
# 分析单个 Go 文件并生成测试
./bin/gopherlens -file path/to/your/handler.go

# 自定义覆盖率目标（默认 80%）
./bin/gopherlens -file path/to/your/handler.go -coverage 85.0
```

### 示例输出

```
GopherLens: analyzing testdata/sample/sample.go (target coverage: 80.0%)

=== GopherLens Report ===
File: testdata/sample/sample.go
Module: github.com/hollow/gopherlens/testdata
Dependencies found: 22
  - sql.DB (sql)
  - http.ResponseWriter (http)
  - http.Request (http)
  ...

Logic paths discovered: 4
  [path_0] Happy path: HandleGetUser succeeds, returns 200
  [path_1] http.Error returns 400
  [path_2] http.Error returns 500
  [path_3] http.Error returns 404

Test cases designed: 6
Estimated coverage: 100.0%

Test file: testdata/sample/sample_test.go
Generated 2846 bytes of test code

Validation (iterations=5):
  Passed: false
  Coverage: 0.0%  (需要真实数据库连接时配合 sqlmock 使用)

Phase: complete
```

---

## 项目结构

```
.
├── cmd/gopherlens/main.go          # CLI 入口
├── pkg/types/types.go              # 共享数据类型定义
├── internal/
│   ├── agent/agent.go              # Agent 核心接口
│   ├── architect/architect.go      # 架构分析 Agent
│   ├── logicminer/logicminer.go    # 逻辑挖掘 Agent
│   ├── testarchitect/testarchitect.go # 测试架构 Agent
│   ├── coder/coder.go              # 代码生成 Agent
│   ├── validator/validator.go      # 闭环验证 Agent
│   └── orchestrator/orchestrator.go # 流水线编排引擎
└── testdata/sample/                # 示例文件
    └── sample.go
```

---

## 工作流

### 第一阶段：感知（Context Extraction）
Architect Agent 读取目标文件和 `go.mod`，解析 AST，识别需要 Mock 的外部依赖。

### 第二阶段：推理（Path Reasoning）
Logic Miner 逐行解析函数，输出 JSON 格式的逻辑描述：
```json
{
  "paths": [
    {"id": "path_0", "description": "ID 不存在 → 返回 404"},
    {"id": "path_1", "description": "数据库超时 → 返回 500 并重试"}
  ]
}
```

### 第三阶段：生成（Code Synthesis）
Test Architect 给出测试矩阵，Coder Agent 生成符合 Go table-driven tests 标准的 `_test.go` 文件。

### 第四阶段：闭环（Execution Loop）
Validator Agent 运行 `go test -v -cover`，如果覆盖率低于目标阈值（默认 80%），自动寻找未覆盖分支，重新进入循环。

---

## 支持的 Mock 类型

| 依赖类型 | 识别模式 | Mock 方案 |
|---|---|---|
| SQL 数据库 | `database/sql`, `gorm.io/gorm` | sqlmock |
| HTTP 客户端 | `net/http` | testify/mock |
| Redis | `go-redis/redis` | testify/mock |
| gRPC | `google.golang.org/grpc` | testify/mock |

---

## 技术栈

- **Go 1.22+**
- **[testify](https://github.com/stretchr/testify)** — Mock 和断言框架
- **[sqlmock](https://github.com/DATA-DOG/go-sqlmock)** — SQL 数据库 Mock
- **`go/ast` + `go/parser`** — Go AST 解析

---

## License

MIT © 2026 Hollow
