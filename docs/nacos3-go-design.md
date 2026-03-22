# Dubbo Admin Go 端 Nacos 3 支持设计

## 1. 背景

- 关联 issue: `https://github.com/apache/dubbo-admin/issues/1413`
- 当前 Go 端仅支持 `nacos2` 类型发现源，相关实现集中在：
  - `pkg/config/discovery/config.go`
  - `pkg/core/clients/nacos.go`
  - `pkg/discovery/nacos2/**`
  - `pkg/governor/nacos2/**`
- Java 侧 Nacos 接入的核心思路是分开处理“服务映射 / 服务发现”和“配置中心 / 元数据中心”，Go 端当前 `nacos2` 也是同样的结构：
  - Naming client 负责服务列表、实例订阅
  - Config client 负责规则、元数据、映射读取与治理写回

从能力边界上看，Go 端距离 Nacos 3 并不远，主要缺口不是资源模型，而是 Nacos client 的构建方式、配置类型注册，以及少量 Nacos 3 兼容项没有显式承接。

### 1.1 参考依据

本设计主要参考以下信息：

1. Java 侧现有 Nacos 接入分层
   - `upstream/java:dubbo-admin-server/src/main/java/org/apache/dubbo/admin/registry/mapping/impl/NacosServiceMapping.java`
   - `upstream/java:dubbo-admin-server/src/main/java/org/apache/dubbo/admin/registry/metadata/impl/NacosMetaDataCollector.java`

2. 当前 Go 侧 Nacos 2 实现
   - `pkg/discovery/nacos2/**`
   - `pkg/governor/nacos2/**`
   - `pkg/core/clients/nacos.go`

3. Nacos 官方 3.0 兼容变化
   - `https://github.com/alibaba/nacos/releases/tag/3.0.0`
   - 其中与 Go 端最相关的是 public namespace 默认值变化，以及 Naming 侧 gRPC 通道要求

## 2. 现状分析

### 2.1 已有能力

当前 `nacos2` 已经覆盖了 Go 管理面的核心链路：

1. 服务发现
   - `pkg/discovery/nacos2/listerwatcher/nacos_service.go`
   - 周期性分页拉取服务列表
   - 为每个服务建立订阅，接收实例变更
   - 将实例数据转换为 `NacosServiceResource`

2. 配置与元数据同步
   - `pkg/discovery/nacos2/listerwatcher/nacos_config.go`
   - 从 Nacos `dubbo` / `mapping` group 拉取规则和元数据
   - 监听配置变更并投递到资源事件总线

3. 治理写回
   - `pkg/governor/nacos2/governor.go`
   - 支持动态配置、条件路由、标签路由的发布、更新、删除

4. 资源订阅后处理
   - `pkg/core/discovery/subscriber/nacos_service.go`
   - 把 `NacosServiceResource` 转换为应用、消费者、RPC 实例等管理面资源

结论：Nacos 3 支持不需要新增资源类型，也不需要重写整个 Discovery / Governor 框架。

### 2.2 当前阻塞点

当前 Go 端对 Nacos client 的构建主要依赖 `dubbo-go` 的 URL builder：

- `pkg/core/clients/nacos.go`
- `pkg/discovery/nacos2/factory.go`

这会带来几个问题：

1. `nacos2` 被写死为唯一的 Nacos 类型
   - `pkg/config/discovery/config.go` 只有 `Nacos2`
   - `pkg/core/bootstrap/init.go` 只注册 `pkg/discovery/nacos2` 和 `pkg/governor/nacos2`
   - `pkg/core/discovery/component.go` 只在 `item.Type == discovery.Nacos2` 时加载 `NacosServiceEventSubscriber`

2. client 构建逻辑重复
   - `pkg/core/clients/nacos.go` 和 `pkg/discovery/nacos2/factory.go` 各自创建一套 Nacos client
   - Nacos 3 适配如果继续在两处复制，后续维护成本高

3. 当前 builder 无法显式承接 Nacos 3 的关键连接参数
   - `dubbo-go/v3/remoting/nacos/builder.go` 只解析 `host:port`、`contextPath`、基础认证等
   - 没有为 Dubbo Admin 暴露显式 `grpcPort` 配置入口
   - 对 Config client 的 namespace 默认值也没有为 Nacos 3 做兼容兜底

