mqtt订阅单例

err := mqtt.Subscribe("device/+/event", 1,
    func(client paho.Client, message paho.Message) {
        logrus.Infof(
            "topic=%s payload=%s",
            message.Topic(),
            message.Payload(),
        )
    })

if err != nil {
    return err
}

## 登录接口

使用 `user init` 初始化默认管理员后：

```http
POST /api/auth/login
Content-Type: application/json

{"username":"admin","password":"admin"}
```

登录成功返回 JWT，登出接口需要携带 `Authorization: Bearer <token>`：

```http
POST /api/auth/logout
Authorization: Bearer <token>
```

登录成功、失败、数据库不可用和登出都会写入 `dashboard_audit_logs`。

