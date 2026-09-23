# 化学侦检设备 MQTT 通信协议 V1.4

## 1. MQTT Topic

### 上行

Telemetry:

``` text
chem/telemetry/{device_id}
```

Events:

``` text
chem/events/{device_id}
```

### 下行 Notify

全设备：

``` text
chem/notify
```

单设备：

``` text
chem/{device_id}/notify
```

------------------------------------------------------------------------

# 2. 消息分类

## Telemetry

设备 → 平台

用于持续上报实时状态。

规则：

-   不使用 type 字段
-   使用 timestamp + message
-   QoS 0

## Events

设备 → 平台

用于：

-   报警开始
-   报警持续更新
-   报警结束
-   指令执行结果

## Notify

平台 → 设备

用于：

-   文本通知
-   参数配置
-   位置共享
-   模式切换

------------------------------------------------------------------------

# 3. QoS

  消息                              QoS
  --------------------------------- -----
  Telemetry                         0
  Events type=0 报警开始/持续更新   1
  Events type=1 报警结束            1
  Events type=2 执行结果            1
  Notify type=0 文本通知            1
  Notify type=1 参数配置            1
  Notify type=2 位置共享            0
  Notify type=3 模式切换            1

所有消息：

``` text
retain=false
```

------------------------------------------------------------------------

# 4. 时间戳

所有 timestamp：

``` text
Unix 时间戳（秒）
```

------------------------------------------------------------------------

# 5. Telemetry 协议

Topic:

``` text
chem/telemetry/{device_id}
```

结构：

``` json
{
    "timestamp":1789720300,
    "message":{
        "device_id":"864793080139046",
        "mode":"monitor",
        "sensor":{
            "pid":{
                "conc":0,
                "threshold":5,
                "alarm":false
            },
            "battery":{
                "pct":95,
                "voltage":8.3
            },
            "alarm":{
                "names":[]
            }
        },
        "gnss":{
            "fixed":true,
            "lat":0,
            "lng":0,
            "speed":0
        },
        "rssi":-57,
        "gsensor":{
            "fall_detected":false,
            "magnitude":1.5
        }
    }
}
```

字段：

  字段            说明
  --------------- --------------
  pid.conc        实时浓度 ppm
  pid.threshold   PID阈值 ppm
  alarm.names     当前识别物质
  mode            设备当前模式

mode:

  值         含义
  ---------- --------------
  monitor    正常监测模式
  training   演练模式

Telemetry 只表示当前状态，不创建报警。

------------------------------------------------------------------------

# 6. Events 协议

统一结构：

``` json
{
    "type":0,
    "timestamp":1789720300,
    "message":{}
}
```

type:

  type   含义
  ------ -------------------
  0      报警开始/持续更新
  1      报警结束
  2      指令执行结果

不使用 status。

------------------------------------------------------------------------

# 7. Events type=0 报警

## 物质报警

``` json
{
"type":0,
"timestamp":1789720300,
"message":{
    "names":["DMMP"],
    "conc":3.8
}
}
```

含义：

-   names：报警物
-   conc：报警浓度 ppm

持续报警：

继续发送 type=0 更新 conc。

## 跌倒报警

``` json
{
"type":0,
"timestamp":1789720300,
"message":{
    "fall_detected":true,
    "magnitude":1.8
}
}
```

------------------------------------------------------------------------

# 8. Events type=1 报警结束

## 物质报警结束

``` json
{
"type":1,
"timestamp":1789720400,
"message":{
    "names":["DMMP"],
    "duration":100,
    "peak_conc":56
}
}
```

字段：

-   duration：秒
-   peak_conc：报警期间最高浓度 ppm

## 跌倒结束

``` json
{
"type":1,
"timestamp":1789720400,
"message":{
    "fall_detected":false,
    "duration":90
}
}
```

------------------------------------------------------------------------

# 9. Events type=2 ACK

``` json
{
"type":2,
"timestamp":1789720400,
"message":{
    "id":"uuid-v7",
    "result":"success"
}
}
```

result:

``` text
success
failed
```

------------------------------------------------------------------------

# 10. Notify

Topic：

``` text
chem/notify

chem/{device_id}/notify
```

## type=0 文本通知

``` json
{
"type":0,
"timestamp":1789720300,
"message":"请注意"
}
```

------------------------------------------------------------------------

## type=1 参数配置

``` json
{
"id":"uuid-v7",
"type":1,
"timestamp":1789720300,
"message":{}
}
```

当前只定义协议，不定义具体参数。暂时不用后期可以添加预置位等设置。

### action=pollution_source 模拟污染源配置

开启训练模式时，平台会额外向同一编队下发本条指令，携带训练页面填写的模拟污染源参数：

``` json
{
"id":"uuid-v7",
"type":1,
"timestamp":1789720300,
"message":{
    "action":"pollution_source",
    "substance":"DMMP",
    "conc":5
}
}
```

字段：

-   substance：模拟物质
-   conc：模拟浓度 ppm

设备执行后返回 Events type=2 ACK。此指令与 type=3 模式切换指令相互独立下发，其 ACK 结果不影响训练任务是否进入 active 状态。

### action=pollution_source_clear 清除模拟污染源

结束训练时，平台会在下发 type=3 恢复监测模式的同时，向同一编队下发本条指令，告知设备停止模拟污染源：

``` json
{
"id":"uuid-v7",
"type":1,
"timestamp":1789720400,
"message":{
    "action":"pollution_source_clear"
}
}
```

只有该训练配置了模拟污染源时才会下发本条指令。设备执行后返回 Events type=2 ACK。

------------------------------------------------------------------------

## type=2 位置共享

平台只发送同组设备位置。

示例：

A、B 同组：

A收到：

``` json
{
"type":2,
"message":{
 "devices":[
   {
    "device_id":"B",
    "lat":22.5,
    "lng":113.9
   }
 ]
}
}
```

规则：

-   不包含自身
-   只发送同组设备
-   不需要ACK
-   单次最多16台

------------------------------------------------------------------------

## type=3 模式切换

进入演练：

``` json
{
"id":"uuid-v7",
"type":3,
"message":{
    "mode":"training"
}
}
```

恢复正常：

``` json
{
"id":"uuid-v7",
"type":3,
"message":{
    "mode":"monitor"
}
}
```

设备执行后返回 ACK。

------------------------------------------------------------------------

# 11. 报警生命周期

开始：

``` text
Events type=0
```

结束：

``` text
Events type=1
```

正常结束：

设备提供：

``` text
duration
peak_conc
```

设备离线：

``` text
15秒没有Telemetry
↓
平台结束报警
```
