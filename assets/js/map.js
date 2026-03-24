// 在瓦片中心坐标设置地图初始视图
var map = L.map('map-container').setView([22.776180, 113.840332], 16);

// 添加瓦片图层
L.tileLayer('http://localhost:8081/data/shenzhen/{z}/{x}/{y}.png', {
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
}).addTo(map);

// 在瓦片中心添加标记
L.marker([22.776180, 113.840332]).addTo(map)
    .bindPopup('瓦片 13/6686/3563 中心点<br>坐标: 22.776180, 113.840332')
    .openPopup();

// 添加瓦片边界矩形，显示瓦片范围
var tileBounds = L.rectangle([
    [22.796439, 113.818359],  // 左上角
    [22.755921, 113.862305]   // 右下角
], {
    color: "#ff7800",
    weight: 2,
    fillOpacity: 0.1
}).addTo(map);

// 在边界矩形上添加提示
tileBounds.bindPopup('瓦片 13/6686/3563 范围').openPopup();

// 添加东华智造园范围多边形（基于宝安大道5003号附近坐标估算）
var donghuaParkBounds = L.polygon([
    [22.615, 113.835],  // 西北角
    [22.615, 113.845],  // 东北角  
    [22.605, 113.845],  // 东南角
    [22.605, 113.835]   // 西南角
], {
    color: "#33cc33",
    weight: 3,
    fillColor: "#33cc33",
    fillOpacity: 0.2
}).addTo(map);

// 在东华智造园多边形上添加标记和提示
donghuaParkBounds.bindPopup('东华智造园<br>地址: 宝安区航城街道三围社区宝安大道5003号<br>占地面积: 1.4万平方米<br>建筑面积: 4.7万平方米<br>主导产业: 新能源技术、智能穿戴、智慧医疗、半导体与集成电路等[1,5](@ref)').openPopup();

// 在东华智造园中心