# 化学侦检设备 MQTT 通信协议

> **版本：V1.2**  
> **适用范围：设备端、平台服务端、联调测试、验收测试**  
> **协议原则：已确认项直接冻结；未确认项明确标记，不自行扩展。**

---

# 1. 协议总体说明

设备与平台通过 MQTT 通信。

当前协议包含三类消息：

```text
Telemetry
设备 → 平台
用于持续上报设备实时状态

Events
设备 → 平台
用于上报报警事件、报警解除事件、指令执行结果

Notify
平台 → 设备
用于发送文本通知、动作指令、参数配置、位置共享、模式切换
```

平台和设备之间的职责边界：

> **Telemetry 表示“设备现在是什么状态”；Events 表示“设备发生了什么”；Notify 表示“平台要求设备做什么”。**

---

# 2. Topic 定义

## 2.1 上行 Topic

### Telemetry

```text
/chem/telemetry/{device_id}
```

方向：

```text
设备 → 平台
```

用途：

```text
实时位置
实时浓度
PID 阈值
当前识别物
电量
RSSI
设备模式
跌倒状态
```

### Events

```text
/chem/events/{device_id}
```

方向：

```text
设备 → 平台
```

用途：

```text
检出物质报警
检出物质报警解除
跌倒报警
跌倒报警解除
指令执行结果
```

---

## 2.2 下行 Topic

### 全设备 Notify

```text
/chem/notify
```

方向：

```text
平台 → 所有设备
```

所有设备监听。

### 单设备 Notify

```text
/chem/{device_id}/notify
```

示例：

```text
/chem/864793080139046/notify
```

方向：

```text
平台 → 指定设备
```

---

# 3. MQTT QoS 与 Retain

## 3.1 QoS

| 消息 | QoS |
|---|---:|
| Telemetry | `0` |
| Events type=0 报警开始 / 持续更新 | `1` |
| Events type=1 报警结束 | `1` |
| Events type=2 指令执行结果 | `1` |
| Notify type=0 文本通知 | `1` |
| Notify type=1 动作指令 | `1` |
| Notify type=2 参数配置 | `1` |
| Notify type=3 位置共享 | `0` |
| Notify type=4 模式切换 | `1` |

原则：

```text
高频、下一条会覆盖上一条
→ QoS 0

不能轻易丢失的业务消息
→ QoS 1
```

## 3.2 Retain

当前协议所有消息：

```text
retain = false
```

包括：

```text
Telemetry
Events
Notify
位置共享
```

原因：

- 不允许设备重连后收到历史动作指令；
- 不允许旧报警事件被平台重新订阅后误认为新报警；
- 不允许旧模式切换或旧参数配置再次执行。

---

# 4. 时间戳

所有协议中的：

```text
timestamp
```

类型：

```text
Number
```

单位：

```text
Unix 时间戳，秒
```

例如：

```json
{
  "timestamp": 1789720300
}
```

---

# 5. Telemetry 协议

Topic：

```text
/chem/telemetry/{device_id}
```

Telemetry 不使用 `type` 字段。

完整结构：

```json
{
  "timestamp": 1789720300,
  "message": {
    "gnss": {
      "fixed": false,
      "lat": 0,
      "lng": 0,
      "speed": 0
    },
    "mode": "monitor",
    "sensor": {
      "pid": {
        "conc": 0,
        "threshold": 5.0,
        "alarm": false
      },
      "battery": {
        "pct": 95,
        "voltage": 8.3791714
      },
      "alarm": {
        "names": []
      }
    },
    "rssi": -57,
    "device_id": "864793080139046",
    "timestamp": 1789720300,
    "gsensor": {
      "fall_detected": false,
      "magnitude": 1.5760642
    }
  }
}
```

已冻结规则：

