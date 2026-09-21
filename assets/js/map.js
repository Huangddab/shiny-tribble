var map = L.map('map-container', { minZoom: 14, maxZoom: 18, zoomControl: false }).setView([22.6335, 113.9035], 17);
var markers = L.layerGroup().addTo(map);
var markerViewKey = '';

L.tileLayer('http://localhost:8081/data/baoan/{z}/{x}/{y}.png', { minZoom: 14, maxZoom: 18, maxNativeZoom: 18, attribution: '' }).addTo(map);

function $(id) { return document.getElementById(id); }
function setText(id, value) { $(id).textContent = value; }
function formatDuration(seconds) { seconds = seconds || 0; return Math.floor(seconds / 60).toString().padStart(2, '0') + ':' + (seconds % 60).toString().padStart(2, '0'); }
function escapeHtml(value) {
    return String(value == null ? '' : value).replace(/[&<>"']/g, function (ch) {
        return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch];
    });
}

function apiRequest(method, url, body) {
    var options = { method: method, headers: {} };
    if (body !== undefined) {
        options.headers['Content-Type'] = 'application/json';
        options.body = JSON.stringify(body);
    }
    return fetch(url, options).then(function (response) {
        if (response.status === 204) { return null; }
        return response.text().then(function (text) {
            var data = null;
            try { data = text ? JSON.parse(text) : null; } catch (err) { /* not JSON */ }
            if (!response.ok) { throw new Error((data && data.error) || ('请求失败 (' + response.status + ')')); }
            return data;
        });
    });
}

