package main

import (
	"fmt"
	"goSync/reveiver"
	"goSync/sender"
	"goSync/signalRPC"
	"goSync/utilsFunc"
	"log"
	"net"
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

	// 本地测试检查
	m, err := utilsFunc.GetFileMD5(clientFilePath)
	if err != nil {
		panic(err)
	}
	fmt.Println("client file checksum", m)

	m, err = utilsFunc.GetFileMD5(serverFilePath + ".new")
	if err != nil {
		panic(err)
	}
	fmt.Println("server new file checksum", m)

	err = signalRPC.StartRPCServer()
	if err != nil {
		log.Fatalln(err)
	}

}
