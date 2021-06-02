package main

import (
	"bufio"
	"fmt"
	"goSync/receiver"
	"goSync/sender"
	"goSync/signalRPC"
	"goSync/structs"
	"goSync/utilsFunc"
	"log"
	"net"
	"net/rpc"
	"time"
)

const blockSize = 4096
const clientFilePath = "/Users/su/ftp_test/client/A.json"
const serverFilePath = "/Users/su/ftp_test/server/A.json"

func coreTest() {
	// cores test
	fCSInfo, err := receiver.GetFileAllChecksum(serverFilePath, blockSize)
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
	err = receiver.ParseMsgsData(serverFilePath, blockSize, conn)
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

	m, err = utilsFunc.GetFileMD5(serverFilePath)
	if err != nil {
		panic(err)
	}
	fmt.Println("server new file checksum", m)
}

func rsyncServerStart() {
	// rpc server start
	go func() {
		err := signalRPC.StartRPCServer()
		if err != nil {
			log.Fatalln(err)
		}
	}()
	time.Sleep(time.Second)
}

func rsyncClientStart() {
	// rpc Client
	client, err := rpc.DialHTTP("tcp", "127.0.0.1:8081")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	localFileInfo := signalRPC.LocalFileInfo{
		Basic: signalRPC.BasicFileInfo{
			LastModifyTime:  time.Time{},
			Size:            0,
			LocalPathPrefix: utilsFunc.StrPtr("/Users/su/ftp_test/client"),
			FullFilePath:    utilsFunc.StrPtr("/Users/su/ftp_test/client/A.json"),
			FileStat:        0,
		},
	}
	var remoteFileInfo signalRPC.RemoteFileInfo
	err = client.Call("Receiver.FileInfo", &localFileInfo, &remoteFileInfo)
	if err != nil {
		log.Fatal("FileInfo error:", err)
	}
	fmt.Println(remoteFileInfo)

	remoteFileChecksum := make([]structs.FileCSInfo, 0)
	err = client.Call("Receiver.GetFileChecksum",
		&signalRPC.GetFileChecksumInfo{
			RemoteFileInfo: remoteFileInfo,
			BlockSize:      blockSize,
		},
		&remoteFileChecksum)
	if err != nil {
		log.Fatal("FileInfo error:", err)
	}
	//fmt.Println(remoteFileChecksum)

	var remoteOpenPort int
	err = client.Call("Receiver.OpenDataPort", &signalRPC.GetFileChecksumInfo{
		RemoteFileInfo: remoteFileInfo,
		BlockSize:      blockSize,
	}, &remoteOpenPort)
	if err != nil {
		log.Fatal("OpenDataPort error:", err)
	}
	fmt.Println(remoteOpenPort)

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", remoteOpenPort))
	if err != nil {
		log.Fatalln(err)
	}

	writerBuffer := bufio.NewWriterSize(conn, blockSize*256)
	err = sender.Checker(clientFilePath, blockSize, remoteFileChecksum, writerBuffer)
	if err != nil {
		log.Fatal(err)
	}
	err = writerBuffer.Flush()
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Second)
	_ = conn.Close()

	closePort := remoteOpenPort
	var reply3 bool
	err = client.Call("Receiver.CloseDataPort", &closePort, &reply3)
	if err != nil {
		log.Fatal("CloseDataPort error:", err)
	}
	fmt.Println(reply3)
}

func main() {
	//coreTest()
	rsyncServerStart()
	rsyncClientStart()
}
