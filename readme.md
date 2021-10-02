## 支持 事件订阅/请求回应 的的TCP通讯框架
---
源于 zinx 的业务定制版本 TCP 通讯框架

---
#### 特点
* 服务端/客户端均支持订阅事件通讯.
* 服务端支持 Broadcast 方式发送信息到客户端.
* 客户端均支持 Request/Reply 方式与服务端通讯
* 服务端用多个worker去处理客户端的请求
 

#### 目前问题
* 未写单元测试
* Request的 router 方式 应该用 Eventbus 而不是 map (后续改进)
* 实际上解包方法可以定制实现,为了方便自己项目使用,故NewServer没有开放函数.后续可以通过追加SetPack(busface.IPack)的方法去传入.
* 服务端和客户端均未做底层的心跳包以及断线重连的机制.
---

服务端例子
```go

package main

// 定义 消息ID 
const MESSAGE_REQUEST_STATE_MESSAGE uint32 = 1
const MESSAGE_RESPONSE_STATE_MESSAGE uint32 = 2

const MESSAGE_BROADCAST_DOEVENT uint32 = 3

type MsgRequestState struct{}

func (this *MsgRequestState) Handle(request busface.IRequest) {
    // 提取服务端发送的信息
    fmt.Println("收到服务端的请求:", string(request.GetData()))
    // 返回给客户端.
    request.GetConnection().Write(
        busnet.NewMsgPackage(MESSAGE_RESPONSE_STATE_MESSAGE,
	    []byte("OK!")))
}


func main() {
    // 绑定端口号
    server := busnet.NewServer(10329) 
    // 设置路由
    server.AddRouter(MESSAGE_REQUEST_STATE_MESSAGE, 
        &MsgRequestState{})
    server.Start()

    for {
        <-time.After(2 * time.Second)
        // 群发一次信息到所有客户端
        server.Broadcast(
            busnet.NewMsgPackage(MESSAGE_BROADCAST_DOEVENT, 
            []byte("群发的信息")))
    }
}

```

客户端例子
```go
package main

// 定义 消息ID 
const MESSAGE_REQUEST_STATE_MESSAGE uint32 = 1
const MESSAGE_RESPONSE_STATE_MESSAGE uint32 = 2

const MESSAGE_BROADCAST_DOEVENT uint32 = 3

type MsgBroadCast struct{}

func (this *MsgBroadCast) Handle(request busface.IRequest) {
    fmt.Println("收到服务端的广播信息:", string(request.GetData()))
}


func main() {
    client := busnet.NewClient("127.0.0.1:10329")
    err := client.Connect()
    if err != nil {
        fmt.Println(err.Error())
        return
    }
    
    // 设置路由
    client.AddRouter(MESSAGE_BROADCAST_DOEVENT, 
        &MsgBroadCast{})
    fmt.Println("客户端和服务端已经建立连接!")
    // 间歇性向服务器请求
    for {
        <-time.After(2 * time.Second)
        // 设置 超时context
        ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
        // 参数如下: context, 请求参数, 期待接收的参数
        // 会临时创建一个 router ,超时或者得到期待应答后 delete router.
        // 后续可以用 eventbus去实现,目前用的是 map实现, 不支持一个信息多个应答.
        response, err := client.Request(ctx,
            busnet.NewMsgPackage(MESSAGE_REQUEST_STATE_MESSAGE, []byte("请求")),
            MESSAGE_RESPONSE_STATE_MESSAGE)
        if err != nil {
            fmt.Println(err.Error())
        }
        // 输出应答信息.
        fmt.Println(string(response.GetData()))
    }
}
```