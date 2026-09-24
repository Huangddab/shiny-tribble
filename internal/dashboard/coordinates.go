package dashboard

import "math"

// wgs84ToGCJ02 converts GNSS coordinates to the coordinate system used by
// Amap tiles. Coordinates outside mainland China are returned unchanged.
func wgs84ToGCJ02(lat, lng float64) (float64, float64) {
	if lng < 72.004 || lng > 137.8347 || lat < 0.8293 || lat > 55.8271 {
		return lat, lng
	}
	const a = 6378245.0
	const ee = 0.006693421622965943
	x, y := lng-105.0, lat-35.0
	dLat := -100.0 + 2*x + 3*y + 0.2*y*y + 0.1*x*y + 0.2*math.Sqrt(math.Abs(x))
	dLat += (20*math.Sin(6*x*math.Pi) + 20*math.Sin(2*x*math.Pi))*2/3
	dLat += (20*math.Sin(y*math.Pi) + 40*math.Sin(y/3*math.Pi))*2/3
	dLat += (160*math.Sin(y/12*math.Pi) + 320*math.Sin(y*math.Pi/30))*2/3
	dLng := 300.0 + x + 2*y + 0.1*x*x + 0.1*x*y + 0.1*math.Sqrt(math.Abs(x))
	dLng += (20*math.Sin(6*x*math.Pi) + 20*math.Sin(2*x*math.Pi))*2/3
	dLng += (20*math.Sin(x*math.Pi) + 40*math.Sin(x/3*math.Pi))*2/3
	dLng += (150*math.Sin(x/12*math.Pi) + 300*math.Sin(x/30*math.Pi))*2/3
	radLat := lat * math.Pi / 180
	sinLat := math.Sin(radLat)
	magic := 1 - ee*sinLat*sinLat
	sqrtMagic := math.Sqrt(magic)
	dLat = dLat * 180 / ((a * (1 - ee) / (magic * sqrtMagic)) * math.Pi)
	dLng = dLng * 180 / ((a / sqrtMagic * math.Cos(radLat)) * math.Pi)
	return lat + dLat, lng + dLng
}
