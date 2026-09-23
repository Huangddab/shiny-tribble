var map = L.map('map-container', { minZoom: 14, maxZoom: 18, zoomControl: false }).setView([22.6335, 113.9035], 17);
var markers = L.layerGroup().addTo(map);
var markerViewKey = '';
var refreshTimer = null;
var authRequired = false;

L.tileLayer('http://localhost:8081/data/baoan/{z}/{x}/{y}.png', { minZoom: 14, maxZoom: 18, maxNativeZoom: 18, attribution: '' }).addTo(map);

function $(id) { return document.getElementById(id); }
function setText(id, value) { $(id).textContent = value; }
function formatDuration(seconds) { seconds = seconds || 0; return Math.floor(seconds / 60).toString().padStart(2, '0') + ':' + (seconds % 60).toString().padStart(2, '0'); }
function escapeHtml(value) {
    return String(value == null ? '' : value).replace(/[&<>"']/g, function (ch) {
        return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch];
    });
}

function showLogin() {
    if (authRequired) { return; }
    authRequired = true;
    if (refreshTimer) { clearInterval(refreshTimer); }
    $('login-gate').hidden = false;
    $('login-username').focus();
}

function hideLogin() {
    authRequired = false;
    $('login-gate').hidden = true;
    refresh();
    refreshTimer = setInterval(refresh, 2000);
}

function apiRequest(method, url, body) {
    var options = { method: method, headers: {} };
    if (body !== undefined) {
        options.headers['Content-Type'] = 'application/json';
        options.body = JSON.stringify(body);
    }
    return fetch(url, options).then(function (response) {
        if (response.status === 401) { showLogin(); }
        if (response.status === 204) { return null; }
        return response.text().then(function (text) {
            var data = null;
            try { data = text ? JSON.parse(text) : null; } catch (err) { /* not JSON */ }
            if (!response.ok) { throw new Error((data && data.error) || ('请求失败 (' + response.status + ')')); }
            return data;
        });
    });
}

$('login-form').addEventListener('submit', function (event) {
    event.preventDefault();
    var feedback = $('login-feedback');
    feedback.textContent = '登录中…';
    fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: $('login-username').value, password: $('login-password').value })
    }).then(function (response) {
        return response.json().then(function (data) {
            if (!response.ok) { throw new Error((data && data.error) || '登录失败'); }
            return data;
        });
    }).then(function () {
        feedback.textContent = '';
        hideLogin();
    }).catch(function (error) {
        feedback.textContent = error.message;
    });
});

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
        return '<div class="alert-item"><div class="alert-main"><span class="alert-device">' + (alert.device_name || alert.device_id) + '</span><span class="alert-badge">● ACTIVE</span></div>' +
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
        return '<tr><td class="device-code">' + escapeHtml(device.id) + '</td><td>' + escapeHtml(device.name) + '</td><td>' + escapeHtml(device.group || '未分组') + '</td><td><span class="status-dot status-' + device.status + '"></span>' + statusText + '</td><td class="mode-' + device.mode + '">' + modeText + '</td><td>' + escapeHtml(device.substance || '—') + '</td><td class="' + (device.status === 'alert' ? 'concentration-alert' : '') + '">' + device.conc.toFixed(1) + ' / ' + device.threshold.toFixed(1) + ' ppm</td><td class="' + (device.battery < 40 ? 'battery-low' : '') + '">' + device.battery + '%</td><td>' + device.rssi + ' dBm</td><td>' + (device.position_valid ? '已定位' : '未定位') + '</td><td>' + device.last_seen + '</td></tr>';
    }).join('');
}

