package main

import (
	"fmt"
	"goSync/reveiver"
	"goSync/sender"
	"log"
	"net"
	"time"
)

func main() {
	fCSInfo, err := reveiver.GetFileAllChecksum("/Users/su/ftp_test/190321153853126488.mp4", 8192)
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
	startTime := time.Now()
	err = reveiver.ParseMsgsData("/Users/su/ftp_test/190321153853126488.mp4", 8192, conn)
	if err != nil {
		panic(err)
	}
	fmt.Println(time.Since(startTime))
}
