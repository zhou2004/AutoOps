# 账号注册、邮箱验证与密码找回

## 邮箱配置

系统优先读取 `AUTOOPS_SMTP_*` 环境变量，其次读取 `api/config.yaml` 的 `accountSMTP`，最后兼容监控中心已有的 `Email_host/Email_port/Email_user/Email_password` 配置。发件信息和授权码不会返回给登录页。

```yaml
accountSMTP:
  host: smtp.163.com
  port: '465'
  user: your-account@163.com
  password: YOUR_SMTP_AUTHORIZATION_CODE
  from: your-account@163.com
  tls: tls
```

使用邮箱服务商提供的 SMTP 授权码。465 使用 `tls`；587 使用 `starttls`。服务器证书必须有效，不支持跳过 TLS 校验。邮件发送连接超时 10 秒，总传输超时 20 秒。

Docker / K8s 对应环境变量：

```dotenv
AUTOOPS_SMTP_HOST=smtp.163.com
AUTOOPS_SMTP_PORT=465
AUTOOPS_SMTP_USER=your-account@163.com
AUTOOPS_SMTP_PASSWORD=YOUR_SMTP_AUTHORIZATION_CODE
AUTOOPS_SMTP_FROM=your-account@163.com
AUTOOPS_SMTP_TLS=tls
```

配置后重启 API。登录页会根据 `/api/v1/auth/options` 判断邮件功能是否可用，未配置时仍可正常登录。

## 注册与审批

1. 登录页选择「注册账号」，填写邮箱和图形验证码。
2. 收到邮件后填写 6 位验证码、用户名和密码。
3. 注册成功后，账号处于禁用状态且不绑定任何角色。
4. 管理员在「用户管理」中查看备注为「邮箱验证注册，等待管理员启用并分配权限」的用户，检查身份后启用并分配角色及资产范围。

设置 `AUTOOPS_REGISTRATION_ENABLED=false` 可关闭公开注册，已有用户登录与密码找回不受影响。

## 密码找回

选择「忘记密码？」并填写账号已绑定的邮箱。邮箱必须唯一对应一个账号；未绑定或重复绑定的邮箱需要管理员先处理。为避免暴露账号是否存在，发送接口统一提示「如邮箱符合条件，验证码已发送」。

完成验证码验证后，新密码采用 bcrypt 保存，旧浏览器会话立即失效。账号原来的启用/禁用状态和权限不变。API Token 是独立凭据，不受浏览器密码重置影响；需要时请在 Token 管理中撤销。

## 验证规则

- 图形验证码一次性消费，5 分钟有效。
- 邮箱验证码 10 分钟有效，仅能用于对应邮箱和用途。
- 每个验证码最多失败 5 次，成功后不可重用。
- 邮箱每分钟最多发送一次、每小时最多 5 次；来源 IP 每小时最多 20 次。
- 密码为 10–72 字节；旧 MD5 账号仍可登录，邮箱重置及个人改密后采用 bcrypt。

验证数据、限流及会话版本保存在数据库中，可用于多 API 实例。共享 JWT 签名密钥和 Redis 配置仍然必需。

## API

| 方法 | 地址 | 用途 |
|---|---|---|
| GET | `/api/v1/auth/options` | 邮件及注册可用状态 |
| POST | `/api/v1/auth/email-code` | 校验图形验证码并发送邮件验证码 |
| POST | `/api/v1/auth/register` | 验证邮箱并创建待审批账号 |
| POST | `/api/v1/auth/reset-password` | 验证邮箱并更新密码、撤销旧会话 |
