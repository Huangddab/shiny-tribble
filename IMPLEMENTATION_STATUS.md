# 平台业务实现清单

依据《化学侦检数据平台业务逻辑说明书 V1.8》和《化学侦检设备 MQTT 通信协议 V1.4》维护。

## 已实现

- [x] Docker Mosquitto Broker 与瓦片服务编排。
- [x] MQTT 客户端连接、发布、订阅、关闭。
- [x] `/map` 领导大屏页面与响应式布局。
- [x] `/api/dashboard/snapshot` 大屏快照接口。
- [x] Telemetry 订阅 `chem/telemetry/+`。
- [x] Events 订阅 `chem/events/+`。
- [x] Telemetry 自动发现设备并更新位置、模式、PID浓度/阈值/报警、电量/电压、RSSI、GNSS速度和跌倒状态/幅度。
- [x] 收到 Telemetry 后设备上线。
- [x] 15 秒无 Telemetry 后设备离线。
- [x] Events type=0 创建或更新一个设备活动报警。
- [x] 报警中的当前浓度和临时最高浓度的基础字段。
- [x] 报警正常结束保留历史，并使用设备 `duration` / `peak_conc`。
- [x] 报警离线结束记录 `offline` 原因和最后有效 Telemetry 时间。
- [x] Notify 下行：type 0/1/3 使用 QoS 1，type 2 位置共享使用 QoS 0；所有消息 `retain=false`。
- [x] Events type=2 ACK 和 10 秒 timeout 基础状态。
- [x] 设备分组接口和每组 10 台容量校验。
- [x] 设备上线后按编队定时自动位置共享，单次最多 16 台设备且不包含自身。
- [x] 报警确认接口，确认状态与报警生命周期独立。
- [x] 训练任务开始/结束 API，使用 Notify type=3 切换模式。
- [x] MQTT 自动重连后恢复已注册订阅。
- [x] 大屏展示训练任务状态和指令逐设备 ACK 结果。
- [x] HTTP 启动时可选初始化 MongoDB，数据库不可用时保留内存模式。
- [x] 报警结束历史和指令创建/ACK/超时结果异步持久化。
- [x] 报警确认状态持久化，并增加支持设备/小组/时间范围/游标分页的报警历史查询接口。
- [x] 清理旧版报警历史文档，仅保留带 Unix 时间字段的新数据格式。
- [x] 训练任务开始/结束持久化，并增加训练历史查询接口。
- [x] 指令历史查询接口，支持结果/目标过滤，并记录 ACK 结果和超时审计字段。
- [x] Notify type 0/1/2/3 字段校验，以及 Telemetry/Events 协议字段校验和错误日志。

## MQTT 协议 V1.4 对照

### 已符合

- [x] 按协议订阅 Telemetry `chem/telemetry/+`（QoS 0）和 Events `chem/events/+`（QoS 1）。
- [x] Notify 按广播 `chem/notify` 或单设备 `chem/{device_id}/notify` 下发。
- [x] Notify type 0/1/2/3 使用协议规定的 QoS；发布消息均不保留。
- [x] Notify 使用 Unix 秒级 `timestamp` 和 `type` / `message` 结构；type 1、type 3 指令携带 UUID v7 `id`。
- [x] 支持 type 1 `pollution_source` / `pollution_source_clear` 与 type 3 `training` / `monitor` 模式切换。
- [x] 支持 Events type 0 报警更新、type 1 报警结束、type 2 指令 ACK。
- [x] 保存 Telemetry 的 `gnss.speed`、`sensor.battery.voltage`、`sensor.pid.alarm`、`gsensor.magnitude`，并保存跌倒事件的 `magnitude`。
- [x] 校验物质/跌倒 Events 的必需字段，包括浓度、幅度、持续时间和峰值浓度。
- [x] 设备离线阈值为 15 秒；离线时平台结束其活动报警。
- [x] 位置共享不包含自身、仅面向同组在线设备，单次最多 16 个位置。

### 字段处理范围

- Telemetry 中的协议字段会被解码并保存在设备状态/遥测采样中；跌倒事件幅度会保存在报警记录中。
- 物质与跌倒事件按协议校验各自必需字段；不符合结构的 Events 会记录日志并丢弃。

## 部分实现

- [x] 已解除报警写入 `dashboard_alert_history` 集合。
- [x] 指令支持单设备、全体和小组；同一 command ID 下按设备记录 ACK 结果并聚合状态。
- [x] 设备名称修改、设备列表、39 个编队目录、容量统计和分组审计接口。
- [x] 遥测按设备周期采样持久化，训练模式采样带 `training` 标记并支持历史查询。
- [x] MQTT 初始化失败不阻断 HTTP；Broker 自动重连后恢复已注册订阅。

## 未实现，按优先级

### P0：业务闭环

P0 已完成。

### P1：平台基础业务

P1 已完成。
8. 位置共享：设备上线后按小组定时下发最新位置，单次最多 16 台且不包含自身。

### P2：数据与运营

- [x] 设备轨迹、遥测采样、报警历史、指令历史、训练历史。
- [x] 报警确认、历史查询、复盘接口（基础历史查询已完成）。
- [x] 审计日志：登录、分组、报警确认、指令、训练、导出。

当前没有未实现的 V1.8 业务项；390 台并发压测按要求不纳入范围。

## 当前迭代

已完成报警状态机、Notify/ACK、设备管理、39 个编队目录、位置共享、训练展示、遥测采样、设备轨迹、训练污染源参数、报警和训练任务基础持久化、报警和指令历史查询、复盘查询、历史导出审计、登录审计和协议校验。设备模拟程序位于独立的 `mock` 目录。
