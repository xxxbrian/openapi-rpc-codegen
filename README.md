# openapi-rpc-codegen

一个有明确约束、偏 RPC 风格的 OpenAPI 代码生成器，目前可以生成：

- Go 后端（net/http + chi）
- TypeScript 客户端（面向微信小程序 wx.request）

本项目不追求“支持 OpenAPI 的一切写法”，而是刻意限制 OpenAPI 的自由度，换取：

- 简单
- 可预测
- 强类型
- 易维护
- 工程一致性

## 设计目标

从 OpenAPI 描述中，生成一套前后端一致的 RPC 接口代码

核心理念：

- OpenAPI 只是 IDL（接口描述语言）
- RPC 是主要抽象（而不是 HTTP 动词本身）
- 前后端通过 同一份 schema 保持类型一致
- 生成的代码 可直接进入生产工程

## OpenAPI 约束

本工具 只支持以下 OpenAPI 子集：

支持：

- OpenAPI 3.0.x
- HTTP 方法：`GET`、`POST`
- 成功响应：只允许一个 `200`
- Content-Type：`application/json`
- `components.schemas` 中的对象定义
- `type: object | string | number | integer | boolean | array`
- `nullable`
- `$ref`（强烈推荐）

不支持（刻意不支持）

- `oneOf / allOf / anyOf`
- `additionalProperties`
- 多个 success response（如 201 / 204）
- 非 JSON（form / multipart / text）
- 复杂 content negotiation
- REST 语义分支（PUT / PATCH / DELETE）

## 架构

```
OpenAPI YAML
     │
     ▼
 OpenAPI Parser
     │
     ▼
     IR (中间表示，语言无关)
     │
     ├─────────► ts-wx emitter   → 微信小程序/小游戏客户端
     │
     └─────────► go-server emitter → Go HTTP Server
```

引入 IR 抽象层是因为OpenAPI 非常自由，不适合直接生成代码，而 IR 是 “带偏见的、固执的抽象层”，IR 保证了生成代码的简单性和一致性。

## 生成产物

### Go 后端 （`go-server` emitter）

- `go-server/types.gen.go`
- `go-server/server.gen.go`
- `go-server/transport.go`

只需要实现：

```go
type UserService struct{}

func (UserService) GetUser(ctx context.Context, path GetUserPath) (User, error) {
    ...
}
```

### TypeScript 微信小程序端（`ts-wx` emitter）

- `ts-wx/types.gen.ts`
- `ts-wx/client.gen.ts`
- `ts-wx/transport.ts`

使用示例：

```ts
const api = makeApi("https://api.example.com");

const user = await api.User.getUser({ id: "123" });
```

## 使用方法

### 编译

```
go build ./cmd/openapi-rpc-codegen
```

### 运行

```
./openapi-rpc-codegen \
  --spec api/openapi.yaml \
  --out ./generated \
  --targets go-server,ts-wx
```

输出结构示例：

```
generated/
  go/server/
    types.gen.go
    server.gen.go
    transport.go
  ts/wx/
    types.gen.ts
    client.gen.ts
    transport.ts
```
