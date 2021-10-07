package busnet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/cmpeax/tcpbus/busface"
)

// 客户端
type TcpBusClient struct {
	ctx        context.Context
	cancel     context.CancelFunc
	conn       net.Conn
	addr       string
	handler    busface.IClientHandler
	pack       busface.IPack
	writerChan chan []byte
}

func NewClient(addr string) *TcpBusClient {
	newCtx, cancel := context.WithCancel(context.Background())
	return &TcpBusClient{
		ctx:        newCtx,
		cancel:     cancel,
		addr:       addr,
		pack:       NewDataPack(),
		handler:    NewClientHandle(),
		writerChan: make(chan []byte, 50),
	}
}

func (this *TcpBusClient) AddRouter(msgID uint32, router busface.IRouter) {
	this.handler.AddRouter(msgID, router)
}

func (this *TcpBusClient) RemoveRouter(msgID uint32) {
	this.handler.RemoveRouter(msgID)
}

func (this *TcpBusClient) Close() {
	this.cancel()
}

func (this *TcpBusClient) Connect() error {
	conn, err := net.Dial("tcp", this.addr)
	if err != nil {
		return err
	}
	this.conn = conn
	go this.startReader()
	go this.startWriter()
	// 建立连接后,启动线程.
	return nil
}

func (this *TcpBusClient) startReader() {
	for {
		select {
		case <-this.ctx.Done():
			return
		default:
			//读取包头
			headData := make([]byte, this.GetPackFunc().GetHeadLen())
			if _, err := io.ReadFull(this.conn, headData); err != nil {
				fmt.Println("read msg head error ", err)
				return
			}

			//拆包，得到msgID 和 datalen 放在msg中
			msg, err := this.GetPackFunc().Unpack(headData)
			if err != nil {
				fmt.Println("unpack error ", err)
				return
			}

			//根据 dataLen 读取 data，放在msg.Data中
			var data []byte
			if msg.GetDataLen() > 0 {
				data = make([]byte, msg.GetDataLen())
				if _, err := io.ReadFull(this.conn, data); err != nil {
					fmt.Println("read msg data error ", err)
					return
				}
			}
			msg.SetData(data)

			//得到当前客户端请求的Request数据
			req := &Request{
				conn: this,
				msg:  msg,
			}

			this.GetHandler().DoHandler(req)

		}
	}
}

func (this *TcpBusClient) Write(data busface.IMessage) error {
	//将data封包，并且发送
	dp := this.GetPackFunc()
	msg, err := dp.Pack(data)
	if err != nil {
		return errors.New("Pack error msg ")
	}

	//写回客户端
	this.writerChan <- msg
	return nil
}

type ITempMessage struct {
	wg      sync.WaitGroup
	msgChan chan busface.IMessage
}

func NewITempMessage(messageChan chan busface.IMessage) *ITempMessage {
	return &ITempMessage{
		msgChan: messageChan,
	}
}

func (this *ITempMessage) Handle(request busface.IRequest) {
	this.msgChan <- NewMsgPackage(request.GetMsgID(), request.GetData())
}

// 请求
func (this *TcpBusClient) Request(ctx context.Context, req busface.IMessage, hopeRecvMessage uint32) (busface.IMessage, error) {
	if err := this.Write(req); err != nil {
		return req, err
	}

	messageChan := make(chan busface.IMessage, 0)
	this.AddRouter(hopeRecvMessage, NewITempMessage(messageChan))
	defer this.RemoveRouter(hopeRecvMessage)
	select {
	case <-ctx.Done():
		return req, errors.New("timeout")
	case recv := <-messageChan:
		return recv, nil
	}

	// 等待接收
}

// 底层的写入
func (this *TcpBusClient) startWriter() {
	for {
		select {
		case data := <-this.writerChan:
			_, err := this.conn.Write(data)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		case <-this.ctx.Done():
			return
		}
	}

}

func (this *TcpBusClient) GetHandler() busface.IClientHandler {
	return this.handler
}

func (this *TcpBusClient) GetPackFunc() busface.IPack {
	return this.pack
}

func (this *TcpBusClient) GetConnID() uint32 {
	return 0
}