4. 配置中心链路存在 Nacos 3 namespace 风险
   - `nacos-sdk-go/v2` 的 Naming client 会在 namespace 为空时回落到 `public`
   - 但 Config client 链路不会自动补这个默认值
   - Nacos 3 默认 public namespace 的 ID 为 `public`，如果 Go 端仍以空字符串访问配置中心，治理规则和元数据读取可能异常

## 3. Nacos 3 兼容点

本次设计需要显式覆盖以下 Nacos 3 差异：

1. 命名服务依赖 gRPC 通道
   - `nacos-sdk-go/v2` 已支持 gRPC
   - 但 Dubbo Admin Go 端需要把 `grpcPort` 作为显式可配置项承接下来，而不是只能依赖 `port + 1000`

2. public namespace 默认值变化
   - Nacos 3 下默认 public namespace 的 ID 为 `public`
   - Go 端 Config client 需要在 Nacos 3 模式下补齐默认 namespace，避免配置中心查询落空

3. 继续使用官方 SDK，而不是旧 OpenAPI 兜底
   - Go 端现有实现已经主要基于 `nacos-sdk-go/v2`
   - 这是正确方向，Nacos 3 适配应继续围绕 SDK，而不是增加对旧管理 OpenAPI 的依赖

4. 保留 Nacos 2 兼容
   - 现网可能仍有 `nacos2` 配置
   - 新增 Nacos 3 支持时不能破坏现有 `nacos2` 行为

## 4. 目标与非目标

### 4.1 目标

本次 Go 端 Nacos 3 支持应完成：

1. 新增 `nacos3` discovery type
2. 支持从 Nacos 3 读取：
   - 服务列表
   - 服务实例变更
   - 动态配置
   - 条件路由
   - 标签路由
   - 服务提供者元数据
   - 服务映射
3. 支持向 Nacos 3 写回治理规则：
   - 创建
   - 更新
   - 删除
4. 提供可落地的配置方式和示例
5. 保持现有 `nacos2` 行为不回退

### 4.2 非目标

本次不做：

1. 不新增新的资源模型或 Store 类型
2. 不改 UI 数据模型
3. 不引入基于 Nacos 管理 OpenAPI 的新实现
4. 不在第一阶段支持每个 server 节点独立配置不同 `grpcPort`

## 5. 推荐方案

推荐采用“外部区分 `nacos2` / `nacos3`，内部复用同一套 Nacos 读写逻辑”的方案。

### 5.1 外部兼容策略

- 对用户配置新增 `type: nacos3`
- 保留现有 `type: nacos2`
- 两者共享相同的资源同步与治理能力
- 差异仅体现在 client 构建和默认参数处理上

这样做的好处：

1. 对升级路径清晰
   - 用户可以明确知道当前发现源是 Nacos 2 还是 Nacos 3

2. 对兼容性风险可控
   - Nacos 3 专属默认值不会误伤现有 Nacos 2

3. 对代码结构友好
   - Nacos 读写逻辑可以抽公共层
   - `factory` 只负责选择 type 和注入不同 client 配置

### 5.2 内部实现策略

核心改动不是复制整套 `nacos2` 包，而是抽出公共 Nacos client 创建逻辑，并让 `nacos2` / `nacos3` 复用现有 lister-watcher 和 governor 行为。

推荐拆分为：

1. 公共 client 构建层
   - 统一从 URL 构造 `nacos-sdk-go/v2` 的 `ServerConfig` 和 `ClientConfig`
   - 统一创建 Config client / Naming client

2. 轻量 discovery/governor factory
   - `nacos2` 和 `nacos3` 只负责声明支持的 type
   - 真正的 list-watch / rule-governor 逻辑复用现有实现

3. Nacos 3 特有默认值注入
   - 主要是 namespace 默认值与 gRPC 端口处理

## 6. 需要改动的模块

### 6.1 配置类型层

文件：

- `pkg/config/discovery/config.go`
- `app/dubbo-admin/dubbo-admin.yaml`

改动：

1. 在 `pkg/config/discovery/config.go` 中新增：
   - `Nacos3 Type = "nacos3"`

2. 在样例配置中新增 `nacos3` 说明
   - discovery type 说明从 `nacos2, zookeeper, mock` 更新为 `nacos2, nacos3, zookeeper, mock`
   - 补充 Nacos 3 推荐参数示例

3. 保持地址格式仍然为 URL
   - 不改现有 `registry/configCenter/metadataReport` 结构
   - 避免影响已有配置模型和解析流程

推荐新增的 URL query 参数约定：

- `namespace`
- `username`
- `password`
- `grpcPort`
- `contextPath`
- `scheme`

示例：

