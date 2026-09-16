console.log('map.js loaded');

var map = L.map('map-container', {
    minZoom: 16,
    maxZoom: 18
}).setView([22.6335, 113.9035], 17);

console.log('map created');

var tileServerUrl =
    'http://localhost:8081/data/baoan/{z}/{x}/{y}.png';

var tileLayer = L.tileLayer(tileServerUrl, {
    minZoom: 16,
    maxZoom: 18,
    maxNativeZoom: 18,
    attribution: ''
});

tileLayer.on('tileload', function (e) {
    console.log('tile loaded:', e.tile.src);
});

tileLayer.on('tileerror', function (e) {
    console.error('tile error:', e.tile.src);
});

tileLayer.addTo(map);

console.log('tile layer added');