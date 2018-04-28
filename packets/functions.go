package packets

import "math"

//CalculateFal : Calculates the two bytes of a FlagsAndLength field of a sACN packet
func calculateFal(length uint16) [2]byte {
	return [2]byte{
		byte(0x70) + byte((length>>8)&0x0F),
		byte(0xFF & length)}
}

func getAsBytes32(i uint32) []byte {
	return []byte{byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i & 0xFF)}
}
func getAsBytes16(i uint16) []byte {
	return []byte{byte(i >> 8), byte(i & 0xFF)}
}

func getAsUint32(arr []byte) uint32 {
	value := uint32(0)
	for i := range arr {
		//calculate in int an then convert to uint32
		value += uint32(float64(arr[i]) * math.Pow(256, float64(len(arr)-i-1)))
	}
	return value
}
