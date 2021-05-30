package sender

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"goSync/checksum"
	"goSync/structs"
	"io"
	"os"
	"sort"
)

var debug = false

func findSameBlock(b []byte, remoteCSMap map[string]int, remoteCS []structs.FileCSInfo) (index int, isSame bool) {
	fcs := checksum.ProduceFileCSInfoFast(b, 0)
	if index, ok := remoteCSMap[fcs.CS16]; ok { // 16 match
		for i := index; remoteCS[i].CS64 == fcs.CS64 && i < len(remoteCS); i++ { // find during 16 match
			if checksum.HashMD5(b) == remoteCS[i].CS128 { // 16 && 64 match
				return i, true
			}
		}
	}
	return 0, false
}

func matchWriter(offset int64, chunkIndex int, w io.Writer) (err error) {
	contentLen := 12
	msg := make([]byte, 0, 1+4+8+4)
	_offset := make([]byte, 8)
	_chunkIndex := make([]byte, 4)
	_contentLen := make([]byte, 4)
	binary.LittleEndian.PutUint64(_offset, uint64(offset))
	binary.LittleEndian.PutUint32(_chunkIndex, uint32(chunkIndex))
	binary.LittleEndian.PutUint32(_contentLen, uint32(contentLen))
	msg = append(msg, 'c')
	msg = append(msg, _contentLen...)
	msg = append(msg, _offset...)
	msg = append(msg, _chunkIndex...)
	_, err = w.Write(msg)
	return
}

func mismatchWriter(offset int64, b []byte, w io.Writer) (err error) {
	contentLen := 8 + len(b)
	msg := make([]byte, 0, 1+4+8+len(b))
	_contentLen := make([]byte, 4)
	_offset := make([]byte, 8)
	binary.LittleEndian.PutUint64(_offset, uint64(offset))
	binary.LittleEndian.PutUint32(_contentLen, uint32(contentLen))
	msg = append(msg, 'b')
	msg = append(msg, _contentLen...)
	msg = append(msg, _offset...)
	msg = append(msg, b...)
	_, err = w.Write(msg)
	return
}

func localFileChecksumWrite(cs string, w io.Writer) (err error) {
	if len(cs) != 32 {
		panic(errors.New("128 bit string, 32 chars needed"))
	}
	contentLen := 16
	msg := make([]byte, 0, 1+4+16)
	_contentLen := make([]byte, 4)
	binary.LittleEndian.PutUint32(_contentLen, uint32(contentLen))
	csBA, _ := hex.DecodeString(cs)
	msg = append(msg, 'm')
	msg = append(msg, _contentLen...)
	msg = append(msg, csBA...)
	_, err = w.Write(msg)
	return
}

