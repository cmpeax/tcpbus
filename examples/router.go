package main

import (
	"fmt"

	"github.com/cmpeax/tcpbus/busface"
	"github.com/cmpeax/tcpbus/busnet"
)

type MsgRequestState struct{}

func (this *MsgRequestState) Handle(request busface.IRequest) {
	fmt.Println("收到服务端的请求:", string(request.GetData()))
	request.GetConnection().Write(
		busnet.NewMsgPackage(MESSAGE_RESPONSE_STATE_MESSAGE,
			[]byte("OK!")))
}

type MsgBroadCast struct{}

func (this *MsgBroadCast) Handle(request busface.IRequest) {
	fmt.Println("收到服务端的广播信息:", string(request.GetData()))

}
