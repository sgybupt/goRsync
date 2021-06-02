package signalRPC

import (
	"net/http"
	"net/rpc"
)

func StartRPCServer() (err error) {
	recv := new(Receiver)
	err = rpc.Register(recv)
	if err != nil {
		return err
	}
	rpc.HandleHTTP()
	if err := http.ListenAndServe("0.0.0.0:8081", nil); err != nil {
		return err
	}
	return err
}
