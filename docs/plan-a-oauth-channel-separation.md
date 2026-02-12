# 方案A：OAuth与Channel分离架构设计

## 架构变更

### 当前架构
```
用户请求 -> Group -> GroupItem -> Channel -> 认证凭据 -> API请求
                                              ├─ API Key模式
                                              └─ OAuth模式（通过Channel包装）
```

### 新架构（方案A）
```
用户请求 -> Group -> GroupItem
                            ├─ Type: "channel" -> Channel -> 认证凭据 -> API请求
                            └─ Type: "oauth" -> OAuthToken -> API请求
```

## 数据库表结构变更

### 1. OAuth Token 表增加字段

```go
type OAuthToken struct {
    // === 现有字段 ===
    ID           uint           `json:"id" gorm:"primaryKey"`
    Type         OAuthTokenType `json:"type" gorm:"index;not null"`
    Email        string         `json:"email" gorm:"index"`
    AccessToken  string         `json:"-" gorm:"type:text"`
    RefreshToken string         `json:"-" gorm:"type:text"`
    IDToken      string         `json:"-" gorm:"type:text"`
    ExpiresAt    *time.Time     `json:"expires_at"`
    LastRefresh  *time.Time     `json:"last_refresh"`
    CreatedAt    time.Time      `json:"created_at"`
    UpdatedAt    time.Time      `json:"updated_at"`
    Enabled      bool           `json:"enabled" gorm:"default:true"`
    Remark       string         `json:"remark"`

    // === 新增字段 ===
    BaseURLs    []string        `json:"base_urls" gorm:"serializer:json"`      // API endpoints
    Models       []string        `json:"models" gorm:"serializer:json"`       // 可用模型列表
    Provider     string         `json:"provider" gorm:"index"`              // "openai", "google"等
}
```

### 2. GroupItem 表增加Type字段

```go
type GroupItem struct {
    ID          int    `json:"id" gorm:"primaryKey"`
    GroupID     int    `json:"group_id" gorm:"not null;index:idx_group_channel_model,unique"`

    // === 新增字段 ===
    Type        string `json:"type" gorm:"index;not null"`           // "channel" 或 "oauth"

    // === 修改为可选字段 ===
    ChannelID   *int    `json:"channel_id,omitempty" gorm:"not null;index:idx_group_channel_model,unique"`
    OAuthTokenID *int   `json:"oauth_token_id,omitempty" gorm:"not null;index:idx_group_channel_model,unique"`

    ModelName    string `json:"model_name" gorm:"not null;index:idx_group_channel_model,unique"`
    Priority     int    `json:"priority"`
    Weight       int    `json:"weight"`
}
```

**约束：**
- Type = "channel" 时，ChannelID 必填，OAuthTokenID 为空
- Type = "oauth" 时，OAuthTokenID 必填，ChannelID 为空
- 两者互斥，通过Type字段区分

### 3. 数据库迁移

```sql
-- 步骤1：添加Type字段
ALTER TABLE group_items ADD COLUMN type VARCHAR(10) NOT NULL DEFAULT 'channel';

-- 步骤2：添加OAuthTokenID字段
ALTER TABLE group_items ADD COLUMN oauth_token_id INT NULL;

-- 步骤3：更新现有数据（可选）
-- 所有现有数据的Type设为'channel'，保持向后兼容
UPDATE group_items SET type = 'channel' WHERE type IS NULL;

-- 步骤4：创建索引（提高查询性能）
CREATE INDEX idx_group_item_type ON group_items(type, oauth_token_id);
```

## 代码实现要点

### 1. OAuth登录后自动获取模型

**位置：** `internal/auth/codex/auth.go` 和 `internal/auth/antigravity/auth.go`

```go
// Token exchange成功后
if tokenResp, claims, err := codexAuth.ExchangeCodeForTokens(ctx, code, pkceCodes); err != nil {
    return err
}

// 获取可用模型列表
client, _ := client.GetHTTPClientSystemProxy(false)
models, err := helper.FetchOpenAIModels(client, ctx, channel)
if err != nil {
    log.Warnf("Failed to fetch models: %v", err)
    models = []string{}  // 失败时使用空列表，但不阻塞OAuth流程
}

// 创建OAuth Token
oauthToken := codex.CreateOAuthToken(tokenResp, claims)
oauthToken.Models = models     // 存储模型列表
oauthToken.Provider = "openai"  // 存储provider
```

### 2. Relay请求逻辑

**位置：** `internal/relay/relay.go`

```go
// 当前：GroupItem只包含ChannelID
channel, _ := op.ChannelGet(item.ChannelID, ctx)
credential := channel.GetAuthCredential()

// 方案A：根据GroupItem.Type决定
if item.Type == "oauth" {
    // OAuth Token模式：直接使用OAuth Token
    token, _ := op.OAuthTokenGet(item.OAuthTokenID, ctx)

    // 验证Token有效性
    if token.IsExpired() {
        iter.Skip(channel.ID, 0, channel.Name, "oauth token expired")
        continue
    }

    // 获取适配器
    adapter := outbound.Get(token.Provider)  // "openai" -> OpenAIChat

    // 构造请求（使用token.AccessToken）
    req := &relayRequest{
        c:         c,
        inbound:    inAdapter,
        internalRequest: request,
        iter:        iter,
        authToken:  token,  // 直接使用OAuth Token
    }

} else {
    // Channel模式：使用Channel（保持现有逻辑）
    channel, _ := op.ChannelGet(item.ChannelID, ctx)
    credential := channel.GetAuthCredential()
    // ... 现有代码
}
```