```yaml
discovery:
  - type: nacos3
    id: nacos3-public
    name: nacos3-public
    address:
      registry: nacos://127.0.0.1:8848?nacosVersion=3&namespace=public&username=nacos&password=nacos&grpcPort=9848
      configCenter: nacos://127.0.0.1:8848?nacosVersion=3&namespace=public&username=nacos&password=nacos&grpcPort=9848
      metadataReport: nacos://127.0.0.1:8848?nacosVersion=3&namespace=public&username=nacos&password=nacos&grpcPort=9848
```

注：

- `nacosVersion=3` 可以不作为必须字段；是否保留仅用于日志和排障，可在实现阶段决定
- 真正的行为开关应由 `type: nacos3` 决定，而不是由 URL query 决定

### 6.2 Nacos client 构建层

文件：

- `pkg/core/clients/nacos.go`
- `pkg/discovery/nacos2/factory.go`

建议新增/重构：

- 把现有 `pkg/core/clients/nacos.go` 重构为公共 Nacos client builder
- `pkg/discovery/nacos2/factory.go` 不再自己直接调用 `dubbo-go` builder，而是复用公共 builder

具体改动：

1. 不再依赖 `dubbo-go/remoting/nacos` 作为 Admin 侧唯一构建入口
   - 改为直接使用 `nacos-sdk-go/v2/clients`
   - 原因是需要精确控制 `ServerConfig.GrpcPort`、`Scheme`、`ContextPath` 和 `ClientConfig.NamespaceId`

2. 新增统一入口，例如：
   - `CreateNacosClientsByType(t discoverycfg.Type, rawURL string, clientName string)`

3. builder 需要完成：
   - 解析 `registry/configCenter/metadataReport` URL
   - 构造 `[]constant.ServerConfig`
   - 构造 `constant.ClientConfig`
   - 当 type 为 `nacos3` 且 namespace 为空时，默认补 `public`
   - 当 `grpcPort` 未显式配置时，沿用 `port + 1000`

4. 统一错误信息
   - 在错误信息中打印 discovery id、访问地址、当前 type
   - 便于区分 Nacos 2 / Nacos 3 问题

这里是本次设计的核心改动点。

### 6.3 Discovery factory 注册层

文件：

- `pkg/core/bootstrap/init.go`
- `pkg/discovery/nacos2/factory.go`
- 新增 `pkg/discovery/nacos3/factory.go`

改动：

1. 新增 `pkg/discovery/nacos3/factory.go`
   - `Support(d discoverycfg.Type) bool { return d == discoverycfg.Nacos3 }`
   - 内部复用现有 Nacos list-watcher 初始化流程

2. 更新 bootstrap 注册
   - 在 `pkg/core/bootstrap/init.go` 中新增：
     - `_ "github.com/apache/dubbo-admin/pkg/discovery/nacos3"`

3. `nacos2` factory 和 `nacos3` factory 共享同一套公共 builder

说明：

- `pkg/discovery/nacos2/listerwatcher/nacos_service.go`
- `pkg/discovery/nacos2/listerwatcher/nacos_config.go`

这两个文件的业务逻辑本身不需要因为 Nacos 3 重写；它们依赖的只是 Naming client 和 Config client 接口。只要 client 构造正确，绝大部分逻辑可以直接复用。

### 6.4 Governor factory 注册层

文件：

- `pkg/governor/nacos2/factory.go`
- `pkg/governor/nacos2/governor.go`
- 新增 `pkg/governor/nacos3/factory.go`

改动：

1. 新增 `pkg/governor/nacos3/factory.go`
   - 负责注册 `nacos3` 类型 governor

2. 治理逻辑复用现有 `RuleGovernor`
   - `PublishConfig`
   - `DeleteConfig`
   - `GetConfig`

3. `RuleGovernor` 需要通过公共 builder 获取 Nacos 3 Config client
   - 不能继续只调用旧的 `CreateNacosClients(cfg.Address.ConfigCenter)`

### 6.5 Discovery subscriber 注册逻辑

文件：

- `pkg/core/discovery/component.go`

改动：

当前代码只在存在 `Nacos2` discovery 时追加 `NacosServiceEventSubscriber`：

- 需要改为 `Nacos2 || Nacos3`

否则即便 `nacos3` 的 list-watcher 正常工作，`NacosServiceResource` 也不会被转换为应用、消费者和实例资源，管理面上仍然是“看不到效果”。

### 6.6 文档与示例

文件：

