# 化学侦检设备 MQTT 通信协议 V1.4

## 1. 协议概述

本协议定义化学侦检设备与数据平台之间基于 MQTT 的通信规范，包括：

- **Telemetry（遥测）**：设备实时状态上报。
- **Events（事件）**：设备报警、报警结束及指令执行结果上报。
- **Notify（通知）**：平台向设备下发通知、参数配置、位置共享和模式切换指令。

### 1.1 MQTT Topic 定义

| 通信方向    | 消息类别         | Topic                        |
| ----------- | ---------------- | ---------------------------- |
| 设备 → 平台 | Telemetry        | `chem/telemetry/{device_id}` |
| 设备 → 平台 | Events           | `chem/events/{device_id}`    |
| 平台 → 设备 | Notify（广播）   | `chem/notify`                |
| 平台 → 设备 | Notify（单设备） | `chem/{device_id}/notify`    |

---

## 2. 消息分类

### 2.1 Telemetry — 遥测消息

**通信方向：** 设备 → 平台

**用途：** 持续上报设备的实时运行状态。

通信规则：

- 不使用 `type` 字段。
- 消息由 `timestamp` 和 `message` 组成。
- MQTT QoS 为 `0`。
- Telemetry 只表示设备当前状态，**不创建报警**。

### 2.2 Events — 事件消息

**通信方向：** 设备 → 平台

**用途：**

- 报警开始。
- 报警持续更新。
- 报警结束。
- 指令执行结果反馈。

### 2.3 Notify — 下行通知

**通信方向：** 平台 → 设备

**用途：**

- 文本通知。
- 参数配置。
- 位置共享。
- 模式切换。

---

## 3. MQTT QoS 规范

| 消息类别  | Type | 消息说明            | QoS |
| --------- | ---- | ------------------- | --- |
| Telemetry | —    | 实时遥测            | 0   |
| Events    | 0    | 报警开始 / 持续更新 | 1   |
| Events    | 1    | 报警结束            | 1   |
| Events    | 2    | 指令执行结果        | 1   |
| Notify    | 0    | 文本通知            | 1   |
| Notify    | 1    | 参数配置            | 1   |
| Notify    | 2    | 位置共享            | 0   |
| Notify    | 3    | 模式切换            | 1   |

**Retain 规则**

所有 MQTT 消息统一设置：

```text
retain=false
```

---

## 4. 时间戳规范

所有 `timestamp` 字段统一使用：

**Unix 时间戳，单位为秒。**

---

## 5. Telemetry — 遥测协议

### 5.1 通信定义

| 属性     | 定义                         |
| -------- | ---------------------------- |
| 通信方向 | 设备 → 平台                  |
| Topic    | `chem/telemetry/{device_id}` |
| QoS      | 0                            |
| Retain   | false                        |
| Type     | 无                           |

### 5.2 消息结构

```json
{
  "timestamp": 1789720300,
  "message": {
    "device_id": "864793080139046",
    "mode": "monitor",
    "sensor": {
      "pid": {
        "conc": 0,
        "threshold": 5,
        "alarm": false
      },
      "battery": {
        "pct": 95,
        "voltage": 8.3
      },
      "alarm": {
        "names": []
      }
    },
    "gnss": {
      "fixed": true,
      "lat": 0,
      "lng": 0,
      "speed": 0
    },
    "rssi": -57,
    "gsensor": {
      "fall_detected": false,
      "magnitude": 1.5
    }
  }
}
```

### 5.3 主要字段说明

| 字段路径               | 说明                   |
| ---------------------- | ---------------------- |
| `sensor.pid.conc`      | PID 实时浓度，单位 ppm |
| `sensor.pid.threshold` | PID 报警阈值，单位 ppm |
| `sensor.alarm.names`   | 当前识别物质           |
| `mode`                 | 设备当前工作模式       |

### 5.4 工作模式定义

| mode       | 含义         |
| ---------- | ------------ |
| `monitor`  | 正常监测模式 |
| `training` | 演练模式     |

**注意：** Telemetry 仅表示设备当前运行状态，不作为创建报警事件的依据。

