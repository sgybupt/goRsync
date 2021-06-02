package signalRPC

import (
	"net/http"
	"net/rpc"
)

func StartRPCServer(addr string) (err error) {
	recv := new(Receiver)
	err = rpc.Register(recv)
	if err != nil {
		return err
	}
	rpc.HandleHTTP()
	if err := http.ListenAndServe(addr, nil); err != nil {
		return err
	}
	return err
}