var toastTimer = null;
function toast(message, isError) {
    var el = $('toast');
    el.textContent = message;
    el.classList.toggle('error', !!isError);
    el.classList.add('show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { el.classList.remove('show'); }, 3200);
}

/* ---------- dashboard render ---------- */

function renderAlerts(alerts) {
    setText('alert-count', alerts.length);
    $('alert-list').innerHTML = alerts.length ? alerts.map(function (alert) {
        var subject = alert.substance || (alert.fall ? '跌倒报警' : '设备报警');
        var confirmed = alert.ack === 'confirmed';
        return '<div class="alert-item"><div class="alert-main"><span class="alert-device">' + alert.device + '</span><span class="alert-badge">● ACTIVE</span></div>' +
            '<div class="alert-detail"><span>报警物<b>' + subject + '</b></span><span>当前浓度<b>' + (alert.current ? alert.current.toFixed(1) + ' <small>ppm</small>' : '--') + '</b></span><span>最高浓度<b>' + (alert.max ? alert.max.toFixed(1) + ' <small>ppm</small>' : '--') + '</b></span></div>' +
            '<div class="alert-time">' + alert.group + ' · ' + alert.started_at + ' · 已持续 ' + formatDuration(alert.duration) + (alert.fall ? ' · <span class="fall-flag">跌倒信号</span>' : '') + '</div>' +
            '<div class="alert-actions"><span class="ack-pill' + (confirmed ? ' confirmed' : '') + '">' + (confirmed ? '已确认' : '未确认') + '</span>' +
            (confirmed ? '' : '<button type="button" class="btn btn-sm" data-alert-id="' + escapeHtml(alert.id) + '">确认报警</button>') +
            '</div></div>';
    }).join('') : '<div class="empty-state">当前没有活动报警</div>';
}

function renderDevices(devices) {
    $('device-table').innerHTML = devices.map(function (device) {
        var statusText = { normal: '正常', alert: '报警', offline: '离线' }[device.status];
        var modeText = device.mode === 'training' ? '训练' : '监测';
        return '<tr><td>' + device.name + '</td><td>' + device.group + '</td><td><span class="status-dot status-' + device.status + '"></span>' + statusText + '</td><td class="mode-' + device.mode + '">' + modeText + '</td><td class="' + (device.status === 'alert' ? 'concentration-alert' : '') + '">' + device.conc.toFixed(1) + ' ppm</td><td class="' + (device.battery < 40 ? 'battery-low' : '') + '">' + device.battery + '%</td><td>' + device.rssi + ' dBm</td><td>' + device.last_seen + '</td></tr>';
    }).join('');
}

function renderCommands(commands) {
    $('command-list').innerHTML = commands.map(function (command) {
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
    $('training-list').innerHTML = trainings.length ? trainings.map(function (training) {
        var statusText = { starting: '启动中', active: '进行中', ended: '已结束' }[training.status] || training.status;
        return '<div class="training-item"><div><strong>' + training.name + '</strong><span>' + training.group + ' · ' + training.devices.length + ' 台设备</span></div><div class="training-state training-' + training.status + '">' + training.mode + ' · ' + statusText + '</div><time>' + training.started_at + (training.ended_at ? ' - ' + training.ended_at : '') + '</time></div>';
    }).join('') : '<div class="empty-state">暂无训练任务</div>';
}

function renderActiveTrainings(trainings) {
    var active = (trainings || []).filter(function (training) { return training.status !== 'ended'; });
    $('active-training-list').innerHTML = active.length ? active.map(function (training) {
        var statusText = { starting: '启动中', active: '进行中' }[training.status] || training.status;
        return '<div class="training-item"><div><strong>' + training.name + '</strong><span>' + training.group + ' · ' + training.devices.length + ' 台设备</span></div>' +
            '<button type="button" class="btn btn-sm btn-danger" data-training-id="' + escapeHtml(training.id) + '">结束训练</button>' +
            '<div class="training-state training-' + training.status + '">' + training.mode + ' · ' + statusText + '</div><time>' + training.started_at + '</time></div>';
    }).join('') : '<div class="empty-state">当前没有进行中的训练</div>';
}

function renderMarkers(devices) {
    markers.clearLayers();
    var visibleDevices = devices.filter(function (device) {
        return device.status !== 'offline' && Number.isFinite(Number(device.lat)) && Number.isFinite(Number(device.lng)) &&
            Number(device.lat) >= -90 && Number(device.lat) <= 90 && Number(device.lng) >= -180 && Number(device.lng) <= 180;
    });
    visibleDevices.forEach(function (device) {
        var icon = L.divIcon({ className: '', html: '<div class="device-marker ' + device.status + '"></div>', iconSize: [14, 14], iconAnchor: [7, 7] });
        L.marker([Number(device.lat), Number(device.lng)], { icon: icon }).bindPopup('<strong>' + device.name + '</strong><br>' + device.group + '<br>状态：' + device.status + '<br>浓度：' + device.conc.toFixed(1) + ' ppm').addTo(markers);
    });
    var viewKey = visibleDevices.map(function (device) { return device.id + ':' + device.lat + ',' + device.lng; }).sort().join('|');
    if (viewKey && viewKey !== markerViewKey) {
        var bounds = L.latLngBounds(visibleDevices.map(function (device) { return [Number(device.lat), Number(device.lng)]; }));
        map.fitBounds(bounds, { padding: [24, 24], maxZoom: 17 });
    }
    markerViewKey = viewKey;
}

function renderDeviceSelect(devices) {
    var select = $('single-device');
    var current = select.value;
    var ids = ['all'].concat(devices.map(function (device) { return device.id; }));
    var options = ['<option value="all">全部设备（广播）</option>'].concat(devices.map(function (device) {
        return '<option value="' + escapeHtml(device.id) + '">' + escapeHtml(device.name) + ' (' + escapeHtml(device.id) + ')</option>';
    }));
    select.innerHTML = options.join('');
    select.value = ids.indexOf(current) >= 0 ? current : 'all';
}

function renderKnownDevices(devices) {
    $('known-devices').innerHTML = devices.map(function (device) { return '<option value="' + escapeHtml(device.id) + '">'; }).join('');
}

function renderDevicesManage(devices) {
    $('devices-manage-tbody').innerHTML = devices.map(function (device) {
        var statusText = { normal: '正常', alert: '报警', offline: '离线' }[device.status] || device.status;
        return '<tr><td>' + escapeHtml(device.id) + '</td><td>' + escapeHtml(device.name) + '</td><td>' + escapeHtml(device.group) + '</td>' +
            '<td><span class="status-dot status-' + device.status + '"></span>' + statusText + '</td>' +
            '<td class="' + (device.mode === 'training' ? 'mode-training' : 'mode-monitor') + '">' + (device.mode === 'training' ? '训练' : '监测') + '</td>' +
            '<td class="' + (device.battery < 40 ? 'battery-low' : '') + '">' + device.battery + '%</td>' +
            '<td><div class="table-actions">' +
            '<input class="rename-input" type="text" value="' + escapeHtml(device.name) + '" placeholder="新名称">' +
            '<button type="button" class="btn btn-sm" data-rename-id="' + escapeHtml(device.id) + '">重命名</button>' +
            '<input class="group-input" type="text" value="' + escapeHtml(device.group) + '" list="known-groups" placeholder="编队">' +
            '<button type="button" class="btn btn-sm" data-group-id="' + escapeHtml(device.id) + '">分配编队</button>' +
            '</div></td></tr>';
    }).join('');
}

function renderGroupSelects(groups) {
    var options = groups.map(function (group) { return '<option value="' + escapeHtml(group.name) + '">' + escapeHtml(group.name) + ' (' + group.total + '台)</option>'; }).join('');
    ['group-target', 'position-group', 'training-group'].forEach(function (id) {
        var select = $(id);
        var current = select.value;
        select.innerHTML = options;
        if (current && groups.some(function (group) { return group.name === current; })) { select.value = current; }
    });
    $('known-groups').innerHTML = groups.map(function (group) { return '<option value="' + escapeHtml(group.name) + '">'; }).join('');
}

function fetchGroups() {
    return apiRequest('GET', '/api/dashboard/groups').then(function (res) {
        renderGroupSelects((res && res.items) || []);
    }).catch(function () { /* keep previous options on failure */ });
}

function renderSnapshot(snapshot) {
    setText('metric-total', snapshot.summary.total_devices); setText('metric-online', snapshot.summary.online_devices); setText('metric-alerts', snapshot.summary.alert_devices); setText('metric-offline', snapshot.summary.offline_devices); setText('metric-training', snapshot.summary.training_devices);
    setText('map-updated', '数据更新 ' + new Date(snapshot.updated_at).toLocaleTimeString('zh-CN', { hour12: false }));
    renderAlerts(snapshot.alerts || []);
    renderDevices(snapshot.devices || []);
    renderCommands(snapshot.commands || []);
    renderTrainings(snapshot.trainings || []);
    renderActiveTrainings(snapshot.trainings || []);
    renderMarkers(snapshot.devices || []);
    renderDeviceSelect(snapshot.devices || []);
    renderKnownDevices(snapshot.devices || []);
    renderDevicesManage(snapshot.devices || []);
    document.querySelector('.connection-pill').innerHTML = '<i></i> 数据链路正常';
}

function refresh() {
    fetch('/api/dashboard/snapshot').then(function (response) { return response.json(); }).then(renderSnapshot).catch(function () {
        document.querySelector('.connection-pill').innerHTML = '<i style="background:#ff3b30;box-shadow:0 0 10px #ff3b30"></i> 数据链路异常';
    });
    fetchGroups();
}
function updateClock() { setText('system-time', new Date().toLocaleTimeString('zh-CN', { hour12: false })); }

/* ---------- page navigation ---------- */

document.querySelectorAll('.nav-btn').forEach(function (btn) {
    btn.addEventListener('click', function () {
        document.querySelectorAll('.nav-btn').forEach(function (b) { b.classList.toggle('active', b === btn); });
        document.querySelectorAll('.page').forEach(function (p) { p.classList.toggle('active', p.id === 'page-' + btn.dataset.page); });
        if (btn.dataset.page === 'history' && !historyLoaded) { loadHistory(); }
    });
});

/* ---------- command center: dynamic notify fields ---------- */

function fieldsMarkup(type) {
    switch (String(type)) {
        case '0':
            return '<label>通知内容<select class="msg-notify-code"><option value="1">警报</option><option value="2">正常</option><option value="3">撤离</option></select></label>';
        case '1':
            return '<label>动作<select class="msg-action">' +
                '<option value="evacuate">立即撤离</option>' +
                '<option value="assemble">集合</option>' +
                '<option value="silent_on">开启静默</option>' +
                '<option value="silent_off">解除静默</option>' +
                '<option value="goto">前往指定坐标</option>' +
                '</select></label>' +
                '<div class="form-row goto-row" hidden>' +
                '<label>纬度<input class="msg-lat" type="number" step="any"></label>' +
                '<label>经度<input class="msg-lng" type="number" step="any"></label>' +
                '</div>';
        case '2':
            return '';
        case '3':
            return '<label>目标模式<select class="msg-mode"><option value="training">训练 training</option><option value="monitor">监测 monitor</option></select></label>';
        default:
            return '';
    }
}

function renderTypeFields(containerId, type) {
    var container = $(containerId);
    container.innerHTML = fieldsMarkup(type);
    var actionSelect = container.querySelector('.msg-action');
    if (actionSelect) {
        actionSelect.addEventListener('change', function () {
            container.querySelector('.goto-row').hidden = actionSelect.value !== 'goto';
        });
    }
}

function readMessage(containerId, type) {
    var container = $(containerId);
    switch (String(type)) {
        case '0': {
            return Number(container.querySelector('.msg-notify-code').value);
        }
        case '1': {
            var action = container.querySelector('.msg-action').value;
            if (action === 'goto') {
                var lat = parseFloat(container.querySelector('.msg-lat').value);
                var lng = parseFloat(container.querySelector('.msg-lng').value);
                if (isNaN(lat) || isNaN(lng)) { throw new Error('请输入有效的经纬度'); }
                return { action: action, lat: lat, lng: lng };
            }
            return { action: action };
        }
        case '2':
            return {};
        case '3':
            return { mode: container.querySelector('.msg-mode').value };
        default:
            return {};
    }
}

renderTypeFields('single-fields', $('single-type').value);
$('single-type').addEventListener('change', function () { renderTypeFields('single-fields', this.value); });
renderTypeFields('group-fields', $('group-type').value);
$('group-type').addEventListener('change', function () { renderTypeFields('group-fields', this.value); });
$('training-source-toggle').addEventListener('change', function () { $('training-source-fields').hidden = !this.checked; });

/* ---------- command center: form submissions ---------- */

$('single-command-form').addEventListener('submit', function (e) {
    e.preventDefault();
    var deviceId = $('single-device').value;
    var type = parseInt($('single-type').value, 10);
    var feedback = $('single-feedback');
    var message;
    try { message = readMessage('single-fields', type); } catch (err) { feedback.textContent = err.message; feedback.className = 'feedback error'; return; }
    feedback.textContent = '下发中…'; feedback.className = 'feedback';
    apiRequest('POST', '/api/dashboard/commands', { device_id: deviceId, type: type, message: message }).then(function (res) {
        feedback.textContent = '已下发，command_id: ' + (res && res.id ? res.id : '-'); feedback.className = 'feedback success';
        toast('指令已下发'); refresh();
    }).catch(function (err) {
        feedback.textContent = err.message; feedback.className = 'feedback error';
        toast(err.message, true);
    });
});

$('group-command-form').addEventListener('submit', function (e) {
    e.preventDefault();
    var group = $('group-target').value;
    var type = parseInt($('group-type').value, 10);
    var feedback = $('group-feedback');
    if (!group) { feedback.textContent = '暂无可用编队'; feedback.className = 'feedback error'; return; }
    var message;
    try { message = readMessage('group-fields', type); } catch (err) { feedback.textContent = err.message; feedback.className = 'feedback error'; return; }
    feedback.textContent = '下发中…'; feedback.className = 'feedback';
    apiRequest('POST', '/api/dashboard/groups/' + encodeURIComponent(group) + '/commands', { type: type, message: message }).then(function (res) {
        feedback.textContent = '已下发，command_id: ' + (res && res.id ? res.id : '-'); feedback.className = 'feedback success';
        toast('编队指令已下发'); refresh();
    }).catch(function (err) {
        feedback.textContent = err.message; feedback.className = 'feedback error';
        toast(err.message, true);
    });
});

$('training-start-form').addEventListener('submit', function (e) {
    e.preventDefault();
    var group = $('training-group').value;
    var name = $('training-name').value.trim();
    var feedback = $('training-feedback');
    if (!group) { feedback.textContent = '暂无可用编队'; feedback.className = 'feedback error'; return; }
    if (!name) { feedback.textContent = '请输入训练名称'; feedback.className = 'feedback error'; return; }
    var body = { name: name };
    if ($('training-source-toggle').checked) {
        body.source = {
            substance: $('training-substance').value.trim(),
            conc: parseFloat($('training-conc').value),
            radius: parseFloat($('training-radius').value),
            diffusion: $('training-diffusion').value.trim()
        };
    }
    feedback.textContent = '启动中…'; feedback.className = 'feedback';
    apiRequest('POST', '/api/dashboard/groups/' + encodeURIComponent(group) + '/trainings', body).then(function () {
        feedback.textContent = '训练已启动'; feedback.className = 'feedback success';
        toast('训练已启动');
        $('training-start-form').reset();
        $('training-source-fields').hidden = true;
        refresh();
    }).catch(function (err) {
        feedback.textContent = err.message; feedback.className = 'feedback error';
        toast(err.message, true);
    });
});

$('active-training-list').addEventListener('click', function (e) {
    var btn = e.target.closest('button[data-training-id]');
    if (!btn) { return; }
    apiRequest('POST', '/api/dashboard/trainings/' + encodeURIComponent(btn.dataset.trainingId) + '/end').then(function () {
        toast('训练已结束'); refresh();
    }).catch(function (err) { toast(err.message, true); });
});

$('alert-list').addEventListener('click', function (e) {
    var btn = e.target.closest('button[data-alert-id]');
    if (!btn) { return; }
    apiRequest('POST', '/api/dashboard/alerts/' + encodeURIComponent(btn.dataset.alertId) + '/confirm').then(function () {
        toast('报警已确认'); refresh();
    }).catch(function (err) { toast(err.message, true); });
});

$('devices-manage-tbody').addEventListener('click', function (e) {
    var renameBtn = e.target.closest('button[data-rename-id]');
    if (renameBtn) {
        var name = renameBtn.closest('tr').querySelector('.rename-input').value.trim();
        if (!name) { toast('请输入设备名称', true); return; }
        apiRequest('PATCH', '/api/dashboard/devices/' + encodeURIComponent(renameBtn.dataset.renameId), { name: name }).then(function () {
            toast('设备名称已更新'); refresh();
        }).catch(function (err) { toast(err.message, true); });
        return;
    }
    var groupBtn = e.target.closest('button[data-group-id]');
    if (groupBtn) {
        var group = groupBtn.closest('tr').querySelector('.group-input').value.trim();
        if (!group) { toast('请输入编队名称', true); return; }
        apiRequest('PUT', '/api/dashboard/devices/' + encodeURIComponent(groupBtn.dataset.groupId) + '/group', { group: group }).then(function () {
            toast('编队已更新'); refresh();
        }).catch(function (err) { toast(err.message, true); });
    }
});

/* ---------- alert history ---------- */

var historyLoaded = false;
function toUnix(value) {
    if (!value) { return ''; }
    var time = new Date(value).getTime();
    return isNaN(time) ? '' : Math.floor(time / 1000);
}

function historyParams() {
    var params = new URLSearchParams();
    var deviceId = $('history-device').value.trim();
    var group = $('history-group').value.trim();
    var from = toUnix($('history-from').value);
    var to = toUnix($('history-to').value);
    if (deviceId) { params.set('device_id', deviceId); }
    if (group) { params.set('group', group); }
    if (from) { params.set('from', from); }
    if (to) { params.set('to', to); }
    return params;
}

function renderHistory(items) {
    $('history-tbody').innerHTML = items.length ? items.map(function (item) {
        var reasonText = { normal: '正常结束', offline: '设备离线' }[item.end_reason] || item.end_reason || '-';
        var ackText = item.ack === 'confirmed' ? '已确认' : '未确认';
        return '<tr><td>' + item.device + '</td><td>' + item.group + '</td><td>' + (item.substance || '-') + '</td><td>' + (item.fall ? '是' : '否') + '</td><td>' + item.started_at + '</td><td>' + (item.ended_at || '-') + '</td><td>' + formatDuration(item.duration) + '</td><td>' + (item.max ? item.max.toFixed(1) + ' ppm' : '-') + '</td><td>' + reasonText + '</td><td>' + ackText + '</td></tr>';
    }).join('') : '<tr><td colspan="10" class="empty-state">暂无符合条件的记录</td></tr>';
}

function loadHistory() {
    historyLoaded = true;
    var params = historyParams();
    params.set('limit', '200');
    var feedback = $('history-feedback');
    feedback.textContent = '查询中…'; feedback.className = 'feedback';
    apiRequest('GET', '/api/dashboard/alerts/history?' + params.toString()).then(function (res) {
        var items = (res && res.items) || [];
        renderHistory(items);
        feedback.textContent = '共 ' + items.length + ' 条记录'; feedback.className = 'feedback success';
    }).catch(function (err) {
        feedback.textContent = err.message; feedback.className = 'feedback error';
    });
}

$('history-filter-form').addEventListener('submit', function (e) { e.preventDefault(); loadHistory(); });
$('history-export-btn').addEventListener('click', function () {
    window.open('/api/dashboard/alerts/export?' + historyParams().toString(), '_blank');
});

refresh(); updateClock(); setInterval(refresh, 15000); setInterval(updateClock, 1000);

