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

## MQTT 模拟设备

启动 HTTP 服务后，可用模拟设备验证 Telemetry、报警、Notify ACK 和超时流程：

```powershell
go run . simulator --simulator.devices=sim-001,sim-002 --simulator.alarm_every=60s --simulator.alarm_duration=20s --simulator.pollutant=DMMP --simulator.conc=3.8
```

默认每 2 秒上报一次 Telemetry。使用 `--simulator.count=390` 可生成 390 台模拟设备；使用 `--simulator.devices` 时以逗号分隔指定设备 ID。模拟器订阅 `/chem/notify` 和 `/chem/{device_id}/notify`，位置共享不回复 ACK，其余 Notify 会返回 Events type=2。