// 对 remote checksum 进行排序
// 创建hash16 fast map
// 与本地的文件block进行比较
func Checker(fp string, blockSize int, remoteCS []structs.FileCSInfo, iw io.Writer) (err error) {

	sort.Slice(remoteCS, func(i, j int) bool {
		return remoteCS[i].CS16 < remoteCS[j].CS16
	})

	remoteCS = append(remoteCS[:1000], remoteCS[1020:]...)
	remoteCS = remoteCS[:0]

	hash16Map := make(map[string]int, len(remoteCS))
	for i := 0; i < len(remoteCS); i++ {
		if _, ok := hash16Map[remoteCS[i].CS16]; !ok {
			hash16Map[remoteCS[i].CS16] = i
		} else {
			fmt.Println("hash collision: ", remoteCS[i])
		}
	}
	fmt.Println("map len", len(hash16Map))
	fmt.Println("remote infos len", len(remoteCS))

	var r, w int // [r, w)
	var finished bool

	f, err := os.OpenFile(fp, os.O_RDONLY, 0664)
	if err != nil {
		return err
	}

	defer f.Close()

	localFileMD5 := md5.New()

	var dataIndex int64
	buff := make([]byte, 0, blockSize*256)
	block := make([]byte, blockSize*256)
	failedDataBuff := make([]byte, 0, blockSize*256) // 传输不匹配bytes
	//var writeCount int
	for ; r < w || !finished; {
		buffLen := w - r
		if buffLen < blockSize && !finished { // 不足block 并且还没读完文件
			// 清除buff中已读数据
			copy(buff[:w-r], buff[r:w])
			buff = buff[:w-r]
			r, w = 0, w-r

			n, err := f.Read(block)
			localFileMD5.Write(block[:n]) // md5
			buff = append(buff, block[:n]...)
			w += n
			if err != nil {
				if err == io.EOF {
					finished = true
					err = nil
					continue
				} else {
					return err
				}
			}
		} else { // buff足够使用, 或者 已经读完文件
			// 数据处理部分
			for ; w-r >= blockSize; { // read buff
				var match bool
				var matchBlockIndex int
				fcs := checksum.ProduceFileCSInfoFast(buff[r:r+blockSize], 0)
				if index, ok := hash16Map[fcs.CS16]; ok { // 16 match
					nowMD5 := checksum.HashMD5(buff[r : r+blockSize])                        // 16 match 说明很大概率要核对md5 所以提前计算md5
					for i := index; i < len(remoteCS) && remoteCS[i].CS16 == fcs.CS16; i++ { // find during 16 match
						if remoteCS[i].CS64 == fcs.CS64 && nowMD5 == remoteCS[i].CS128 { // check roll && md5
							match = true
							matchBlockIndex = int(remoteCS[i].BlockIndex)
							break
						}
					}
				}

				if match {
					if len(failedDataBuff) != 0 {
						err = mismatchWriter(dataIndex, failedDataBuff, iw)
						if err != nil {
							return
						}
						dataIndex += int64(len(failedDataBuff))
						failedDataBuff = failedDataBuff[:0] // clean
					}
					if debug {
						fmt.Println(dataIndex, matchBlockIndex, fcs.CS16)
					}
					err = matchWriter(dataIndex, matchBlockIndex, iw)
					//writeCount += 1
					if err != nil {
						return
					}
					r += blockSize
					dataIndex += int64(blockSize)
				} else {
					// mis match
					failedDataBuff = append(failedDataBuff, buff[r])
					if len(failedDataBuff) >= blockSize*256 { // 超过一定容限
						err = mismatchWriter(dataIndex, failedDataBuff, iw)
						//writeCount += 1

						if err != nil {
							return
						}
						dataIndex += int64(len(failedDataBuff))
						failedDataBuff = failedDataBuff[:0] // clean
					}
					r += 1
				}
			}

			if !finished {
				continue // read data
			} else {
				// 最后一段数据
				var match bool
				var matchBlockIndex int
				fcs := checksum.ProduceFileCSInfoFast(buff[r:w], 0)
				if index, ok := hash16Map[fcs.CS16]; ok { // 16 match
					nowMD5 := checksum.HashMD5(buff[r:w])                                    // 16 match 说明很大概率要核对md5 所以提前计算md5
					for i := index; i < len(remoteCS) && remoteCS[i].CS16 == fcs.CS16; i++ { // find during 16 match
						if remoteCS[i].CS64 == fcs.CS64 && nowMD5 == remoteCS[i].CS128 { // check roll && md5
							match = true
							matchBlockIndex = int(remoteCS[i].BlockIndex)
							break
						}
					}
				}

				if match {
					if debug {
						fmt.Println(dataIndex, matchBlockIndex, fcs.CS16)
					}
					err = matchWriter(dataIndex, matchBlockIndex, iw)
					//writeCount += 1

					if err != nil {
						return
					}
					r += blockSize
					dataIndex += int64(blockSize)
				} else {
					// 最后一段数据, 如果不匹配则立即全部发出, 因为后面也不会匹配了
					//failedDataBuff = append(failedDataBuff, buff[r])
					err = mismatchWriter(dataIndex, failedDataBuff, iw) // 先write 之前残留的failedData
					if err != nil {
						return
					}
					dataIndex += int64(len(failedDataBuff))
					failedDataBuff = failedDataBuff[:0]            // clean
					err = mismatchWriter(dataIndex, buff[r:w], iw) // 将文件最后所有内容全部write
					if err != nil {
						return
					}
					dataIndex += int64(w - r)
					//writeCount += 1
					//dataIndex += int64(len(failedDataBuff))

					r += w - r
				}
			}
		}
	}
	localFileMD5Str := hex.EncodeToString(localFileMD5.Sum(nil))
	fmt.Println(localFileMD5Str)
	//fmt.Println(writeCount)
	err = localFileChecksumWrite(localFileMD5Str, iw)
	return err
}