function renderCommands(commands) {
    var html = commands.map(function (command) {
        var pending = command.result === 'pending';
        var resultText = { success: '成功', failed: '失败', timeout: '超时', pending: '执行中' }[command.result] || command.result;
        var deviceResults = (command.results || []).map(function (result) {
            var text = { success: '成功', failed: '失败', timeout: '超时', pending: '等待 ACK' }[result.result] || result.result;
            return '<span class="command-result command-result-' + result.result + '">' + result.device_id + ' · ' + text + (result.reason ? ' · ' + result.reason : '') + '</span>';
        }).join('');
        return '<div class="command-item ' + (pending ? 'command-pending' : '') + '"><div class="command-line"><span>' + command.action + ' · ' + command.target + '</span><span>' + resultText + ' ' + command.progress + '%</span></div><div class="progress-track"><i style="width:' + command.progress + '%"></i></div><div class="command-results">' + deviceResults + '</div><div class="command-id">' + command.id + '</div></div>';
    }).join('');
    ['command-list', 'dashboard-command-list'].forEach(function (id) { var el = $(id); if (el) { el.innerHTML = html; } });
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
        return device.position_valid === true && device.status !== 'offline' && Number.isFinite(Number(device.lat)) && Number.isFinite(Number(device.lng)) &&
            Number(device.lat) >= -90 && Number(device.lat) <= 90 && Number(device.lng) >= -180 && Number(device.lng) <= 180;
    });
    visibleDevices.forEach(function (device) {
        var icon = L.divIcon({ className: '', html: '<div class="marker-wrap"><div class="device-marker ' + device.status + '"></div><span>' + escapeHtml(device.name || device.id) + '</span></div>', iconSize: [120, 38], iconAnchor: [10, 19] });
        L.marker([Number(device.lat), Number(device.lng)], { icon: icon }).bindPopup('<strong>' + escapeHtml(device.name) + '</strong><br>编号：' + escapeHtml(device.id) + '<br>编队：' + escapeHtml(device.group || '未分组') + '<br>状态：' + ({normal:'正常',alert:'报警',offline:'离线'}[device.status] || device.status) + '<br>物质：' + escapeHtml(device.substance || '—') + '<br>浓度：' + device.conc.toFixed(1) + ' ppm<br>电量：' + device.battery + '%<br>RSSI：' + device.rssi + ' dBm').addTo(markers);
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

    var quickDevice = $('quick-device');
    if (quickDevice) {
        var quickCurrent = quickDevice.value;
        var deviceOptions = devices.map(function (device) {
            return '<option value="' + escapeHtml(device.id) + '">' + escapeHtml(device.name) + ' (' + escapeHtml(device.id) + ')</option>';
        }).join('');
        quickDevice.innerHTML = deviceOptions;
        if (devices.some(function (device) { return device.id === quickCurrent; })) { quickDevice.value = quickCurrent; }
    }
}

function renderKnownDevices(devices) {
    $('known-devices').innerHTML = devices.map(function (device) { return '<option value="' + escapeHtml(device.id) + '">'; }).join('');
}

var latestDevices = [];

function renderDeviceSummary(devices) {
    var total = devices.length;
    var online = devices.filter(function (device) { return device.status !== 'offline'; }).length;
    var offline = devices.filter(function (device) { return device.status === 'offline'; }).length;
    var alerts = devices.filter(function (device) { return device.status === 'alert'; }).length;
    var training = devices.filter(function (device) { return device.mode === 'training'; }).length;
    if ($('summary-total')) { setText('summary-total', total); }
    if ($('summary-online-offline')) { setText('summary-online-offline', online + ' / ' + offline); }
    if ($('summary-alerts')) { setText('summary-alerts', alerts); }
    if ($('summary-training')) { setText('summary-training', training); }
}

function renderDevicesManage(devices) {
    latestDevices = devices;
    $('devices-manage-tbody').innerHTML = devices.map(function (device) {
        var statusText = { normal: '正常', alert: '报警', offline: '离线' }[device.status] || device.status;
        return '<tr data-status="' + device.status + '" data-group="' + escapeHtml(device.group || '') + '" data-search="' + escapeHtml((device.id + ' ' + device.name).toLowerCase()) + '">' +
            '<td class="device-id-cell">' + escapeHtml(device.id) + '</td><td>' + escapeHtml(device.name) + '</td>' +
            '<td>' + (device.group ? '<span class="group-tag">' + escapeHtml(device.group) + '</span>' : '未分组') + '</td>' +
            '<td><span class="status-dot status-' + device.status + '"></span>' + statusText + '</td>' +
            '<td class="' + (device.mode === 'training' ? 'mode-training' : 'mode-monitor') + '">' + (device.mode === 'training' ? '训练' : '监测') + '</td>' +
            '<td class="' + (device.status === 'alert' ? 'concentration-alert' : '') + '">' + device.conc.toFixed(1) + ' / ' + device.threshold.toFixed(1) + ' ppm</td>' +
            '<td class="' + (device.battery < 40 ? 'battery-low' : '') + '">' + device.battery + '%</td>' +
            '<td>' + device.rssi + ' dBm</td><td>' + (device.position_valid ? '已定位' : '未定位') + '</td><td>' + escapeHtml(device.last_seen) + '</td>' +
            '<td><button type="button" class="manage-btn" data-manage-id="' + escapeHtml(device.id) + '">管理</button></td></tr>';
    }).join('');
    applyDeviceFilter();
}

function applyDeviceFilter() {
    var searchEl = $('device-search'), groupEl = $('device-group-filter'), statusEl = $('device-status-filter');
    if (!searchEl) { return; }
    var query = searchEl.value.trim().toLowerCase();
    var group = groupEl.value;
    var status = statusEl.value;
    document.querySelectorAll('#devices-manage-tbody tr').forEach(function (row) {
        var matches = (!query || row.dataset.search.indexOf(query) >= 0) &&
            (!group || row.dataset.group === group) &&
            (!status || row.dataset.status === status);
        row.hidden = !matches;
    });
}

var latestGroups = [];

function renderGroupSelects(groups) {
    latestGroups = groups;
    var options = groups.map(function (group) { return '<option value="' + escapeHtml(group.name) + '">' + escapeHtml(group.name) + ' (' + group.total + '/' + group.capacity + '台)</option>'; }).join('');
    ['group-target', 'training-group', 'quick-group'].forEach(function (id) {
        var select = $(id);
        if (!select) { return; }
        var current = select.value;
        select.innerHTML = options;
        if (current && groups.some(function (group) { return group.name === current; })) { select.value = current; }
    });
    if ($('device-group-filter')) {
        var filterSelect = $('device-group-filter');
        var currentFilter = filterSelect.value;
        filterSelect.innerHTML = '<option value="">全部编队</option>' + options;
        filterSelect.value = currentFilter;
    }
    if ($('modal-device-group')) { $('modal-device-group').placeholder = groups.length ? '选择已有或输入新编队' : '输入新编队名称'; }
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
    renderDeviceSummary(snapshot.devices || []);
    if ($('settings-device-count')) { setText('settings-device-count', snapshot.summary.total_devices + ' 台'); }
    if ($('settings-updated-at')) { setText('settings-updated-at', new Date(snapshot.updated_at).toLocaleString('zh-CN', { hour12: false })); }
    document.querySelector('.connection-pill').innerHTML = '<i></i> 数据链路正常';
}

function refresh() {
    fetch('/api/dashboard/snapshot').then(function (response) {
        if (response.status === 401) { showLogin(); }
        return response.json();
    }).then(renderSnapshot).catch(function () {
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
        if (btn.dataset.page === 'settings') { loadUsers(); }
    });
});

