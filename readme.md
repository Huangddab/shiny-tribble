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