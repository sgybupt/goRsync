package utilsFunc

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

func CheckFileIsExist(filename string) bool {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false
	}
	return true
}

func GetFileMD5(fp string) (m string, err error) {
	f, err := os.OpenFile(fp, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return m, err
	}
	fChecksum := md5.New()
	block := make([]byte, 4096)
	for {
		n, err := f.Read(block)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return m, err
			}
		}
		fChecksum.Write(block[:n])
	}
	return hex.EncodeToString(fChecksum.Sum(nil)), nil
}
