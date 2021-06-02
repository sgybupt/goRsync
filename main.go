package main

import (
	"fmt"
	"goSync/reveiver"
	"goSync/sender"
	"goSync/signalRPC"
	"goSync/utilsFunc"
	"log"
	"net"
	"net/rpc"
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

	// rpc server start
	go func() {
		err = signalRPC.StartRPCServer()
		if err != nil {
			log.Fatalln(err)
		}
	}()
	time.Sleep(time.Second)
	// rpc Client
	client, err := rpc.DialHTTP("tcp", "127.0.0.1:8081")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	args1 := signalRPC.LocalFileInfo{
		Basic: signalRPC.BasicFileInfo{
			LastModifyTime:  time.Time{},
			Size:            0,
			LocalPathPrefix: utilsFunc.StrPtr("/Users/su/Downloads"),
			FullFilePath:    utilsFunc.StrPtr("/Users/su/Downloads/B.json"),
			FileStat:        0,
		},
	}
	var reply1 signalRPC.RemoteFileInfo
	err = client.Call("Receiver.FileInfo", &args1, &reply1)
	if err != nil {
		log.Fatal("FileInfo error:", err)
	}
	fmt.Println(reply1)

	var args2 bool
	var reply2 int
	err = client.Call("Receiver.OpenDataPort", &args2, &reply2)
	if err != nil {
		log.Fatal("OpenDataPort error:", err)
	}
	fmt.Println(reply2)

	conn, err = net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", reply2))
	if err != nil {
		log.Fatalln(err)
	}

	_, err = conn.Write([]byte{1, 2, 3, 4})
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Second)
	_ = conn.Close()

	args3 := reply2
	var reply3 bool
	err = client.Call("Receiver.CloseDataPort", &args3, &reply3)
	if err != nil {
		log.Fatal("CloseDataPort error:", err)
	}
	fmt.Println(reply3)

}
