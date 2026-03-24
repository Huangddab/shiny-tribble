package router

import "testing"

func TestWGS842Tile(t *testing.T) {
	type args struct {
		lat  float64
		lng  float64
		zoom int
	}
	tests := []struct {
		name  string
		args  args
		wantX int
		wantY int
		wantZ int
	}{
		{
			name:  "Test case 1: lat=0, lng=0, zoom=0",
			args:  args{lat: 0, lng: 0, zoom: 0},
			wantX: 0,
			wantY: 0,
			wantZ: 0,
		},
		{
			name:  "Test case 2: lat=85.05112878, lng=180, zoom=1",
			args:  args{lat: 85.05112878, lng: 180, zoom: 1},
			wantX: 1,
			wantY: 0,
			wantZ: 1,
		},
		{
			name:  "Test case 3: lat=-85.05112878, lng=-180, zoom=1",
			args:  args{lat: -85.05112878, lng: -180, zoom: 1},
			wantX: 0,
			wantY: 1,
			wantZ: 1,
		},
		// 深圳宝安区测试数据 (修正为纯正 WGS-84 标准瓦片坐标)
		{
			name:  "Test case 4: Shenzhen Bao'an Center (宝安中心区/区政府), zoom=15",
			args:  args{lat: 22.55329, lng: 113.88308, zoom: 15},
			wantX: 26749, // 修正
			wantY: 14275, // 修正
			wantZ: 15,
		},
		{
			name:  "Test case 5: Shenzhen Bao'an International Airport (深圳宝安国际机场), zoom=15",
			args:  args{lat: 22.63925, lng: 113.81066, zoom: 15},
			wantX: 26743, // 保持正确
			wantY: 14267, // 修正
			wantZ: 15,
		},
		{
			name:  "Test case 6: Shenzhen Bao'an Shajing (宝安沙井街道), zoom=15",
			args:  args{lat: 22.72750, lng: 113.82900, zoom: 15},
			wantX: 26744, // 修正
			wantY: 14258, // 修正
			wantZ: 15,
		},
		{
			name:  "Test case 7: Shenzhen Bao'an Center (宝安中心区), zoom=12",
			args:  args{lat: 22.55329, lng: 113.88308, zoom: 12},
			wantX: 3343, // 保持正确
			wantY: 1784, // 修正
			wantZ: 12,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotX, gotY, gotZ := WGS842Tile(tt.args.lat, tt.args.lng, tt.args.zoom)
			if gotX != tt.wantX {
				t.Errorf("WGS842Tile() gotX = %v, want %v", gotX, tt.wantX)
			}
			if gotY != tt.wantY {
				t.Errorf("WGS842Tile() gotY = %v, want %v", gotY, tt.wantY)
			}
			if gotZ != tt.wantZ {
				t.Errorf("WGS842Tile() gotZ = %v, want %v", gotZ, tt.wantZ)
			}
		})
	}
}

func TestWGS842Mercator(t *testing.T) {
	// 深圳宝安中心区坐标
	lat := 22.55329
	lng := 113.88308

	x, y := WGS842Mercator(lat, lng)

	// 打印输出的物理米数
	t.Logf("宝安中心区 Web Mercator 坐标: X = %.2f 米, Y = %.2f 米", x, y)

	// 验证极点是否被正确拦截避免返回 NaN 或 Inf
	xPole, yPole := WGS842Mercator(90.0, 0.0)
	t.Logf("北极点(lat=90) Web Mercator 坐标: X = %.2f 米, Y = %.2f 米", xPole, yPole)
}

func TestWGS84ToTilePixel(t *testing.T) {
	// 深圳宝安中心区坐标
	lat := 22.55329
	lng := 113.88308
	zoom := 15

	tileX, tileY, pixelX, pixelY := WGS84ToTilePixel(lat, lng, zoom)

	t.Logf("坐标(Lat: %f, Lng: %f) 在 Zoom=%d 下：", lat, lng, zoom)
	t.Logf("--> 落在瓦片: X=%d, Y=%d", tileX, tileY)
	t.Logf("--> 在该瓦片上的像素位置: [ %d, %d ] (图片左上角为0,0)", pixelX, pixelY)
}
