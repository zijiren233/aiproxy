# Deploy

## Use Docker Compose

Copy [docker-compose.yaml](./docker-compose.yaml) to the same directory as the `aiproxy` binary.

```bash
docker-compose up -d
```

## Envs

### Basic Configuration

- `ADMIN_KEY`: The admin key for the AI Proxy Service, admin key is used to admin api and relay api, default is empty
- `SQL_DSN`: The database connection string, default is empty
- `LOG_SQL_DSN`: The log database connection string, default is empty
- `REDIS_CONN_STRING`: The redis connection string, default is empty
- `INTERNAL_TOKEN`: Internal token for service authentication, default is empty
- `FFPROBE_ENABLED`: Whether to enable ffprobe, default is `false`

### Debug Options

- `DEBUG`: Enable debug mode, default is `false`
- `DEBUG_SQL`: Enable SQL debugging, default is `false`

### Database Options

- `DISABLE_AUTO_MIGRATE_DB`: Disable automatic database migration, default is `false`
- `SQL_MAX_IDLE_CONNS`: The maximum number of idle connections in the database, default is `100`
- `SQL_MAX_OPEN_CONNS`: The maximum number of open connections to the database, default is `1000`
- `SQL_MAX_LIFETIME`: The maximum lifetime of a connection in seconds, default is `60`

### Notify Options

- `NOTIFY_NOTE`: Custom notification note, default is empty
- `NOTIFY_FEISHU_WEBHOOK`: The feishu notify webhook url, default is empty

### Model Configuration

- `DISABLE_MODEL_CONFIG`: Disable model configuration, default is `false`
- `RETRY_TIMES`: Number of retry attempts, default is determined at runtime
- `ENABLE_MODEL_ERROR_AUTO_BAN`: Enable automatic banning of models with errors, default is determined at runtime
- `MODEL_ERROR_AUTO_BAN_RATE`: Rate threshold for auto-banning models with errors, default is `0.3`
- `TIMEOUT_WITH_MODEL_TYPE`: Timeout settings for different model types, default is empty map
- `DEFAULT_CHANNEL_MODELS`: Default models for each channel, default is empty map
- `DEFAULT_CHANNEL_MODEL_MAPPING`: Model mapping for each channel, default is empty map

### Logging Configuration

- `LOG_STORAGE_HOURS`: Hours to store logs (0 means no limit), default is `0`
- `SAVE_ALL_LOG_DETAIL`: Save all log details, default is determined at runtime
- `LOG_DETAIL_REQUEST_BODY_MAX_SIZE`: Maximum size for request body in log details, default is `128KB`
- `LOG_DETAIL_RESPONSE_BODY_MAX_SIZE`: Maximum size for response body in log details, default is `128KB`
- `LOG_DETAIL_STORAGE_HOURS`: Hours to store log details, default is `72` (3 days)

### Service Control

- `DISABLE_SERVE`: Disable serving requests, default is determined at runtime
- `GROUP_MAX_TOKEN_NUM`: Maximum number of tokens per group (0 means unlimited), default is determined at runtime
- `GROUP_CONSUME_LEVEL_RATIO`: Consumption level ratio for groups, default is determined at runtime
- `GEMINI_SAFETY_SETTING`: Safety setting for Gemini models, default is `BLOCK_NONE`
- `BILLING_ENABLED`: Enable billing functionality, default is `true`