function renderUsers(users) {
    var body = $('user-table-body');
    if (!body) { return; }
    body.innerHTML = users.length ? users.map(function (user) {
        var role = user.role === 'admin' ? '管理员' : '普通用户';
        var created = user.created_at ? new Date(user.created_at).toLocaleString('zh-CN', { hour12: false }) : '-';
        return '<tr><td>' + escapeHtml(user.username) + '</td><td>' + role + '</td><td>' + escapeHtml(user.email || '-') + '</td><td>' + created + '</td>' +
            '<td><button type="button" class="btn btn-sm btn-danger" data-delete-user="' + escapeHtml(user.username) + '">删除</button></td></tr>';
    }).join('') : '<tr><td colspan="5" class="empty-state">暂无用户</td></tr>';
}

function loadUsers() {
    var state = $('user-management-state');
    apiRequest('GET', '/api/dashboard/users').then(function (res) {
        renderUsers((res && res.items) || []);
        if (state) { state.textContent = '管理员权限'; }
    }).catch(function (err) {
        if (state) { state.textContent = err.message.indexOf('administrator') >= 0 ? '仅管理员可用' : '加载失败'; }
        if ($('user-table-body')) { $('user-table-body').innerHTML = '<tr><td colspan="5" class="empty-state">暂无权限查看</td></tr>'; }
    });
}

if ($('user-create-form')) {
    $('user-create-form').addEventListener('submit', function (event) {
        event.preventDefault();
        var feedback = $('user-feedback');
        feedback.textContent = '创建中…'; feedback.className = 'feedback';
        apiRequest('POST', '/api/dashboard/users', {
            username: $('user-username').value.trim(),
            password: $('user-password').value,
            email: $('user-email').value.trim(),
            role: $('user-role').value
        }).then(function () {
            feedback.textContent = '用户已创建'; feedback.className = 'feedback success';
            $('user-create-form').reset();
            loadUsers();
        }).catch(function (err) {
            feedback.textContent = err.message; feedback.className = 'feedback error';
        });
    });
}

if ($('user-table-body')) {
    $('user-table-body').addEventListener('click', function (event) {
        var button = event.target.closest('button[data-delete-user]');
        if (!button || !window.confirm('确定删除用户“' + button.dataset.deleteUser + '”吗？')) { return; }
        apiRequest('DELETE', '/api/dashboard/users/' + encodeURIComponent(button.dataset.deleteUser)).then(function () {
            toast('用户已删除'); loadUsers();
        }).catch(function (err) { toast(err.message, true); });
    });
}

