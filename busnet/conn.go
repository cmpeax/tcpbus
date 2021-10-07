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

// tcp链接
type TcpConn struct {
	sync.RWMutex
	log        busface.ILog
	conn       *net.TCPConn
	connID     uint32
	addr       string
	ctx        context.Context
	cancel     context.CancelFunc
	server     busface.IServer
	writerChan chan []byte
}

func NewTcpConn(ctx context.Context, log busface.ILog, server busface.IServer, connID uint32, conn *net.TCPConn, addr string) *TcpConn {
	connCtx, cancel := context.WithCancel(ctx)
	return &TcpConn{
		log:        log,
		conn:       conn,
		addr:       addr,
		connID:     connID,
		ctx:        connCtx,
		cancel:     cancel,
		server:     server,
		writerChan: make(chan []byte, 50),
	}
}

func (this *TcpConn) Start() {
	go this.StartReader()
	go this.StartWriter()
}

func (this *TcpConn) GetContext() context.Context {
	return this.ctx
}

func (this *TcpConn) GetConnID() uint32 {
	return this.connID
}

func (this *TcpConn) StartReader() {
	defer func() {
		this.log.Write(busface.LOG_LEVEL_DEBUG, fmt.Sprintf("[服务端][%s] 读协程 已被释放.", this.addr))
	}()
	for {
		select {
		case <-this.ctx.Done():
			return
		default:
			//读取包头
			headData := make([]byte, this.server.GetPackFunc().GetHeadLen())
			if _, err := io.ReadFull(this.conn, headData); err != nil {
				this.log.Write(busface.LOG_LEVEL_ERROR, fmt.Sprintf("[服务端][%s] 错误: 读取头部失败. [%s]", this.addr, err.Error()))
				this.Close()
				return
			}

			//拆包，得到msgID 和 datalen 放在msg中
			msg, err := this.server.GetPackFunc().Unpack(headData)
			if err != nil {
				this.log.Write(busface.LOG_LEVEL_ERROR, fmt.Sprintf("[服务端][%s] 错误: 解包失败. [%s]", this.addr, err.Error()))
				this.Close()
				return
			}

			//根据 dataLen 读取 data，放在msg.Data中
			var data []byte
			if msg.GetDataLen() > 0 {
				data = make([]byte, msg.GetDataLen())
				if _, err := io.ReadFull(this.conn, data); err != nil {
					this.log.Write(busface.LOG_LEVEL_ERROR, fmt.Sprintf("[服务端][%s] 错误: 读数据没有达到指定的长度. [%s]", this.addr, err.Error()))
					this.Close()
					return
				}
			}
			msg.SetData(data)

			//得到当前客户端请求的Request数据
			req := &Request{
				conn: this,
				msg:  msg,
			}

			this.server.GetHandler().SendMsgToTaskQueue(req)

		}
	}
}

func (this *TcpConn) StartWriter() {
	defer func() {
		this.log.Write(busface.LOG_LEVEL_DEBUG, fmt.Sprintf("[服务端][%s] 写协程 已被释放.", this.addr))
	}()

	for {
		select {
		case data := <-this.writerChan:
			_, err := this.conn.Write(data)
			if err != nil {
				this.log.Write(busface.LOG_LEVEL_ERROR, fmt.Sprintf("[服务端][%s] 发送协程错误: [%s]", this.addr, err.Error()))
				this.Close()
				return
			}
		case <-this.ctx.Done():
			return
		}
	}
}

func (this *TcpConn) Write(data busface.IMessage) error {

	//将data封包，并且发送
	dp := this.server.GetPackFunc()
	msg, err := dp.Pack(data)
	if err != nil {
		return errors.New("Pack error msg ")
	}

	//写回客户端
	this.writerChan <- msg

	return nil
}

func (this *TcpConn) Close() {
	this.cancel()
	this.conn.Close()
}
