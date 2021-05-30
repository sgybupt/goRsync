package main

import (
	"errors"
	"fmt"
	"goSync/reveiver"
	"goSync/sender"
	"io"
	"log"
	"net"
	"sort"
	"time"
)

type BuffChunk struct {
	buf     []byte
	maxSize int64
	r, w    int //[r, w)
}

func (t *BuffChunk) Init(size int64) {
	t.buf = make([]byte, 0, size)
	t.maxSize = size
}

func (t *BuffChunk) Read(p []byte) (n int, err error) {
	c := 0
	pL := len(p)
	for ; c < 10; c++ {
		if t.w == t.r {
			time.Sleep(time.Millisecond * 100)
			continue
		}
		if t.w-t.r >= pL {
			copy(p, t.buf[t.r:t.r+pL])
			t.r += pL
			return pL, nil
		} else {
			copy(p, t.buf[t.r:t.w])
			t.r += t.w - t.r
			return t.w - t.r, nil
		}
	}
	return 0, io.EOF
}

func (t *BuffChunk) Write(p []byte) (n int, err error) {
	if int64(len(p)+t.w) >= t.maxSize {
		fmt.Println(t.buf)
		return 0, errors.New("full filled")
	}
	t.buf = append(t.buf, p...)
	t.w += len(p)
	return len(p), nil
}

func main() {
	fCSInfo, err := reveiver.GetFileAllChecksum("/Users/su/ftp_test/190321153853126488.mp4", 8192)
	if err != nil {
		panic(err)
	}

	sort.Slice(fCSInfo, func(i, j int) bool {
		return fCSInfo[i].CS16 < fCSInfo[j].CS16
	})

	listen, err := net.Listen("tcp", "127.0.0.1:10023")
	if err != nil {
		log.Fatalln(err)
	}
	defer listen.Close()

	go func() {
		buffChan, err := listen.Accept()
		if err != nil {
			log.Fatalln(err)
		}
		defer buffChan.Close()

		//var buff bytes.Buffer
		//buffChan := bufio.NewReadWriter(bufio.NewReader(&buff), bufio.NewWriter(&buff))
		//var buff BuffChunk
		//buff.Init(math.MaxInt32)
		//buffChan := bufio.NewReadWriter(bufio.NewReader(&buff), bufio.NewWriter(&buff))

		err = sender.Checker("/Users/su/ftp_test/190321153853126488.mp4", 8192, fCSInfo, buffChan)
		if err != nil {
			log.Fatal(err)
		}
	}()
	conn, err := net.Dial("tcp", "127.0.0.1:10023")
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	reveiver.ParseMsgsData(conn)
}