### 3. Helper模型获取API

**已实现：** `internal/helper/fetch.go`

```go
// OpenAI模型列表
func FetchOpenAIModels(client *http.Client, ctx context.Context, request model.Channel) ([]string, error) {
    req, _ := http.NewRequestWithContext(
        ctx,
        http.MethodGet,
        request.GetBaseUrl()+"/models",  // 使用channel的base_url
        nil,
    )
    req.Header.Set("Authorization", "Bearer "+request.GetChannelKey().ChannelKey)

    resp, err := client.Do(req)
    // ... 解析响应，返回模型ID列表
}
```

## 前端UI调整

### GroupItem表单

```tsx
// 选择认证类型
<Select value={item.type || 'channel'} onChange={(v) => setType(v)}>
  <SelectItem value="channel">Channel</SelectItem>
  <SelectItem value="oauth">OAuth Token</SelectItem>
</Select>

{item.type === 'channel' && (
  // Channel选择器
  <ChannelSelect
    value={item.channel_id}
    onChange={(id) => setChannelId(id)}
    filters={(ch) => ch.Type === type}  // 根据类型过滤
  />
)}

{item.type === 'oauth' && (
  // OAuth Token选择器
  <OAuthTokenSelect
    value={item.oauth_token_id}
    onChange={(id) => setOAuthTokenId(id)}
    filters={(token) =>
      token.Enabled &&
      !token.IsExpired &&
      (type === 'oauth_codex' ? token.type === 'codex' : token.type === 'antigravity')
    }
  />
)}
```

### Channel创建表单

```tsx
// 选择OAuth Token后，自动填充可用模型
<OAuthTokenSelect
  value={oauth_token_id}
  onChange={(id) => {
    const token = oauthTokens.find(t => t.id === id)
    if (token) {
      // 自动填充Type
      form.setValue('type',
        token.type === 'codex' ? ChannelType.OpenAIChat :
        ChannelType.Gemini
      )

      // 自动填充BaseURLs（如果OAuth Token有存储）
      if (token.base_urls && token.base_urls.length > 0) {
        form.setValue('base_urls', token.base_urls)
      }

      // 自动填充Model选项
      if (token.models && token.models.length > 0) {
        setModelOptions(token.models)
      }
    }
  }}
/>
```

## 实现优先级

### 阶段1：OAuth Token完整化（高优先级）
1. OAuth Token表增加`base_urls`、`models`、`provider`字段
2. OAuth登录成功后调用API获取并存储模型列表
3. GroupItem添加`type`和`oauth_token_id`字段
4. Relay实现OAuth Token直接调用

**优点：**
- ✅ OAuth完全独立，不需要Channel包装
- ✅ 模型信息一次获取，永久存储
- ✅ GroupItem可以选择Channel或OAuth
- ✅ 请求逻辑清晰分离

### 阶段2：仅自动填充（低优先级）
1. OAuth Token表只增加`models`字段
2. OAuth登录成功后调用API获取并存储
3. Channel创建/编辑表单，选择OAuth Token后自动填充模型下拉框
4. 不修改GroupItem和Relay（保持现有逻辑）

**优点：**
- ✅ 改动最小
- ✅ 不影响现有架构
- ✅ 只增强用户体验（自动填充）

## Codex API端点总结

### 当前实现使用的接口

1. **OAuth认证端点**：
   - Auth: `https://auth.openai.com/oauth/authorize`
   - Token: `https://auth.openai.com/oauth/token`
   - Redirect: `http://localhost:1455/auth/callback`

2. **聊天API端点**（需要配置在Channel中）：
   - BaseURL: 用户在Channel中配置的 `base_urls[0].url`
   - Path: `/chat/completions`
   - 完整URL示例：`https://api.openai.com/v1/chat/completions`
   - Method: POST
   - Headers: `Authorization: Bearer <access_token>`
   - Body: JSON格式请求

3. **模型列表端点**（已实现）：
   - URL: `https://api.openai.com/v1/models`
   - Method: GET
   - Headers: `Authorization: Bearer <access_token>`
   - 响应：返回可用模型ID列表
   - 参考：`internal/helper/fetch.go:52`

## 连接错误排查

### 错误信息
```
dial tcp: address missing
```

### 可能原因

1. **Base URL格式错误**
   - ❌ `api.openai.com/v1/chat/completions`  (缺少协议)
   - ❌ `https://api.openai.com/v1/chat/completions` (正确)
   - ❌ `localhost:1455` (缺少协议和路径)

2. **端口缺失**
   - ❌ `https://api.openai.com` (缺少端口和路径)

3. **代理配置未生效**
   - 系统代理配置路径：系统设置 -> Proxy URL
   - 检查点：Channel创建时是否勾选了"使用代理"

4. **网络连接问题**
   - 防火墙阻止
   - DNS解析失败

### 快速检查

**你的Channel配置是什么？**
- Type: OpenAI Chat (1)
- Base URLs: [ { url: "具体值", delay: 0 } ]
- 认证类型: OAuth (oauth_codex)
- OAuth Token: 已选择并启用

**是否使用代理？**
- 检查：Channel表单中的"代理"复选框
- 检查：系统设置中的Proxy URL配置

## 建议实现顺序

1. ✅ **修复当前连接错误**（Base URL格式、代理配置）
2. 📝 **实现自动填充功能**（OAuth登录后获取模型列表）
3. 📝 **执行方案A重构**（OAuth与Channel分离）