---

## 6. Events — 事件协议

### 6.1 通信定义

| 属性     | 定义                      |
| -------- | ------------------------- |
| 通信方向 | 设备 → 平台               |
| Topic    | `chem/events/{device_id}` |
| QoS      | 1                         |
| Retain   | false                     |

### 6.2 通用消息结构

```json
{
  "type": 0,
  "timestamp": 1789720300,
  "message": {}
}
```

### 6.3 事件类型定义

| type | 事件类型 | 说明                |
| ---- | -------- | ------------------- |
| `0`  | 报警     | 报警开始 / 持续更新 |
| `1`  | 报警结束 | 结束当前报警        |
| `2`  | ACK      | 指令执行结果        |

**规则：** Events 不使用 `status` 字段。

---

## 7. Events type=0 — 报警事件

`type=0` 用于上报报警开始以及报警期间的持续更新。

### 7.1 物质报警

**消息示例**

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

**字段说明**

| 字段    | 类型   | 说明               |
| ------- | ------ | ------------------ |
| `names` | Array  | 报警物质           |
| `conc`  | Number | 报警浓度，单位 ppm |

**持续报警规则**

报警持续期间，设备继续发送 `type=0` 消息，更新 `conc` 浓度值。

### 7.2 跌倒报警

**消息示例**

```json
{
  "type": 0,
  "timestamp": 1789720300,
  "message": {
    "fall_detected": true,
    "magnitude": 1.8
  }
}
```

---

## 8. Events type=1 — 报警结束

`type=1` 用于通知平台当前报警结束。

### 8.1 物质报警结束

**消息示例**

```json
{
  "type": 1,
  "timestamp": 1789720400,
  "message": {
    "names": ["DMMP"],
    "duration": 100,
    "peak_conc": 56
  }
}
```

**字段说明**

| 字段        | 类型   | 说明                       |
| ----------- | ------ | -------------------------- |
| `names`     | Array  | 报警物质                   |
| `duration`  | Number | 报警持续时间，单位秒       |
| `peak_conc` | Number | 报警期间最高浓度，单位 ppm |

### 8.2 跌倒报警结束

**消息示例**

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

---

## 9. Events type=2 — 指令执行结果（ACK）

设备执行平台下发的指令后，通过 Events `type=2` 返回执行结果。

### 9.1 消息结构

```json
{
  "type": 2,
  "timestamp": 1789720400,
  "message": {
    "id": "uuid-v7",
    "result": "success"
  }
}
```

### 9.2 字段说明

| 字段     | 说明               |
| -------- | ------------------ |
| `id`     | 指令 ID（UUID v7） |
| `result` | 指令执行结果       |

### 9.3 执行结果定义

| result    | 含义     |
| --------- | -------- |
| `success` | 执行成功 |
| `failed`  | 执行失败 |

---

## 10. Notify — 平台下行协议

### 10.1 通信定义

**通信方向：** 平台 → 设备

支持以下两种 Topic。

| 发送方式   | Topic                     |
| ---------- | ------------------------- |
| 全设备广播 | `chem/notify`             |
| 单设备下发 | `chem/{device_id}/notify` |

**Notify 消息类型**

| type | 名称     | QoS |
| ---- | -------- | --- |
| `0`  | 文本通知 | 1   |
| `1`  | 参数配置 | 1   |
| `2`  | 位置共享 | 0   |
| `3`  | 模式切换 | 1   |

### 10.2 Notify type=0 — 文本通知

**消息示例**

```json
{
  "type": 0,
  "timestamp": 1789720300,
  "message": "请注意"
}
```

### 10.3 Notify type=1 — 参数配置

**通用消息结构**

```json
{
  "id": "uuid-v7",
  "type": 1,
  "timestamp": 1789720300,
  "message": {}
}
```

**说明：**

当前仅定义通信协议，不定义其他具体配置参数。后续可扩展预置位等配置功能。

#### 10.3.1 action=pollution_source — 模拟污染源配置

**触发时机**