```text
1. Telemetry 没有 type。
2. 外层 timestamp 保留。
3. message.timestamp 保留。
4. Topic 中 device_id 保留。
5. message.device_id 保留。
6. sensor.alarm.level 删除。
7. PID 增加 threshold。
```

---

# 6. Telemetry 字段定义

## 6.1 外层

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `timestamp` | Number | 是 | MQTT 消息时间，Unix 秒时间戳 |
| `message` | Object | 是 | 设备实时遥测数据 |

## 6.2 基础字段

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `message.device_id` | String | 是 | 设备唯一标识 |
| `message.timestamp` | Number | 是 | 设备遥测时间，Unix 秒时间戳 |
| `message.mode` | String | 是 | 当前设备工作模式 |
| `message.rssi` | Number | 是 | 当前通信信号值 |

### mode 枚举

| 值 | 含义 |
|---|---|
| `monitor` | 正常监测模式 |
| `training` | 培训 / 训练模式 |

---

## 6.3 GNSS

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `message.gnss.fixed` | Boolean | 是 | 是否定位成功 |
| `message.gnss.lat` | Number | 是 | 纬度 |
| `message.gnss.lng` | Number | 是 | 经度 |
| `message.gnss.speed` | Number | 是 | 当前速度 |

---

## 6.4 PID

| 字段 | 类型 | 必填 | 单位 | 说明 |
|---|---|---:|---|---|
| `message.sensor.pid.conc` | Number | 是 | ppm | 当前实时浓度 |
| `message.sensor.pid.threshold` | Number | 是 | ppm | 当前 PID 阈值 |
| `message.sensor.pid.alarm` | Boolean | 是 | - | 设备内部 PID 状态 |

重要规则：

> `Telemetry.sensor.pid.conc` 只用于实时设备状态展示，不作为报警记录当前浓度和最高浓度的数据来源。

报警浓度统一使用：

```text
Events type=0
message.conc
```

---

## 6.5 电池

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `message.sensor.battery.pct` | Number | 是 | 电池百分比 |
| `message.sensor.battery.voltage` | Number | 是 | 电池电压 |

---

## 6.6 当前识别物质

```text
message.sensor.alarm.names
```

类型：

```text
Array<String>
```

表示设备当前识别出的物质列表。

例如：

```json
{
  "names": ["DMMP"]
}
```

空数组：

```json
{
  "names": []
}
```

表示当前没有识别物。

Telemetry 中该字段仅用于实时状态展示。

报警记录的创建、持续和解除由 Events 决定。

---

## 6.7 跌倒状态

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `message.gsensor.fall_detected` | Boolean | 是 | 当前是否检测到跌倒 |
| `message.gsensor.magnitude` | Number | 是 | 当前运动 / 加速度幅值 |

Telemetry 中该字段用于实时状态展示。

跌倒报警开始通过 Events `type=0` 上报，正常结束通过 Events `type=1` 上报。

---

# 7. Events 协议

Topic：

```text
/chem/events/{device_id}
```

统一结构：

```json
{
  "type": 0,
  "timestamp": 1789720300,
  "message": {}
}
```

当前冻结事件类型：

| type | 事件 |
|---:|---|
| `0` | 报警开始 / 报警持续更新 |
| `1` | 报警结束 |
| `2` | 指令执行结果 |

Events 不使用 `status` 字段。

报警类型不再由 `type` 区分，而由 `message` 中携带的数据区分：

```text
names / conc
→ 检出物质报警信息

fall_detected / magnitude
→ 跌倒报警信息
```

同一设备同一时刻只维护一个当前报警过程。物质报警和跌倒状态可以存在于同一次报警过程中。

---

# 8. Events type=0：报警开始 / 报警持续更新

`type=0` 表示设备当前发生报警，或者对当前报警过程进行持续更新。

平台收到同一设备的第一条 `type=0` 时：

```text
当前没有活动报警
      ↓
创建新的报警记录
```

如果当前已经存在活动报警：

