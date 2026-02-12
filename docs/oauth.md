# OAuth 集成指南

Octopus 支持两种认证模式，可以共存：

1. **API Key 模式** - 传统的 API Key 认证
2. **OAuth 模式** - 通过 OAuth 登录获取 Token

## 支持的 OAuth 提供商

| 提供商 | 认证类型 | 说明 |
|--------|----------|------|
| Codex / ChatGPT | `oauth_codex` | 使用 ChatGPT 账号登录 |
| Antigravity / Gemini | `oauth_antigravity` | 使用 Google 账号登录 |

### OAuth 认证流程说明

系统支持两种认证流程：

1. **自动回调模式**（推荐）
   - 服务器监听回调端口（Codex: 1455, Antigravity: 51121）
   - 用户完成认证后，OAuth 提供商自动回调到本地服务器
   - 服务器自动处理 code 换取 token 并保存

2. **手动提交模式**（备用）
   - 用户完成认证后，复制浏览器地址栏中的回调 URL
   - 手动粘贴到系统输入框
   - 系统解析 code 参数并换取 token

**为什么需要手动提交模式？**
- 防火墙阻止本地端口回调
- 回调端口被其他程序占用
- OAuth 提供商未正确重定向到本地
- 作为备用方案确保认证成功

## 使用步骤

### 概述

Octopus 支持两种方式添加 OAuth Token：

1. **OAuth 登录** - 推荐方式，通过浏览器自动完成认证
2. **手动添加** - 适合已有 Token 的用户

### 1. 添加 OAuth Token

1. 进入 **OAuth** 页面
2. 点击右上角的 **+** 按钮
3. 选择添加方式：
   - **OAuth 登录** - 自动通过浏览器登录
   - **手动添加** - 手动输入 Token 信息

#### OAuth 登录方式

1. 选择提供商（Codex 或 Antigravity）
2. 点击 **开始登录** 打开认证窗口
3. 在弹出的浏览器窗口中完成认证
4. 完成认证后，复制浏览器地址栏中的回调 URL
5. 将回调 URL 粘贴到输入框中
6. 点击 **提交回调 URL**，系统会自动换取 Token 并保存到列表

**注意事项：**
- Codex 回调 URL 格式：`http://localhost:1455/auth/callback?code=xxx&state=xxx`
- Antigravity 回调 URL 格式：`http://localhost:51121/oauth-callback?code=xxx&state=xxx`
- 系统会自动解析 code 参数并换取 access_token

#### 手动添加方式

如果你已经有 Token，可以手动添加。这适合以下场景：
- 你已经通过其他方式获取了 Access Token
- 你想从其他系统迁移 Token
- OAuth 登录遇到问题，需要手动配置

**字段说明：**

| 字段 | 必填 | 说明 |
|------|------|------|
| Provider | 是 | codex 或 antigravity |
| Email | 否 | 关联的邮箱地址 |
| Access Token | 是 | OAuth 访问令牌 |
| Refresh Token | 否 | 刷新令牌（用于自动刷新） |
| Expires In | 否 | 过期时间（秒） |
| Remark | 否 | 备注信息 |

### 2. 创建使用 OAuth 的 Channel

1. 进入 **Channel** 页面
2. 点击 **+** 创建新 Channel
3. 填写基本信息：
   - **渠道名称** - 自定义名称
   - **渠道类型** - 选择对应的类型
   - **Base URLs** - API 端点地址
4. 选择 **认证类型**：
   - `API Key` - 传统模式
   - `OAuth (Codex/ChatGPT)` - Codex OAuth
   - `OAuth (Antigravity/Gemini)` - Antigravity OAuth
5. 如果选择了 OAuth，在 **OAuth Token** 下拉框中选择一个有效的 Token
6. 选择支持的模型
7. 点击 **创建渠道**

### 3. 管理 OAuth Token

在 OAuth 页面可以：

- **查看状态** - Active / Expired / Disabled
- **刷新 Token** - 手动刷新即将过期的 Token
- **启用/禁用** - 切换 Token 的启用状态
- **删除 Token** - 删除不再使用的 Token
- **查看详情** - 查看过期时间、上次刷新时间等

## Token 自动刷新

系统会自动处理 Token 刷新：

