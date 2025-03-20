
[English](./README.md) | 简体中文

# AI Proxy

新一代 AI 网关，使用 OpenAI 作为协议入口。

## 新功能

- 智能错误重试
- 基于优先级与错误率选择渠道
- 告警通知
  - 渠道余额预警
  - 错误率预警
  - 无权限渠道预警
  - 更多...
- 日志与审计
  - 完善的请求日志数据
  - 请求体、响应体记录
  - 请求日志链路追踪
- 数据统计分析
  - 请求量统计
  - 错误量统计
  - RPM TPM 统计
  - 消费统计
  - 模型统计
  - 渠道错误率分析
  - 更多...
- Rerank 支持
- PDF 支持
- STT 模型映射支持
- 多租户系统分离
- 模型 RPM TPM 限制
- Think 模型支持 `<think>` 切分到 `reasoning_content`
- 提示词缓存计费支持
- 内敛分词器，无需额外下载 tiktoken 文件
- API `Swagger` 文档支持 `http://host:port/swagger/index.html`

## 部署

## 使用 Docker

```bash
docker run -d --name aiproxy -p 3000:3000 -v $(pwd)/aiproxy:/aiproxy ghcr.io/labring/aiproxy:latest
```

## 使用 Docker Compose

将 [docker-compose.yaml](./docker-compose.yaml) 复制到目录。

```bash
docker-compose up -d
```

## 环境变量

### 基础配置

- `LISTEN`: 监听地址，默认 `:3000`
- `ADMIN_KEY`: 管理员密钥，用于管理 API 和转发 API，默认空
- `INTERNAL_TOKEN`: 内部服务认证 token，默认空
- `FFPROBE_ENABLED`: 是否启用 ffprobe，默认 `false`

### Debug 选项

- `DEBUG`: 启用调试模式，默认 `false`
- `DEBUG_SQL`: 启用 SQL 调试，默认 `false`

### 数据库选项

- `SQL_DSN`: 数据库连接字符串，默认空，eg: `postgres://postgres:postgres@localhost:5432/postgres`
- `LOG_SQL_DSN`: 日志数据库连接字符串，默认空，eg: `postgres://postgres:postgres@localhost:5432/postgres`
- `REDIS_CONN_STRING`: Redis 连接字符串，默认空，eg: `redis://localhost:6379`
- `DISABLE_AUTO_MIGRATE_DB`: 禁用自动数据库迁移，默认 `false`
- `SQL_MAX_IDLE_CONNS`: 数据库最大空闲连接数，默认 `100`
- `SQL_MAX_OPEN_CONNS`: 数据库最大打开连接数，默认 `1000`
- `SQL_MAX_LIFETIME`: 数据库连接最大生命周期，默认 `60`
- `SQLITE_PATH`: SQLite 数据库路径，默认 `aiproxy.db`
- `SQL_BUSY_TIMEOUT`: 数据库繁忙超时时间，默认 `3000`

### 通知选项

- `NOTIFY_NOTE`: 自定义通知备注，默认 `AI Proxy`
- `NOTIFY_FEISHU_WEBHOOK`: 飞书通知 webhook url，默认空，eg: `https://open.feishu.cn/open-apis/bot/v2/hook/xxxx`

### 模型配置

- `DISABLE_MODEL_CONFIG`: 禁用模型配置，默认 `false`
- `RETRY_TIMES`: 重试次数，默认 `0`
- `ENABLE_MODEL_ERROR_AUTO_BAN`: 启用模型错误自动禁用，默认 `false`
- `MODEL_ERROR_AUTO_BAN_RATE`: 模型错误自动禁用阈值，默认 `0.3`
- `TIMEOUT_WITH_MODEL_TYPE`: 不同模型类型超时设置，默认 `{}`
- `DEFAULT_CHANNEL_MODELS`: 每个渠道默认模型，默认 `{}`
- `DEFAULT_CHANNEL_MODEL_MAPPING`: 每个渠道模型映射，默认 `{}`

### 日志配置

- `LOG_STORAGE_HOURS`: 日志存储时间（0 表示不限），默认 `0`
- `SAVE_ALL_LOG_DETAIL`: 保存所有日志详情，默认 `false` 则只保存错误日志
- `LOG_DETAIL_REQUEST_BODY_MAX_SIZE`: 日志详情请求体最大大小，默认 `128KB`
- `LOG_DETAIL_RESPONSE_BODY_MAX_SIZE`: 日志详情响应体最大大小，默认 `128KB`
- `LOG_DETAIL_STORAGE_HOURS`: 日志详情存储时间，默认 `72`（3 天）

### 服务控制

- `DISABLE_SERVICE_CONTROL`: 禁用服务控制，默认 `false`
- `GROUP_MAX_TOKEN_NUM`: 每个组最大 token 数量（0 表示不限），默认 `0`
- `GROUP_CONSUME_LEVEL_RATIO`: 每个组消费等级比例，默认 `{}`
- `GEMINI_SAFETY_SETTING`: Gemini 模型安全设置，默认 `BLOCK_NONE`
- `BILLING_ENABLED`: 启用计费功能，默认 `true`
