package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cmpeax/tcpbus/busnet"
)

type LogObject struct{}

func (this *LogObject) Write(code uint32, msg string) {
	fmt.Println(msg)
}

func main() {
	// 启动服务器
	go func() {
		server := busnet.NewServer(10329, &LogObject{})
		server.AddRouter(MESSAGE_REQUEST_STATE_MESSAGE, &MsgRequestState{})
		server.Start()
		for {
			<-time.After(2 * time.Second)
			server.Broadcast(busnet.NewMsgPackage(MESSAGE_BROADCAST_DOEVENT, []byte("干活啦")))
		}
	}()

	// 启动客户端
	for _, v := range []int{1, 2, 3, 4, 5, 6, 7, 8} {
		fmt.Println("启动了N个客户端", v)
		go func() {
			<-time.After(2 * time.Second)
			client := busnet.NewClient("127.0.0.1:10329", &LogObject{})
			client.AddRouter(MESSAGE_BROADCAST_DOEVENT, &MsgBroadCast{})
			client.Start()

			// 请求一次
			<-time.After(2 * time.Second)
			ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
			response, err := client.Request(ctx,
				busnet.NewMsgPackage(MESSAGE_REQUEST_STATE_MESSAGE, []byte("请求")),
				MESSAGE_RESPONSE_STATE_MESSAGE)
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Println(string(response.GetData()))

			// 退出客户端
			<-time.After(2 * time.Second)
			client.Close()
		}()
	}

	for {
		<-time.After(1 * time.Second)
	}
}