```text
收到 type=0
      ↓
更新当前报警
      ↓
不重复创建
```

## 8.1 检出物质报警开始

```json
{
  "type": 0,
  "timestamp": 1789720300,
  "message": {
    "names": ["DMMP"],
    "conc": 3.8
  }
}
```

字段：

| 字段 | 类型 | 单位 | 说明 |
|---|---|---|---|
| `message.names` | Array<String> | - | 当前报警物 |
| `message.conc` | Number | ppm | 当前报警浓度 |

平台处理：

```text
报警开始时间 = Event.timestamp
报警物 = names
当前报警浓度 = conc
```

## 8.2 检出物质报警持续更新

物质报警期间，设备可以持续发送 `type=0`：

```json
{
  "type": 0,
  "timestamp": 1789720302,
  "message": {
    "names": ["DMMP"],
    "conc": 4.6
  }
}
```

平台报警进行中实时维护：

```text
当前浓度 = 最新 Event.conc
临时最高浓度 = 当前已收到 Event.conc 的最大值
```

例如：

```text
3.8 → 4.6 → 5.3 → 4.1
```

当前实时展示：

```text
当前浓度：4.1 ppm
临时最高浓度：5.3 ppm
```

正常报警结束后，最终最高浓度不以平台临时统计值为权威，而以设备 `type=1.message.max_conc` 为准。

## 8.3 跌倒报警开始

跌倒同样使用 `type=0`：

```json
{
  "type": 0,
  "timestamp": 1789720320,
  "message": {
    "fall_detected": true,
    "magnitude": 1.8
  }
}
```

平台处理：

```text
当前没有活动报警
→ 创建报警

当前已经存在活动报警
→ 将跌倒状态合并到当前报警过程
```

## 8.4 物质报警与跌倒同时存在

例如设备先检出 DMMP，随后发生跌倒。

平台只维护一个活动报警：

```text
设备03 报警
报警物：DMMP
跌倒：是
当前浓度：4.6 ppm
```

不因为跌倒再建立第二个独立报警生命周期。

---

# 9. Events type=1：报警结束

`type=1` 表示设备明确结束当前报警过程。

正常报警结束必须由设备主动发送 `type=1`。

平台不能使用：

```text
一段时间没有继续收到报警 Event
```

作为正常报警解除依据。

## 9.1 检出物质报警结束

```json
{
  "type": 1,
  "timestamp": 1789720400,
  "message": {
    "names": ["DMMP"],
    "duration": 100,
    "max_conc": 5.3
  }
}
```

字段：

| 字段 | 类型 | 单位 | 说明 |
|---|---|---|---|
| `message.names` | Array<String> | - | 本次报警识别出的物质 |
| `message.duration` | Number | 秒 | 本次报警最终持续时间 |
| `message.max_conc` | Number | ppm | 本次报警最终最高浓度 |

正常结束后：

```text
解除时间 = Event.timestamp
最终持续时间 = message.duration
最终最高浓度 = message.max_conc
```

设备提供的 `duration` 和 `max_conc` 是正常结束情况下的最终权威值。

## 9.2 跌倒报警结束

```json
{
  "type": 1,
  "timestamp": 1789720400,
  "message": {
    "fall_detected": false,
    "duration": 90
  }
}
```

跌倒报警不需要 `max_conc`。

平台记录：

```text
报警结束时间 = Event.timestamp
最终持续时间 = duration
```

## 9.3 物质报警与跌倒合并结束

如果本次报警过程中既存在报警物，又发生过跌倒，可以在同一个结束事件中返回：

```json
{
  "type": 1,
  "timestamp": 1789720400,
  "message": {
    "names": ["DMMP"],
    "fall_detected": false,
    "duration": 100,
    "max_conc": 5.3
  }
}
```

平台收到后结束当前设备的整个报警过程。

## 9.4 持续时间权威规则

报警进行过程中，平台为了实时展示可以临时计算：

