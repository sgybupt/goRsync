package signalRPC

import (
	"errors"
	"fmt"
	"goSync/BasicConfig"
	"goSync/utilsFunc"
	"log"
	"math/rand"
	"net"
	"os"
	"path"
	"strings"
	"time"
)

type Receiver int

func (r *Receiver) FileInfo(localFileInfo *LocalFileInfo, remoteFileInfo *RemoteFileInfo) error {
	fmt.Println(*localFileInfo)
	if localFileInfo.Basic.LocalPathPrefix == nil || localFileInfo.Basic.FullFilePath == nil {
		return errors.New("LocalPathPrefix and FullFilePath is needed")
	}
	pureFilePath := strings.TrimPrefix(path.Clean(*localFileInfo.Basic.FullFilePath),
		path.Clean(*localFileInfo.Basic.LocalPathPrefix))
	pureFilePath = path.Clean(pureFilePath)
	// file not exist
	localFileFullPath := path.Join(BasicConfig.LocalPathPrefix, pureFilePath)
	fmt.Println(localFileFullPath)
	if !utilsFunc.CheckFileIsExist(localFileFullPath) {
		remoteFileInfo.Exist = false
		return nil
	}
	remoteFileInfo.Exist = true
	f, err := os.OpenFile(localFileFullPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return nil
	}
	fStat, err := f.Stat()
	if err != nil {
		return nil
	}
	basicFileInfo := BasicFileInfo{
		LastModifyTime:  fStat.ModTime(),
		Size:            fStat.Size(),
		LocalPathPrefix: &BasicConfig.LocalPathPrefix,
		FullFilePath:    &localFileFullPath,
		FileStat:        fStat.Mode(),
	}
	remoteFileInfo.Basic = basicFileInfo
	return nil
}

func (r *Receiver) OpenDataPort(arg *bool, port *int) error {
	rand.Seed(time.Now().UnixNano())
	var randPort int
	for {
		fmt.Println("selecting port")
		randPort = rand.Intn(BasicConfig.PortRangeH-BasicConfig.PortRangeL) + BasicConfig.PortRangeL
		if _, ok := Connections.Load(randPort); ok {
			continue
		}
		listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", randPort))
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println("open port:", randPort)
		go func() {
			Connections.Store(randPort, &TCPConnection{
				Listener: listener,
			})
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("[Warning]: accept error:", err)
				Connections.Delete(randPort)
				_ = listener.Close()
			} else {
				fmt.Println("client is dial success:", conn.RemoteAddr())
			}
			if err == nil {
				Connections.Store(randPort, &TCPConnection{
					Listener:   listener,
					Connection: conn,
				})
			}
		}()
		break
	}
	*port = randPort
	return nil
}

func (r *Receiver) CloseDataPort(port *int, reply *bool) error {
	tcpConnInter, ok := Connections.Load(*port)
	if !ok {
		log.Println("[Info]: this port is not opened:", *port)
		return nil
	}
	fmt.Println(*port)
	tcpConn, ok := tcpConnInter.(*TCPConnection)
	if !ok {
		log.Fatal("load tcpConnection failed")
	}
	Connections.Delete(*port)
	var err error
	if tcpConn.Connection != nil {
		err = tcpConn.Connection.Close()
		if err != nil {
			return err
		}
	}

	if tcpConn.Listener != nil {
		err = tcpConn.Listener.Close()
	}
	*reply = true
	fmt.Println("close port", *port)
	return err
}