开启训练模式时，平台向同一编队额外下发模拟污染源配置指令。

指令中包含训练页面填写的模拟污染源参数。

**消息示例**

```json
{
  "id": "uuid-v7",
  "type": 1,
  "timestamp": 1789720300,
  "message": {
    "action": "pollution_source",
    "substance": "DMMP",
    "conc": 5
  }
}
```

**字段说明**

| 字段        | 说明               |
| ----------- | ------------------ |
| `action`    | `pollution_source` |
| `substance` | 模拟物质           |
| `conc`      | 模拟浓度，单位 ppm |

**执行规则**

- 设备执行后通过 Events `type=2` 返回 ACK。
- 本指令与 Notify `type=3` 模式切换指令相互独立下发。
- 本指令的 ACK 结果不影响训练任务是否进入 `active` 状态。

#### 10.3.2 action=pollution_source_clear — 清除模拟污染源

**触发时机**

结束训练时，平台在下发 Notify `type=3` 恢复正常监测模式的同时，向同一编队下发本指令，通知设备停止模拟污染源。

**消息示例**

```json
{
  "id": "uuid-v7",
  "type": 1,
  "timestamp": 1789720400,
  "message": {
    "action": "pollution_source_clear"
  }
}
```

**执行规则**

- 只有该训练配置了模拟污染源时，才下发此指令。
- 设备执行后通过 Events `type=2` 返回 ACK。

### 10.4 Notify type=2 — 位置共享

**功能说明**

平台只向设备发送同组其他设备的位置。

**示例场景**

设备 A 和设备 B 属于同一编组。

平台发送给设备 A 的位置共享消息：

```json
{
  "type": 2,
  "message": {
    "devices": [
      {
        "device_id": "B",
        "lat": 22.5,
        "lng": 113.9
      }
    ]
  }
}
```

**位置共享规则**

| 规则     | 说明           |
| -------- | -------------- |
| 设备范围 | 仅发送同组设备 |
| 自身设备 | 不包含自身     |
| ACK      | 不需要         |
| 单次数量 | 最多 16 台     |
| QoS      | 0              |

### 10.5 Notify type=3 — 模式切换

**功能说明**

平台向设备下发工作模式切换指令。

支持正常监测模式与演练模式之间的切换。

#### 10.5.1 进入演练模式

```json
{
  "id": "uuid-v7",
  "type": 3,
  "message": {
    "mode": "training"
  }
}
```

#### 10.5.2 恢复正常监测模式

```json
{
  "id": "uuid-v7",
  "type": 3,
  "message": {
    "mode": "monitor"
  }
}
```

**模式定义**

| mode       | 含义         |
| ---------- | ------------ |
| `training` | 演练模式     |
| `monitor`  | 正常监测模式 |

**执行规则：** 设备执行模式切换后，通过 Events `type=2` 返回 ACK。

---

## 11. 报警生命周期

### 11.1 报警开始

设备检测到报警后，发送：

```text
Events type=0
```

### 11.2 报警持续更新

报警持续期间，设备继续发送：

```text
Events type=0
```

更新报警浓度等相关数据。

### 11.3 报警正常结束

设备结束报警时，发送：

```text
Events type=1
```

物质报警正常结束时，设备提供：

| 字段        | 说明             |
| ----------- | ---------------- |
| `duration`  | 报警持续时间     |
| `peak_conc` | 报警期间最高浓度 |

### 11.4 设备离线处理

平台检测设备 Telemetry 上报情况。

当设备连续 **15 秒**未上报 Telemetry 时，平台结束报警。

### 11.5 报警生命周期流程

```text
设备触发报警
      │
      ▼
Events type=0
  报警开始
      │
      ▼
报警持续期间
Events type=0
  更新报警数据
      │
      ▼
  报警结束判断
      │
      ├── 设备正常结束
      │       │
      │       ▼
      │   Events type=1
      │   携带结束数据
      │       │
      │       ▼
      │   平台结束报警
      │
      └── 设备离线
              │
              ▼
       15秒无Telemetry
              │
              ▼
         平台结束报警
```
