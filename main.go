package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"goSync/reveiver"
	"goSync/sender"
	"io"
	"log"
	"net"
	"os"
	"time"
)

const blockSize = 4096
const clientFilePath = "/Users/su/Downloads/B.json"
const serverFilePath = "/Users/su/Downloads/A.json"

func main() {
	fCSInfo, err := reveiver.GetFileAllChecksum(serverFilePath, blockSize)
	if err != nil {
		panic(err)
	}

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
		// buffChan.Flush()  // DO NOT FORGET TO FLUSH
		err = sender.Checker(clientFilePath, blockSize, fCSInfo, buffChan)
		if err != nil {
			log.Fatal(err)
		}
	}()
	conn, err := net.Dial("tcp", "127.0.0.1:10023")
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	startTime := time.Now()
	err = reveiver.ParseMsgsData(serverFilePath, blockSize, conn)
	if err != nil {
		panic(err)
	}
	fmt.Println(time.Since(startTime))

	fClient, _ := os.OpenFile(clientFilePath, os.O_RDONLY, os.ModePerm)
	fClientChecksum := md5.New()
	block := make([]byte, 4096)
	for {
		n, err := fClient.Read(block)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				panic(err)
			}
		}
		fClientChecksum.Write(block[:n])
	}
	fmt.Println("client file checksum:", hex.EncodeToString(fClientChecksum.Sum(nil)))

	fServerNew, _ := os.OpenFile(serverFilePath+".new", os.O_RDONLY, os.ModePerm)
	fServerNewChecksum := md5.New()
	for {
		n, err := fServerNew.Read(block)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				panic(err)
			}
		}
		fServerNewChecksum.Write(block[:n])
	}
	fmt.Println("server new file checksum:", hex.EncodeToString(fServerNewChecksum.Sum(nil)))
}
