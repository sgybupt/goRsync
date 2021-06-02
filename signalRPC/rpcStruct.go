package signalRPC

import (
	"net"
	"os"
	"time"
)

type BasicFileInfo struct {
	LastModifyTime  time.Time
	Size            int64
	LocalPathPrefix *string
	FullFilePath    *string
	FileStat        os.FileMode // uint32  //all system has isDirMod, in linux it may has executable
}

type LocalFileInfo struct {
	Basic BasicFileInfo
}

type RemoteFileInfo struct {
	Exist bool
	Basic BasicFileInfo
}

type TCPConnection struct {
	Listener   net.Listener
	Connection net.Conn
}