1. **请求时刷新** - 当 Token 即将过期（5分钟内）时，系统会自动刷新
2. **后台刷新** - 可以启用后台定时刷新（可选）

如果刷新失败，系统会跳过该 Channel，尝试其他可用的 Channel。

## 认证模式对比

| 特性 | API Key 模式 | OAuth 模式 |
|------|-------------|------------|
| 认证方式 | API Key | OAuth Token |
| Token 管理 | 手动更新 | 自动刷新 |
| 适用场景 | 企业 API Key | 个人订阅账号 |
| 安全性 | Key 可能泄露 | Token 定期刷新 |
| 多账号支持 | 多 Key 负载均衡 | 多 Token 负载均衡 |

## API 端点

### OAuth 管理

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/oauth/start/:provider` | POST | 启动 OAuth 登录，返回认证 URL |
| `/api/v1/oauth/callback/codex` | GET | Codex 回调端点（自动处理）|
| `/api/v1/oauth/callback/antigravity` | GET | Antigravity 回调端点（自动处理）|
| `/api/v1/oauth/callback/submit` | POST | 手动提交回调 URL |
| `/api/v1/oauth/token/list` | GET | 获取 Token 列表 |
| `/api/v1/oauth/token/create` | POST | 手动创建 Token |
| `/api/v1/oauth/token/update` | POST | 更新 Token |
| `/api/v1/oauth/token/delete/:id` | DELETE | 删除 Token |
| `/api/v1/oauth/token/refresh/:id` | POST | 刷新 Token |
| `/api/v1/oauth/status/:id` | GET | 获取 Token 状态 |

#### 启动 OAuth 登录

**请求：** `POST /api/v1/oauth/start/{provider}`

**响应：**
```json
{
  "auth_url": "https://auth.openai.com/oauth/authorize?...",
  "session_id": "abc123...",
  "provider": "codex",
  "code_verifier": "xxx..."  // 仅 Codex，手动提交时需要
}
```

#### 手动提交回调 URL

**请求：** `POST /api/v1/oauth/callback/submit`

**请求体：**
```json
{
  "provider": "codex",           // "codex" 或 "antigravity"
  "callback_url": "http://localhost:1455/auth/callback?code=xxx&state=xxx",
  "code_verifier": "xxx..."      // 仅 Codex 必需，Antigravity 不需要
}
```

**响应：**
```json
{
  "id": 1,
  "type": "codex",
  "email": "user@example.com",
  "enabled": true,
  "is_expired": false,
  ...
}
```

## 注意事项

1. **Token 安全**
   - Token 信息在 API 响应中会被隐藏
   - 建议定期检查 Token 状态

2. **Token 过期**
   - 过期的 Token 不会被用于请求
   - 需要刷新或重新登录

3. **Channel 关联**
   - 被 Channel 使用的 Token 无法删除
   - 需要先删除或修改关联的 Channel

4. **OAuth 回调**
   - OAuth 登录需要本地服务监听回调
   - Codex 回调端口: 1455
   - Antigravity 回调端口: 51121
   - 手动提交回调 URL 时，需要完整复制浏览器地址栏中的 URL（包含 code 和 state 参数）
   - 对于 Codex，系统会在启动登录时提供 code_verifier，提交回调时会自动使用

## 故障排除

### Token 刷新失败

1. 检查 Refresh Token 是否有效
2. 尝试重新登录获取新 Token
3. 检查网络连接

### Channel 无法使用 OAuth Token

1. 确认 Token 状态为 Active
2. 确认 Token 没有过期
3. 确认 Channel 的认证类型与 Token 类型匹配

### OAuth 登录失败

1. 检查回调端口是否被占用
   - Codex: 确保 1455 端口可用
   - Antigravity: 确保 51121 端口可用
2. 检查浏览器是否阻止了弹出窗口
3. 检查网络连接是否正常
4. 尝试使用手动添加方式

### 提交回调 URL 失败

1. **确认回调 URL 完整** - 必须包含 `code` 和 `state` 参数
2. **检查 provider 匹配** - Codex 必须使用 Codex 的回调 URL
3. **检查 URL 格式** - 确保复制的是完整的回调 URL，不是重定向后的页面
4. **查看错误信息** - 系统会返回具体的错误原因（如缺少 code_verifier）
