package reveiver

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"goSync/checksum"
	"goSync/structs"
	"io"
	"os"
)

var debug = true

func GetFileAllChecksum(p string, blockSize int) (fcsl []structs.FileCSInfo, err error) {
	f, err := os.OpenFile(p, os.O_RDONLY, 0664)
	if err != nil {
		return fcsl, err
	}
	defer f.Close()

	fs, err := f.Stat()
	if err != nil {
		return fcsl, err
	}
	fileSize := fs.Size()
	blockCount := fileSize / int64(blockSize)
	if fileSize%int64(blockSize) != 0 {
		blockCount += 1
	}
	fmt.Println("file size ", fileSize)
	fmt.Println("block count ", blockCount)

	fcsl = make([]structs.FileCSInfo, 0, blockCount)

	var blockIndex int64
	bs := make([]byte, blockSize)
	bsTmp := make([]byte, 0, blockSize)
	for ; ; {
		n, err := f.Read(bs)
		if err != nil {
			if err == io.EOF { // end of file
				fcsl = append(fcsl, checksum.ProduceFileCSInfo(bsTmp, blockIndex))
				break
			} else {
				return fcsl, err
			}
		}
		if n < blockSize {
			if len(bsTmp)+n < blockSize { // bsBuffer
				bsTmp = append(bsTmp, bs[:n]...)
				continue
			} else {
				bsTmp = append(bsTmp, bs[:blockSize-len(bsTmp)]...)
				fcsl = append(fcsl, checksum.ProduceFileCSInfo(bsTmp, blockIndex))
				blockIndex += 1

				bsTmp = bsTmp[:0]
				bsTmp = append(bsTmp, bs[blockSize-len(bsTmp):n]...)
			}
		} else {
			fcsl = append(fcsl, checksum.ProduceFileCSInfo(bs, blockIndex))
			blockIndex += 1
		}
	}
	return fcsl, nil
}

/*
// c, offset, chunkIndex, \n
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
*/

func ParseMsgsData(ir io.Reader) {
	f, err := os.OpenFile("/Users/su/ftp_test/190321153853126488.mp4", os.O_RDONLY, 0664)
	if err != nil {
		panic(nil)
	}
	defer f.Close()

	chunk := make([]byte, 8192)
	newFileMd5 := md5.New()

	msgReader := bufio.NewReader(ir)
	head := make([]byte, 1+4)
	cContent := make([]byte, 8+4)
	mContent := make([]byte, 16)
	var cOffset int64
	var cChunkIndex int
	//var readMsgCount int
	for ; ; {
		n, err := msgReader.Read(head)
		if err != nil {
			if err == io.EOF {
				fmt.Println("EOF")
				break
			}
			panic(err)
		}
		if n != 1+4 {
			panic(errors.New("fail read head"))
		}

		h, dataLen := head[0], binary.LittleEndian.Uint32(head[1:5]) // content length

		switch h {
		case 'c':
			cbCount := 0
			for ; cbCount != 8+4; {
				cn, err := msgReader.Read(cContent[cbCount:])
				if err != nil {
					if err == io.EOF {
						panic(errors.New("protocol wrong"))
					} else {
						panic(err)
					}
				}
				cbCount += cn
			}
			cOffset = int64(binary.LittleEndian.Uint64(cContent[0:8]))
			cChunkIndex = int(binary.LittleEndian.Uint32(cContent[8 : 8+4]))
			n, err := f.ReadAt(chunk, int64(cChunkIndex)*8192)
			if err != nil {
				if err == io.EOF {
					err = nil
				} else {
					panic(err)
				}
			}
			if n != 8192 {
				fmt.Println(n)
			}
			newFileMd5.Write(chunk[:n])
			if debug {
				fmt.Println(cOffset, cChunkIndex)
			}
			//readMsgCount += 1
		case 'b':
			bBytesCount := 0
			bContent := make([]byte, dataLen)
			for ; bBytesCount != int(dataLen); {
				bn, err := msgReader.Read(bContent[bBytesCount:])
				if err != nil {
					if err == io.EOF {
						panic(errors.New("protocol wrong"))
					} else {
						panic(err)
					}
				}
				bBytesCount += bn
			}
			offset := int64(binary.LittleEndian.Uint64(bContent[0:8]))
			rawData := bContent[8:]
			newFileMd5.Write(rawData)
			if debug || true {
				fmt.Println("offset: ", offset, "data: ", dataLen-8)
			}
		case 'm':
			mbCount := 0
			for ; mbCount != 16; {
				mn, err := msgReader.Read(mContent[mbCount:])
				if err != nil {
					if err == io.EOF {
						fmt.Println("EOF")
						goto final

					} else {
						panic(err)
					}
				}
				mbCount += mn
			}
			fmt.Println("final file checksum expected:", hex.EncodeToString(mContent))
			goto final
		default:
			panic(errors.New("wrong msg"))
		}
	}
final:
	fmt.Println("new file checksum", hex.EncodeToString(newFileMd5.Sum(nil)))
	//fmt.Println(readMsgCount)
}
