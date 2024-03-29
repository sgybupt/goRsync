package receiver

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"goSync/checksum"
	"goSync/structs"
	"goSync/utilsFunc"
	"io"
	"log"
	"os"
	"time"
)

var debug = false

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
	fileChecksum := md5.New()

	fileReadBuf := bufio.NewReaderSize(f, blockSize*256)
	fcsl = make([]structs.FileCSInfo, 0, blockCount)

	var blockIndex int64
	bs := make([]byte, blockSize)
	bsTmp := make([]byte, 0, blockSize)
	for {
		n, err := fileReadBuf.Read(bs)
		fileChecksum.Write(bs[:n])
		if err != nil {
			if err == io.EOF { // end of file
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
	fmt.Println("server old file checksum:", hex.EncodeToString(fileChecksum.Sum(nil)))
	return fcsl, nil
}

// fp is the local file
func ParseMsgsData(fp string, blockSize int, ir io.Reader) (err error) {
	serverTime := time.Now().UnixNano()
	f, err := os.OpenFile(fp, os.O_RDONLY, 0664)
	if err != nil {
		panic(nil)
	}
	defer f.Close()

	fNew := new(os.File)
	tmpFileName := fp + fmt.Sprintf("~%d", serverTime)
	if utilsFunc.CheckFileIsExist(tmpFileName) {
		_ = os.Remove(tmpFileName)
		fNew, err = os.OpenFile(tmpFileName, os.O_CREATE|os.O_WRONLY, 0666)
	} else {
		fNew, err = os.OpenFile(tmpFileName, os.O_CREATE|os.O_WRONLY, 0666)
	}
	if err != nil {
		return err
	}
	defer fNew.Close()

	chunk := make([]byte, blockSize)
	newFileMd5 := md5.New()

	msgReader := bufio.NewReaderSize(ir, blockSize*256)
	newFileWriter := bufio.NewWriterSize(fNew, blockSize*256)
	defer func() {
		err = newFileWriter.Flush()
		if err != nil {
			log.Println("[Error]: flush data to file failed with err:", err)
		}
	}()
	head := make([]byte, 1+4)
	cContent := make([]byte, 8+4)
	mContent := make([]byte, 16)
	var cOffset int64
	var cChunkIndex int
	//var readMsgCount int
	var expectFileChecksum string
	for {
		n, err := msgReader.Read(head)
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if n != 1+4 {
			return errors.New("fail read head")
		}

		h, dataLen := head[0], binary.LittleEndian.Uint32(head[1:5]) // content length

		switch h {
		case 'c':
			cbCount := 0
			for cbCount != 8+4 {
				cn, err := msgReader.Read(cContent[cbCount:])
				if err != nil {
					if err == io.EOF {
						return errors.New("protocol wrong")
					} else {
						return err
					}
				}
				cbCount += cn
			}
			cOffset = int64(binary.LittleEndian.Uint64(cContent[0:8]))
			cChunkIndex = int(binary.LittleEndian.Uint32(cContent[8 : 8+4]))
			n, err := f.ReadAt(chunk, int64(cChunkIndex)*int64(blockSize))
			if err != nil {
				if err == io.EOF {
					err = nil // 此处error 被忽略, 因为最后一个chunk被读取的时候会返回eof
				} else {
					return nil
				}
			}
			newFileMd5.Write(chunk[:n])
			_, err = newFileWriter.Write(chunk[:n])
			if err != nil {
				fmt.Println("[Error]: write failed with err:", err)
				return err
			}
			if debug {
				fmt.Println(cOffset, cChunkIndex)
			}
			//readMsgCount += 1
		case 'b':
			bBytesCount := 0
			bContent := make([]byte, dataLen)
			for bBytesCount != int(dataLen) {
				bn, err := msgReader.Read(bContent[bBytesCount:])
				if err != nil {
					if err == io.EOF {
						return errors.New("protocol wrong")
					} else {
						return err
					}
				}
				bBytesCount += bn
			}
			offset := int64(binary.LittleEndian.Uint64(bContent[0:8]))
			rawData := bContent[8:]
			newFileMd5.Write(rawData)
			_, err = newFileWriter.Write(rawData)
			if err != nil {
				fmt.Println("[Error]: write failed with err:", err)
				return err
			}
			if debug || true {
				fmt.Println("offset: ", offset, "data: ", dataLen-16)
			}
		case 'm':
			mbCount := 0
			for mbCount != 16 {
				mn, err := msgReader.Read(mContent[mbCount:])
				if err != nil {
					if err == io.EOF {
						fmt.Println("EOF")
						goto final

					} else {
						return err
					}
				}
				mbCount += mn
			}
			expectFileChecksum = hex.EncodeToString(mContent)
			fmt.Println("final file checksum expected:", expectFileChecksum)
			goto final
		default:
			return errors.New("wrong msg")
		}
	}
final:
	newFileChecksum := hex.EncodeToString(newFileMd5.Sum(nil))
	fmt.Println("new file checksum", newFileChecksum)

	if expectFileChecksum != newFileChecksum {
		return errors.New("new file md5 is not expect, some internal error occurs")
	} else {
		_ = f.Close()
		err = os.Rename(tmpFileName, fp)
	}
	return err
}