document.querySelectorAll('[data-page-link]').forEach(function (btn) {
    btn.addEventListener('click', function () {
        var target = btn.dataset.pageLink;
        var navBtn = document.querySelector('.nav-btn[data-page="' + target + '"]');
        if (navBtn) { navBtn.click(); }
    });
});

['device-search', 'device-group-filter', 'device-status-filter'].forEach(function (id) {
    var el = $(id);
    if (!el) { return; }
    el.addEventListener('input', applyDeviceFilter);
    el.addEventListener('change', applyDeviceFilter);
});

/* ---------- device management modal ---------- */

var editingDeviceId = null;
function updateGroupCapacity() {
    var group = $('modal-device-group').value;
    var groupInfo = latestGroups.find(function (item) { return item.name === group; });
    var currentDevice = latestDevices.find(function (device) { return device.id === editingDeviceId; });
    var alreadyInGroup = currentDevice && currentDevice.group === group;
    var total = groupInfo ? groupInfo.total - (alreadyInGroup ? 1 : 0) : 0;
    var capacity = groupInfo ? groupInfo.capacity : 10;
    setText('group-capacity-text', group ? (total + ' / ' + capacity + ' 台') : '不限制');
    $('group-capacity-bar').style.width = group ? Math.min(100, (total / capacity) * 100) + '%' : '0%';
}
if ($('modal-device-group')) { $('modal-device-group').addEventListener('input', updateGroupCapacity); }

document.getElementById('devices-manage-tbody').addEventListener('click', function (e) {
    var btn = e.target.closest('button[data-manage-id]');
    if (!btn) { return; }
    var device = latestDevices.find(function (item) { return item.id === btn.dataset.manageId; });
    if (!device) { return; }
    editingDeviceId = device.id;
    setText('modal-device-id', 'DEVICE ID · ' + device.id);
    $('modal-device-name').value = device.name;
    $('modal-device-group').value = device.group || '';
    updateGroupCapacity();
    $('device-modal-backdrop').classList.add('show');
    $('modal-device-name').focus();
});

function closeDeviceModal() { $('device-modal-backdrop').classList.remove('show'); editingDeviceId = null; }
$('device-modal-close').addEventListener('click', closeDeviceModal);
$('device-modal-cancel').addEventListener('click', closeDeviceModal);
$('device-modal-backdrop').addEventListener('click', function (e) { if (e.target === $('device-modal-backdrop')) { closeDeviceModal(); } });
document.addEventListener('keydown', function (e) { if (e.key === 'Escape') { closeDeviceModal(); } });

$('device-modal-save').addEventListener('click', function () {
    if (!editingDeviceId) { return; }
    var name = $('modal-device-name').value.trim();
    var group = $('modal-device-group').value;
    if (!name) { toast('设备名称不能为空', true); return; }
    var device = latestDevices.find(function (item) { return item.id === editingDeviceId; });
    var tasks = [];
    if (!device || device.name !== name) { tasks.push(apiRequest('PATCH', '/api/dashboard/devices/' + encodeURIComponent(editingDeviceId), { name: name })); }
    if (group && (!device || device.group !== group)) { tasks.push(apiRequest('PUT', '/api/dashboard/devices/' + encodeURIComponent(editingDeviceId) + '/group', { group: group })); }
    Promise.all(tasks).then(function () {
        toast('设备信息已更新');
        closeDeviceModal();
        refresh();
    }).catch(function (err) { toast(err.message, true); });
});

/* ---------- quick dispatch panel ---------- */

var quickScope = 'all', quickNotify = 1;
document.querySelectorAll('.scope-options .choice').forEach(function (btn) {
    btn.addEventListener('click', function () {
        document.querySelectorAll('.scope-options .choice').forEach(function (b) { b.classList.remove('active'); });
        btn.classList.add('active');
        quickScope = btn.dataset.scope;
        $('quick-group-row').hidden = quickScope !== 'group';
        $('quick-device-row').hidden = quickScope !== 'device';
    });
});
document.querySelectorAll('.notify-options .choice').forEach(function (btn) {
    btn.addEventListener('click', function () {
        document.querySelectorAll('.notify-options .choice').forEach(function (b) { b.classList.remove('active'); });
        btn.classList.add('active');
        quickNotify = Number(btn.dataset.notify);
    });
});

function quickDispatchTarget() {
    if (quickScope === 'all') { return { kind: 'all' }; }
    if (quickScope === 'group') { return { kind: 'group', value: $('quick-group').value }; }
    return { kind: 'device', value: $('quick-device').value };
}