- `app/dubbo-admin/dubbo-admin.yaml`
- 可能需要更新 `README.md` 或后续补充运行文档

改动：

1. 增加 `nacos3` discovery 示例
2. 明确说明：
   - Nacos 3 应连服务端口而不是 console 端口
   - 默认 namespace 推荐写成 `public`
   - 如果 gRPC 端口不是 `8848 + 1000`，必须显式传 `grpcPort`

## 7. 功能交付清单

本次完成后，Go 端应具备以下能力：

1. `type: nacos3` 的配置可正常启动
2. 首页和服务列表可看到 Nacos 3 注册的应用与实例
3. 服务消费者、服务映射、提供者元数据可正常同步
4. 动态配置、条件路由、标签路由可正常读取
5. 从 Admin UI 发起治理变更后，可成功写回 Nacos 3 配置中心
6. Nacos 2 的已有能力保持可用

## 8. 风险与边界

### 8.1 主要风险

1. namespace 兼容风险
   - 如果不处理 Config client 默认 namespace，Nacos 3 下规则同步和治理写回会出现“服务能看见，规则读不到”的割裂现象

2. gRPC 端口不等于 `port + 1000`
   - 某些代理、LB 或网关环境下可能成立
   - 因此必须支持显式 `grpcPort`

3. 多节点独立 gRPC 端口配置
   - 当前 URL 方案天然适合“同构集群”
   - 如果未来遇到每个节点端口不一致的部署，需要升级为结构化 server config

### 8.2 本阶段边界

本阶段默认以下前提成立：

1. Nacos 3 集群的 HTTP 服务地址可直接访问
2. gRPC 端口对 Dubbo Admin 可达
3. 同一个 discovery 下各节点端口规则一致

## 9. 测试与验收

### 9.1 单元测试

建议新增以下测试：

1. client builder URL 解析测试
   - 带 `namespace=public`
   - namespace 为空但 type 为 `nacos3`
   - 带 `grpcPort`
   - 多地址
   - 带 `contextPath`
   - 带认证参数

2. `Support()` / factory 注册测试
   - `nacos2`
   - `nacos3`

3. subscriber 激活条件测试
   - 当 discovery 中存在 `nacos3` 时，`NacosServiceEventSubscriber` 会被注册

### 9.2 集成验证

建议最少覆盖以下场景：

1. Nacos 3 standalone，namespace=`public`
   - 能拉到服务
   - 能收到实例变更
   - 能读取配置和元数据

2. Nacos 3 自定义 namespace
   - 能读取对应 namespace 的服务和规则

3. 从 Admin 创建 / 更新 / 删除一条治理规则
   - Nacos 3 配置中心可见对应变更

4. 回归 Nacos 2
   - 现有 discovery 与 governance 功能不回退

## 10. 实施顺序

建议按下面顺序实施：

1. 新增 `nacos3` type，并补充样例配置
2. 重构公共 Nacos client builder，先打通 Nacos 3 Config client / Naming client
3. 新增 `pkg/discovery/nacos3/factory.go`
4. 新增 `pkg/governor/nacos3/factory.go`
5. 更新 `pkg/core/discovery/component.go` 的 subscriber 激活条件
6. 补充单元测试和集成验证说明

## 11. 预期代码改动清单

必改：

- `pkg/config/discovery/config.go`
- `pkg/core/clients/nacos.go`
- `pkg/core/bootstrap/init.go`
- `pkg/core/discovery/component.go`
- `pkg/discovery/nacos2/factory.go`
- `pkg/governor/nacos2/governor.go`
- `app/dubbo-admin/dubbo-admin.yaml`

新增：

- `pkg/discovery/nacos3/factory.go`
- `pkg/governor/nacos3/factory.go`

大概率无需改动：

- `pkg/discovery/nacos2/listerwatcher/nacos_service.go`
- `pkg/discovery/nacos2/listerwatcher/nacos_config.go`
- `pkg/core/discovery/subscriber/nacos_service.go`
- `api/mesh/v1alpha1/**`
- `pkg/core/resource/apis/mesh/v1alpha1/**`

## 12. 结论

Go 端支持 Nacos 3 的重点不在“补一个新资源种类”，而在于把当前仅面向 `nacos2` 的注册方式和 client 构建方式升级成“类型可扩展、连接参数可控、namespace 默认值正确”的实现。

如果按本文方案推进，Go 端可以在保持现有 `nacos2` 兼容的前提下，以较小改动完成 Nacos 3 的服务发现、配置同步和治理写回支持。
