var map = L.map('map-container', { minZoom: 14, maxZoom: 18, zoomControl: false }).setView([22.6335, 113.9035], 14);
var markers = L.layerGroup().addTo(map);

L.tileLayer('http://localhost:8081/data/baoan/{z}/{x}/{y}.png', { minZoom: 14, maxZoom: 18, maxNativeZoom: 18, attribution: '' }).addTo(map);

function setText(id, value) { document.getElementById(id).textContent = value; }
function formatDuration(seconds) { return Math.floor(seconds / 60).toString().padStart(2, '0') + ':' + (seconds % 60).toString().padStart(2, '0'); }

function renderAlerts(alerts) {
    setText('alert-count', alerts.length);
    document.getElementById('alert-list').innerHTML = alerts.length ? alerts.map(function (alert) {
        var subject = alert.substance || (alert.fall ? '跌倒报警' : '设备报警');
        return '<div class="alert-item"><div class="alert-main"><span class="alert-device">' + alert.device + '</span><span class="alert-badge">● ACTIVE</span></div>' +
            '<div class="alert-detail"><span>报警物<b>' + subject + '</b></span><span>当前浓度<b>' + (alert.current ? alert.current.toFixed(1) + ' <small>ppm</small>' : '--') + '</b></span><span>最高浓度<b>' + (alert.max ? alert.max.toFixed(1) + ' <small>ppm</small>' : '--') + '</b></span></div>' +
            '<div class="alert-time">' + alert.group + ' · ' + alert.started_at + ' · 已持续 ' + formatDuration(alert.duration) + (alert.fall ? ' · <span class="fall-flag">跌倒信号</span>' : '') + '</div></div>';
    }).join('') : '<div class="empty-state">当前没有活动报警</div>';
}

function renderDevices(devices) {
    document.getElementById('device-table').innerHTML = devices.map(function (device) {
        var statusText = { normal: '正常', alert: '报警', offline: '离线' }[device.status];
        var modeText = device.mode === 'training' ? '训练' : '监测';
        return '<tr><td>' + device.name + '</td><td>' + device.group + '</td><td><span class="status-dot status-' + device.status + '"></span>' + statusText + '</td><td class="mode-' + device.mode + '">' + modeText + '</td><td class="' + (device.status === 'alert' ? 'concentration-alert' : '') + '">' + device.conc.toFixed(1) + ' ppm</td><td class="' + (device.battery < 40 ? 'battery-low' : '') + '">' + device.battery + '%</td><td>' + device.rssi + ' dBm</td><td>' + device.last_seen + '</td></tr>';
    }).join('');
}

function renderCommands(commands) {
    document.getElementById('command-list').innerHTML = commands.map(function (command) {
        var pending = command.result === 'pending';
        var resultText = { success: '成功', failed: '失败', timeout: '超时', pending: '执行中' }[command.result] || command.result;
        var deviceResults = (command.results || []).map(function (result) {
            var text = { success: '成功', failed: '失败', timeout: '超时', pending: '等待 ACK' }[result.result] || result.result;
            return '<span class="command-result command-result-' + result.result + '">' + result.device_id + ' · ' + text + (result.reason ? ' · ' + result.reason : '') + '</span>';
        }).join('');
        return '<div class="command-item ' + (pending ? 'command-pending' : '') + '"><div class="command-line"><span>' + command.action + ' · ' + command.target + '</span><span>' + resultText + ' ' + command.progress + '%</span></div><div class="progress-track"><i style="width:' + command.progress + '%"></i></div><div class="command-results">' + deviceResults + '</div><div class="command-id">' + command.id + '</div></div>';
    }).join('');
}

function renderTrainings(trainings) {
    document.getElementById('training-list').innerHTML = trainings.length ? trainings.map(function (training) {
        var statusText = { starting: '启动中', active: '进行中', ended: '已结束' }[training.status] || training.status;
        return '<div class="training-item"><div><strong>' + training.name + '</strong><span>' + training.group + ' · ' + training.devices.length + ' 台设备</span></div><div class="training-state training-' + training.status + '">' + training.mode + ' · ' + statusText + '</div><time>' + training.started_at + (training.ended_at ? ' - ' + training.ended_at : '') + '</time></div>';
    }).join('') : '<div class="empty-state">暂无训练任务</div>';
}

function renderMarkers(devices) {
    markers.clearLayers();
    devices.forEach(function (device) {
        var icon = L.divIcon({ className: '', html: '<div class="device-marker ' + device.status + '"></div>', iconSize: [14, 14], iconAnchor: [7, 7] });
        L.marker([device.lat, device.lng], { icon: icon }).bindPopup('<strong>' + device.name + '</strong><br>' + device.group + '<br>状态：' + device.status + '<br>浓度：' + device.conc.toFixed(1) + ' ppm').addTo(markers);
    });
}

function renderSnapshot(snapshot) {
    setText('metric-total', snapshot.summary.total_devices); setText('metric-online', snapshot.summary.online_devices); setText('metric-alerts', snapshot.summary.alert_devices); setText('metric-offline', snapshot.summary.offline_devices); setText('metric-training', snapshot.summary.training_devices);
    setText('map-updated', '数据更新 ' + new Date(snapshot.updated_at).toLocaleTimeString('zh-CN', { hour12: false }));
    renderAlerts(snapshot.alerts); renderDevices(snapshot.devices); renderCommands(snapshot.commands); renderTrainings(snapshot.trainings || []); renderMarkers(snapshot.devices);
}

function refresh() { fetch('/api/dashboard/snapshot').then(function (response) { return response.json(); }).then(renderSnapshot).catch(function () { document.querySelector('.connection-pill').innerHTML = '<i style="background:#f37d6c"></i> 数据链路异常'; }); }
function updateClock() { setText('system-time', new Date().toLocaleTimeString('zh-CN', { hour12: false })); }

refresh(); updateClock(); setInterval(refresh, 15000); setInterval(updateClock, 1000);
