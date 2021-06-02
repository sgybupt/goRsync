package signalRPC

import (
	"errors"
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
	if localFileInfo.Basic.LocalPathPrefix == nil || localFileInfo.Basic.FullFilePath == nil {
		return errors.New("LocalPathPrefix and FullFilePath is needed")
	}
	pureFilePath := strings.TrimPrefix(path.Clean(*localFileInfo.Basic.FullFilePath),
		path.Clean(*localFileInfo.Basic.LocalPathPrefix))
	pureFilePath = path.Clean(pureFilePath)
	// file not exist
	localFileFullPath := path.Join(BasicConfig.LocalPathPrefix, pureFilePath)
	if !utilsFunc.CheckFileIsExist(localFileFullPath) {
		remoteFileInfo.Exist = false
		return nil
	}
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
		randPort = rand.Intn(BasicConfig.PortRangeH-BasicConfig.PortRangeL) + BasicConfig.PortRangeL
		if _, ok := Connections.Load(randPort); ok {
			continue
		}
		listener, err := net.ListenTCP("tcp", &net.TCPAddr{
			IP:   net.IPv4('0', '0', '0', '0'),
			Port: randPort,
		})
		if err != nil {
			continue
		}
		go func() {
			conn, err := listener.AcceptTCP()
			if err != nil {
				_ = listener.Close()
			}
			Connections.Store(randPort, &TCPConnection{
				Listener:   listener,
				Connection: conn,
			})
		}()
		break
	}
	*port = randPort
	return nil
}

func (r *Receiver) CloseDataPort(port *int, reply *int) error {
	tcpConnInter, ok := Connections.Load(*port)
	if !ok {
		return nil
	}
	tcpConn, ok := tcpConnInter.(*TCPConnection)
	if !ok {
		log.Fatal("load tcpConnection failed")
	}

	var err error
	err = tcpConn.Connection.Close()
	if err != nil {
		return err
	}
	err = tcpConn.Listener.Close()
	return err
}
