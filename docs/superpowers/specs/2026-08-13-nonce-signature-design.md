# 签名验签增加随机参数 nonce 设计文档

## 背景

当前下载代理的 HMAC 签名仅基于 `url` 和 `time`：

```
url=<target-url>&time=<unix-seconds>
```

同一目标 URL 在同一时间戳下生成的签名完全相同，存在被重放或共享链接的风险。需要引入随机参数 `nonce`，使每次请求的签名唯一，并在服务端对 nonce 进行去重校验。

## 目标

1. 每次签名的请求必须携带一个随机字符串 `nonce`。
2. `nonce` 参与 HMAC-SHA256 签名计算。
3. 服务端在 5 分钟窗口内拒绝重复的 `nonce`。
4. 不兼容旧链接：不带 `nonce` 的请求直接拒绝。

## 非目标

- 多实例共享 nonce 状态（本项目为单二进制，暂不需要 Redis 等外部存储）。
- 对 `nonce` 的格式做额外限制（长度、字符集由调用方自行保证足够随机即可）。

## 设计方案

### 1. 签名字符串

```
url=<target-url>&time=<unix-seconds>&nonce=<nonce>
```

### 2. 接口变更

`internal/security/sign.go`：

```go
func Sign(url string, timestamp int64, nonce string, secret string) string
func Verify(url string, timestamp int64, nonce string, sign string, secret string, maxExpire int64, store NonceStore) bool
```

新增接口：

```go
type NonceStore interface {
    // Seen 返回 true 表示该 nonce 在最近窗口内已经出现过。
    // 同时会记录本次 nonce，供后续调用判断。
    Seen(nonce string) bool
}
```

### 3. nonce 去重实现

采用**时间分桶缓存**：

- 以分钟为粒度分桶，每个桶是一个 `map[string]struct{}`。
- 保留当前分钟及前 4 分钟共 5 个桶，覆盖约 5 分钟窗口。
- `Seen()` 被调用时：
  1. 获取当前时间，确定当前桶。
  2. 清理早于窗口的桶。
  3. 在所有活跃桶中查找 nonce；找到则返回 `true`。
  4. 未找到则写入当前桶，返回 `false`。
- 使用 `sync.Mutex` 保证并发安全。

优点：
- 无需后台 goroutine，清理在调用时隐式完成。
- 内存占用稳定，最多保留 5 个桶。

### 4. Handler 变更

`internal/proxy/proxy.go`：

- 从 query 参数读取 `nonce`。
- `nonce` 缺失或为空时返回 `400 Bad Request`。
- 将 `nonce` 和 `NonceStore` 传给 `security.Verify`。
- `Verify` 失败（签名错误、时间无效、nonce 重复）返回 `403 Forbidden`。

`Proxy` 结构体持有 `NonceStore` 实例，在 `proxy.New()` 中初始化。

### 5. 测试

- 更新 `internal/security/sign_test.go`：
  - 调整现有用例的 `Sign`/`Verify` 调用，补充 `nonce`。
  - 新增用例验证重复 nonce 被拒绝、超过窗口后 nonce 可复用。
- 更新 `internal/proxy/proxy_test.go`：
  - 所有测试请求补充 `nonce` 参数。
  - 新增缺失 nonce 返回 400 的测试。

### 6. 文档

更新 `AGENTS.md` 中的“Signature format”段落，说明新的签名格式。

## 风险与考虑

- **内存占用**：极端情况下 5 分钟内大量唯一 nonce 会占用内存。对内部/小规模使用可接受；如流量很大，可改用外部存储或限制 nonce 数量。
- **时钟同步**：`Verify` 已有 ±3 分钟时钟偏移容忍，nonce 窗口基于相同时间源。
- **向后不兼容**：旧链接失效，需通知调用方升级签名生成逻辑。