```text
当前时间 - 报警开始时间
```

正常报警结束后：

```text
最终持续时间 = type=1.message.duration
```

因此：

```text
报警进行中 → 平台临时计时
正常结束后 → 设备 duration 为最终结果
```

## 9.5 设备断联导致报警结束

设备正常情况下每 2 秒固定上报一次 Telemetry。

平台规则：

```text
15 秒没有收到 Telemetry
→ 设备离线
```

如果设备离线时仍存在活动报警：

```text
设备报警中
      ↓
15 秒无 Telemetry
      ↓
设备离线
      ↓
平台结束当前报警
```

这属于异常终止，不属于设备正常解除。

平台内部结束原因：

```text
normal  → 收到设备 type=1 正常结束
offline → 设备离线，平台结束当前报警
```

设备离线时：

```text
报警结束时间 = 最后一条有效 Telemetry 时间
```

持续时间：

```text
最后一条有效 Telemetry 时间 - 报警开始时间
```

离线判定等待的 15 秒不计入报警持续时间。

如果是物质报警，且设备没有来得及发送最终 `max_conc`：

```text
最高浓度 = 平台在报警期间已经收到的 Event.conc 最大值
```

该值表示截至设备断联前平台已收到的最高浓度。

---

# 10. Events type=2：指令执行结果

平台下发需要确认执行结果的 Notify 后，设备通过 Events `type=2` 返回结果。

成功：

```json
{
  "type": 2,
  "timestamp": 1789720310,
  "message": {
    "id": "0195f0d0-0000-7000-8000-000000000001",
    "result": "success"
  }
}
```

失败：

```json
{
  "type": 2,
  "timestamp": 1789720310,
  "message": {
    "id": "0195f0d0-0000-7000-8000-000000000001",
    "result": "failed",
    "reason": "invalid parameter"
  }
}
```

字段：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `message.id` | String | 是 | 原 Notify 的 command ID |
| `message.result` | String | 是 | 执行结果 |
| `message.reason` | String | 否 | 执行失败原因 |

`result` 固定为：

```text
success
failed
```

`timeout` 不由设备返回。

平台在规定时间内没有收到对应 ACK，则自行判定：

```text
timeout
```

---

# 11. Command ID

需要设备 ACK 的 Notify 必须携带：

```text
id
```

推荐位置：

```json
{
  "id": "0195f0d0-0000-7000-8000-000000000001",
  "type": 1,
  "timestamp": 1789720300,
  "message": {
    "action": "evacuate"
  }
}
```

## 11.1 生成规则

Command ID：

```text
由平台生成 UUID v7
```

要求：

```text
全局唯一
同一次业务指令只生成一个 ID
设备 ACK 原样返回该 ID
```

## 11.2 单设备命令

```text
平台生成 command_id=A
      ↓
发送给设备01
      ↓
设备01 ACK id=A
```

## 11.3 小组命令

一次小组操作只生成一个 command ID。

例如侦察一组 10 台设备：

```text
command_id=A

设备01 → id=A
设备02 → id=A
...
设备10 → id=A
```

平台以：

```text
command_id + device_id
```

区分每台设备的执行结果。

## 11.4 全体命令

全体广播同样只生成一个 command ID。

所有目标设备 ACK 都返回相同 command ID。

---

# 12. 指令超时

V1 默认：

```text
10 秒
```

流程：

```text
平台发布 Notify
      ↓
开始等待 ACK
      ↓
10 秒内收到 Events type=2
      ├── result=success → 执行成功
      └── result=failed  → 执行失败

10 秒没有收到 ACK
      ↓
执行超时
```

V1 不自动重试。

超时后由用户决定是否重新发送。

---

# 13. 指令幂等

由于 QoS 1 可能产生重复消息，设备必须按 command ID 做幂等。

规则：

