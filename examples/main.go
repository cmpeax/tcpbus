package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cmpeax/tcpbus/busnet"
)

func main() {
	// 启动服务器
	go func() {
		server := busnet.NewServer(10329)
		server.AddRouter(MESSAGE_REQUEST_STATE_MESSAGE, &MsgRequestState{})
		server.Start()
		for {
			<-time.After(2 * time.Second)
			fmt.Println("群发通知信息.")
			server.Broadcast(busnet.NewMsgPackage(MESSAGE_BROADCAST_DOEVENT, []byte("干活啦")))
		}
	}()

	// 启动客户端
	for _, v := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		fmt.Println("启动了N个客户端", v)
		go func() {
			<-time.After(2 * time.Second)
			client := busnet.NewClient("127.0.0.1:10329")
			err := client.Connect()
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			client.AddRouter(MESSAGE_BROADCAST_DOEVENT, &MsgBroadCast{})
			fmt.Println("客户端和服务端已经建立连接!")
			// 间歇性向服务器请求
			for {
				<-time.After(2 * time.Second)
				ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
				response, err := client.Request(ctx,
					busnet.NewMsgPackage(MESSAGE_REQUEST_STATE_MESSAGE, []byte("请求")),
					MESSAGE_RESPONSE_STATE_MESSAGE)
				if err != nil {
					fmt.Println(err.Error())
				}
				fmt.Println(string(response.GetData()))
			}
		}()
	}

	for {
		<-time.After(1 * time.Second)
		fmt.Println("倒计时执行中.")
	}
}
