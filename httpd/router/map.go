package router

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
)

func MapView() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "map/index.html", gin.H{})
	}
}

// WGS842Tile 将WGS84经纬度转换为指定缩放级别(zoom)下的瓦片坐标(x, y, z)
func WGS842Tile(lat float64, lng float64, zoom int) (x int, y int, z int) {
	// n = 2^zoom
	n := math.Exp2(float64(zoom))

	// 1. 计算 X 坐标
	x = int(math.Floor((lng + 180.0) / 360.0 * n))

	// 2. 计算 Y 坐标
	latRad := lat * math.Pi / 180.0
	y = int(math.Floor((1.0 - math.Log(math.Tan(latRad)+(1.0/math.Cos(latRad)))/math.Pi) / 2.0 * n))

	z = zoom

	// 3. 边界限制 (Clamping)
	// 确保 x 和 y 的值在合法的索引范围内: [0, 2^zoom - 1]
	// 使用位移运算 1 << zoom 来快速计算 2^zoom
	maxTileIndex := (1 << zoom) - 1

	if x < 0 {
		x = 0
	} else if x > maxTileIndex {
		x = maxTileIndex
	}

	if y < 0 {
		y = 0
	} else if y > maxTileIndex {
		y = maxTileIndex
	}

	return x, y, z
}

// WGS842Mercator 将 WGS84 经纬度转换为 Web Mercator (EPSG:3857) 物理坐标(单位: 米)
func WGS842Mercator(lat float64, lng float64) (x float64, y float64) {
	// 1. 极点保护：限制纬度范围，防止 Y 轴计算出现无穷大 (+Inf / -Inf)
	// 85.05112878 是 Web Mercator 投影的最大有效纬度 (使得地图呈现正方形)
	const maxLat = 85.05112877980659
	if lat > maxLat {
		lat = maxLat
	} else if lat < -maxLat {
		lat = -maxLat
	}

	// 2. Web Mercator 的标准地球赤道半径 (单位: 米)
	const earthRadius = 6378137.0

	// 3. 计算 X 坐标 (经度转米)
	// 公式: x = R * lon_rad
	x = earthRadius * (lng * math.Pi / 180.0)

	// 4. 计算 Y 坐标 (纬度转米)
	// 公式: y = R * ln(tan(PI/4 + lat_rad/2))
	latRad := lat * math.Pi / 180.0
	y = earthRadius * math.Log(math.Tan(math.Pi/4.0+latRad/2.0))

	return x, y
}

// WGS84ToTilePixel 计算经纬度所在的瓦片编号，以及在瓦片内的局部像素坐标
// 返回值:
//
//	tileX, tileY: 瓦片编号 (用于下载或定位瓦片)
//	pixelX, pixelY: 该经纬度在这个 256x256 瓦片图片上的局部像素坐标 (范围: 0~255)
func WGS84ToTilePixel(lat, lng float64, zoom int) (tileX, tileY int, pixelX, pixelY int) {
	const tileSize = 256.0 // 标准 Web 瓦片尺寸为 256x256 像素

	// 1. 限制纬度范围，防止极点崩溃
	const maxLat = 85.05112877980659
	if lat > maxLat {
		lat = maxLat
	} else if lat < -maxLat {
		lat = -maxLat
	}

	n := math.Exp2(float64(zoom))

	// 2. 计算精确的瓦片浮点坐标 (不取整)
	exactX := (lng + 180.0) / 360.0 * n

	latRad := lat * math.Pi / 180.0
	exactY := (1.0 - math.Log(math.Tan(latRad)+(1.0/math.Cos(latRad)))/math.Pi) / 2.0 * n

	// 3. 提取整数部分，得到瓦片编号
	tileX = int(math.Floor(exactX))
	tileY = int(math.Floor(exactY))

	// 4. 提取小数部分，乘以瓦片尺寸，得到局部像素坐标
	// math.Mod(exactX, 1.0) 相当于 exactX - math.Floor(exactX)
	fractionalX := exactX - math.Floor(exactX)
	fractionalY := exactY - math.Floor(exactY)

	pixelX = int(math.Floor(fractionalX * tileSize))
	pixelY = int(math.Floor(fractionalY * tileSize))

	return tileX, tileY, pixelX, pixelY
}