var trainingModeEnabled = false;
if ($('training-toggle')) {
    $('training-toggle').addEventListener('click', function () {
        var button = this;
        var next = !trainingModeEnabled;
        button.disabled = true;
        apiRequest('POST', '/api/dashboard/commands', { device_id: 'all', type: 3, message: { mode: next ? 'training' : 'monitor' } }).then(function () {
            trainingModeEnabled = next;
            button.classList.toggle('on', next);
            button.setAttribute('aria-checked', String(next));
            setText('training-toggle-state', next ? '已开启' : '关闭');
            toast(next ? '已下发全体演练模式指令' : '已下发恢复监测指令');
            refresh();
        }).catch(function (err) { toast(err.message, true); }).finally(function () { button.disabled = false; });
    });
}

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
            return '<label>目标模式<select class="msg-mode"><option value="training">训练</option><option value="monitor">监测</option></select></label>';
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


/* ---------- alert history ---------- */

var historyLoaded = false;
var historyPageSize = 20;
var historyPageIndex = 0;
var historyCursors = [{ before: '', beforeId: '' }];
var historyNextCursor = { before: 0, beforeId: '' };
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

function resetHistoryPagination() {
    historyPageIndex = 0;
    historyCursors = [{ before: '', beforeId: '' }];
    historyNextCursor = { before: 0, beforeId: '' };
}

function updateHistoryPagination(itemCount) {
    var previous = $('history-prev-btn');
    var next = $('history-next-btn');
    previous.disabled = historyPageIndex === 0;
    next.disabled = itemCount < historyPageSize || !historyNextCursor.before;
    setText('history-page-label', '第 ' + (historyPageIndex + 1) + ' 页');
}

function renderHistory(items) {
    $('history-tbody').innerHTML = items.length ? items.map(function (item) {
        var reasonText = { normal: '正常结束', offline: '设备离线' }[item.end_reason] || item.end_reason || '-';
        var ackText = item.ack === 'confirmed' ? '已确认' : '未确认';
        return '<tr><td>' + (item.device_name || item.device_id) + '</td><td>' + item.group + '</td><td>' + (item.substance || '-') + '</td><td>' + (item.fall ? '是' : '否') + '</td><td>' + item.started_at + '</td><td>' + (item.ended_at || '-') + '</td><td>' + formatDuration(item.duration) + '</td><td>' + (item.max ? item.max.toFixed(1) + ' ppm' : '-') + '</td><td>' + reasonText + '</td><td>' + ackText + '</td></tr>';
    }).join('') : '<tr><td colspan="10" class="empty-state">暂无符合条件的记录</td></tr>';
}

function loadHistory(reset) {
    historyLoaded = true;
    if (reset) { resetHistoryPagination(); }
    var params = historyParams();
    var cursor = historyCursors[historyPageIndex];
    params.set('limit', String(historyPageSize));
    if (cursor.before) { params.set('before', String(cursor.before)); }
    if (cursor.beforeId) { params.set('before_id', cursor.beforeId); }
    var feedback = $('history-feedback');
    feedback.textContent = '查询中…'; feedback.className = 'feedback';
    apiRequest('GET', '/api/dashboard/alerts/history?' + params.toString()).then(function (res) {
        var items = (res && res.items) || [];
        historyNextCursor = { before: Number(res && res.next_before) || 0, beforeId: (res && res.next_before_id) || '' };
        renderHistory(items);
        updateHistoryPagination(items.length);
        feedback.textContent = items.length + ' 条记录'; feedback.className = 'feedback success';
    }).catch(function (err) {
        feedback.textContent = err.message; feedback.className = 'feedback error';
    });
}

$('history-filter-form').addEventListener('submit', function (e) { e.preventDefault(); loadHistory(true); });
$('history-prev-btn').addEventListener('click', function () {
    if (historyPageIndex === 0) { return; }
    historyPageIndex -= 1;
    loadHistory(false);
});
$('history-next-btn').addEventListener('click', function () {
    if (!historyNextCursor.before) { return; }
    historyPageIndex += 1;
    historyCursors[historyPageIndex] = { before: historyNextCursor.before, beforeId: historyNextCursor.beforeId };
    loadHistory(false);
});
$('history-export-btn').addEventListener('click', function () {
    window.open('/api/dashboard/alerts/export?' + historyParams().toString(), '_blank');
});

refresh(); updateClock(); refreshTimer = setInterval(refresh, 2000); setInterval(updateClock, 1000);
