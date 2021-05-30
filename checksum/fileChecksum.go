package checksum

import (
	"crypto/md5"
	"encoding/hex"
	"goSync/structs"
)

func HashMD5(b []byte) string {
	md := md5.Sum(b)
	return hex.EncodeToString(md[:])
}

/*
BKDRHash 2 0 4774 481 96.55 100 90.95 82.05 92.64  // best
APHash 2 3 4754 493 96.55 88.46 100 51.28 86.28
DJBHash 2 2 4975 474 96.55 92.31 0 100 83.43
JSHash 1 4 4761 506 100 84.62 96.83 17.95 81.94
RSHash 1 0 4861 505 100 100 51.58 20.51 75.96
SDBMHash 3 2 4849 504 93.1 92.31 57.01 23.08 72.41
PJWHash 30 26 4878 513 0 0 43.89 0 21.95
ELFHash 30 26 4878 513 0 0 43.89 0 21.95
*/

func bkdrHash16(b []byte) uint16 {
	var seed uint16 = 131 // 31 131 1313 13131 131313 etc..
	var hash uint16 = 0
	for i := 0; i < len(b); i++ {
		hash = hash*seed + uint16(b[i])
	}
	return hash & 0x7FFF
}

func bkdrHash32(b []byte) uint32 {
	var seed uint32 = 131 // 31 131 1313 13131 131313 etc..
	var hash uint32 = 0
	for i := 0; i < len(b); i++ {
		hash = hash*seed + uint32(b[i])
	}
	return hash & 0x7FFFFFFF
}

// BKDR Hash Function 64
func bkdrHash64(b []byte) uint64 {
	var seed uint64 = 131 // 31 131 1313 13131 131313 etc..
	var hash uint64 = 0
	for i := 0; i < len(b); i++ {
		hash = hash*seed + uint64(b[i])
	}
	return hash & 0x7FFFFFFFFFFFFFFF
}

func Hash64(b []byte) string {
	resUint64 := bkdrHash64(b)
	bArray := [8]byte{
		byte(0xFF & resUint64),
		byte(0xFF & (resUint64 >> 8)),
		byte(0xFF & (resUint64 >> 16)),
		byte(0xFF & (resUint64 >> 24)),
		byte(0xFF & (resUint64 >> 32)),
		byte(0xFF & (resUint64 >> 40)),
		byte(0xFF & (resUint64 >> 48)),
		byte(0xFF & (resUint64 >> 56)),
	}
	return hex.EncodeToString(bArray[:])
}

func Hash32(b []byte) string {
	resUint32 := bkdrHash32(b)
	bArray := [8]byte{
		byte(0xFF & resUint32),
		byte(0xFF & (resUint32 >> 8)),
		byte(0xFF & (resUint32 >> 16)),
		byte(0xFF & (resUint32 >> 24)),
	}
	return hex.EncodeToString(bArray[:])
}

func Hash16(b []byte) string {
	resUint16 := bkdrHash16(b)
	bArray := [2]byte{
		byte(0xFF & resUint16),
		byte(0xFF & (resUint16 >> 8)),
	}
	return hex.EncodeToString(bArray[:])
}

func ProduceFileCSInfo(b []byte, i int64) structs.FileCSInfo {
	md5 := HashMD5(b)
	roll := Hash64(b)
	decB, err := hex.DecodeString(roll)
	if err != nil {
		panic(err)
	}
	fastHashRoll := Hash16(decB)
	return structs.FileCSInfo{
		BlockIndex: i,
		CS16:       fastHashRoll,
		CS64:       roll,
		CS128:      md5,
	}
}

func ProduceFileCSInfoFast(b []byte, i int64) structs.FileCSInfo {
	roll := Hash64(b)
	decB, err := hex.DecodeString(roll)
	if err != nil {
		panic(err)
	}
	fastHashRoll := Hash16(decB)
	return structs.FileCSInfo{
		BlockIndex: i,
		CS16:       fastHashRoll,
		CS64:       roll,
		CS128:      "",
	}
}