```text
第一次收到 id=A
   ↓
执行指令
   ↓
保存执行结果
   ↓
返回 ACK

再次收到 id=A
   ↓
不重复执行
   ↓
直接返回之前保存的 ACK 结果
```

V1 建议设备缓存已执行 command ID：

```text
5 分钟
```

---

# 14. Notify 协议

全设备：

```text
/chem/notify
```

单设备：

```text
/chem/{device_id}/notify
```

当前 Notify 类型：

| type | 名称 | message 类型 | 是否必须 ID | 是否 ACK |
|---:|---|---|---|---|
| `0` | 文本通知 | String | 是 | 是 |
| `1` | 动作指令 | Object | 是 | 是 |
| `2` | 参数配置 | Object | 是 | 是 |
| `3` | 位置共享 | Object | 否 | 否 |
| `4` | 模式切换 | Object | 是 | 是 |

---

# 15. Notify type=0：文本通知

```json
{
  "id": "0195f0d0-0000-7000-8000-000000000001",
  "type": 0,
  "timestamp": 1789720300,
  "message": "请立即返回安全区域"
}
```

设备处理后返回 Events type=2。

---

# 16. Notify type=1：动作指令

结构：

```json
{
  "id": "0195f0d0-0000-7000-8000-000000000001",
  "type": 1,
  "timestamp": 1789720300,
  "message": {
    "action": "evacuate"
  }
}
```

当前动作：

| action | 含义 |
|---|---|
| `evacuate` | 立即撤离 |
| `assemble` | 集合 |
| `silent_on` | 开启静默模式 |
| `silent_off` | 解除静默模式 |
| `goto` | 前往指定坐标 |

前往坐标：

```json
{
  "id": "0195f0d0-0000-7000-8000-000000000001",
  "type": 1,
  "timestamp": 1789720300,
  "message": {
    "action": "goto",
    "lat": 22.5431,
    "lng": 114.0579
  }
}
```

---

# 17. Notify type=2：参数配置

结构保留：

```json
{
  "id": "0195f0d0-0000-7000-8000-000000000001",
  "type": 2,
  "timestamp": 1789720300,
  "message": {}
}
```

当前只冻结：

```text
type=2 表示参数配置
```

允许修改的完整参数列表和字段结构：

> **暂不定义。**

后续由设备端与平台端共同确认后扩展。

---

# 18. Notify type=3：位置共享

用途：

> 平台将同组设备最新位置整理后下发给设备。

结构：

```json
{
  "type": 3,
  "timestamp": 1789720300,
  "message": {
    "devices": [
      {
        "device_id": "864793080139046",
        "lat": 22.5431,
        "lng": 114.0579
      },
      {
        "device_id": "864793080139047",
        "lat": 22.5435,
        "lng": 114.0583
      }
    ]
  }
}
```

单次：

```text
devices.length <= 16
```

位置共享：

```text
不需要 command ID
不需要 ACK
QoS 0
retain=false
```

---

# 19. Notify type=4：模式切换

设备模式：

```text
monitor
training
```

进入训练模式：

```json
{
  "id": "0195f0d0-0000-7000-8000-000000000001",
  "type": 4,
  "timestamp": 1789720300,
  "message": {
    "mode": "training"
  }
}
```

恢复正常监测：

```json
{
  "id": "0195f0d0-0000-7000-8000-000000000002",
  "type": 4,
  "timestamp": 1789720400,
  "message": {
    "mode": "monitor"
  }
}
```

设备执行后返回 Events type=2。

后续 Telemetry 中：

```text
message.mode
```

必须反映当前实际模式。

---

# 20. 设备在线 / 离线判定

设备正常情况下每 2 秒固定上报一次 Telemetry。

V1 默认：

```text
15 秒未收到 Telemetry
→ 设备离线
```

重新上线：

```text
收到任意有效 Telemetry
→ 立即恢复在线
```

规则：

```text
上线立即生效
离线等待 15 秒确认
```

---

# 21. 编队与容量约束

业务要求：

```text
编队总数 ≥ 39
每编队设备上限 ≥ 10
```

平台最低设计容量：

```text
39 × 10 = 390 台设备
```

V1 最低目标：

> **至少支持 390 台设备同时在线。**

位置共享协议单次设备数量：

```text
最多 16 台
```

即：

```text
业务要求每队 ≥10
协议能力每次最多 16
```

留有扩展余量。

---

# 22. 协议闭环

## 实时状态

```text
Telemetry
 ↓
平台实时设备状态
```

## 报警

```text
Events type=0
 ↓
报警开始 / 报警持续更新
 ↓
物质报警期间可持续发送 Event.conc
 ↓
平台实时显示当前浓度和临时最高浓度
 ↓
Events type=1
 ↓
设备返回最终 duration
 ↓
物质报警同时返回最终 max_conc
 ↓
报警正常结束
```

设备断联：

```text
15 秒无 Telemetry
 ↓
设备离线
 ↓
平台结束当前报警
 ↓
结束时间取最后一条有效 Telemetry 时间
```

## 指令

```text
Notify + command_id
 ↓
设备执行
 ↓
Events type=2
 ↓
平台得到执行结果
```

## 训练

```text
Notify type=4 training
 ↓
Telemetry.mode=training

训练结束

Notify type=4 monitor
 ↓
Telemetry.mode=monitor
```

---

# 23. V1.2 已冻结规则

```text
Topic:
- /chem/telemetry/{device_id}
- /chem/events/{device_id}
- /chem/notify
- /chem/{device_id}/notify

Telemetry:
- 无 type
- timestamp + message
- 正常情况下每 2 秒上报一次
- PID: conc / threshold / alarm
- conc 和 threshold 单位均为 ppm
- mode: monitor / training

Events:
- type=0 报警开始 / 报警持续更新
- type=1 报警结束
- type=2 指令执行结果
- 不使用 status

报警:
- 同一设备同一时刻只维护一个活动报警
- 物质报警和跌倒状态可以合并在同一次报警过程中
- 物质报警 type=0 携带 names / conc
- 跌倒报警 type=0 携带 fall_detected / magnitude
- 物质报警期间可持续发送 Event.conc
- 平台报警进行中可临时计时和统计临时最高浓度
- 正常结束必须由设备发送 type=1
- 正常结束最终 duration 以设备 type=1.duration 为准
- 物质报警正常结束最终 max_conc 以设备 type=1.max_conc 为准
- 设备离线时平台直接结束当前报警
- 离线结束时间取最后一条有效 Telemetry 时间
- 离线持续时间由平台计算
- 离线物质报警最高浓度取已收到 Event.conc 最大值

Notify:
- type=0 文本通知
- type=1 动作指令
- type=2 参数配置
- type=3 位置共享
- type=4 模式切换

Command:
- UUID v7
- 文本/动作/配置/模式必须带 ID
- 位置共享不带 ID
- ACK result = success / failed
- timeout = 平台判定
- 超时时间 10 秒
- V1 不自动重试
- command ID 幂等
- 已执行 ID 建议缓存 5 分钟

MQTT:
- Telemetry QoS 0
- Events QoS 1
- 普通 Notify QoS 1
- 位置共享 QoS 0
- retain 全部 false

Online:
- 15 秒无 Telemetry = 离线
- 收到 Telemetry = 立即上线

Capacity:
- 编队 ≥39
- 每队 ≥10
- 同时在线最低设计规模 390
- 单次位置共享最多 16 台
```

---

# 24. 暂未定义项

以下内容继续保留，不在本版本自行扩展：

1. Notify type=2 参数配置的完整字段列表；
2. 不同 `failed` 场景的 reason 标准枚举；
3. 未来新增动作指令；
4. 未来新增事件类型；
5. 超过 390 台后的性能等级与容量分级。
