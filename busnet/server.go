package busnet

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/cmpeax/tcpbus/busface"
)

// tcp池
type tcpPool struct {
	conns sync.Map
	log   busface.ILog
}

func NewTcpPool(log busface.ILog) *tcpPool {
	return &tcpPool{
		log: log,
	}
}

func (this *tcpPool) Join(id uint32, conn *TcpConn) {
	// 加入时,监听. 以便删除
	this.conns.Store(id, conn)
	go func(cn *TcpConn, tid uint32) {
		<-cn.GetContext().Done()
		// 检测到连接失效时,删除该链接。
		this.log.Write(busface.LOG_LEVEL_INFO, fmt.Sprintf("[服务端][%s] 检测到该连接已被断开.", cn.addr))
		this.conns.Delete(tid)
	}(conn, id)
}

func (this *tcpPool) Broadcast(data busface.IMessage) {
	this.Send("", data)
}

func (this *tcpPool) Send(ip string, data busface.IMessage) {
	this.conns.Range(func(key interface{}, value interface{}) bool {
		matchConn := value.(*TcpConn)
		connAddr := strings.Split(matchConn.addr, ":")

		if ip == "" || (len(connAddr) == 2 && connAddr[0] == ip) {
			err := matchConn.Write(data)
			if err != nil {
				//错误的连接. 将该连接从pool里删除
				this.log.Write(busface.LOG_LEVEL_ERROR, fmt.Sprintf("[服务端][%s] 发送错误.", matchConn.addr))
				matchConn.Close()
			}
		}

		return true
	})
}

// 服务端
type TcpBusServer struct {
	IP       string
	log      busface.ILog
	Port     int
	MaxConn  int
	handler  busface.IHandler
	pack     busface.IPack
	connPool *tcpPool
}

func NewServer(port int, log busface.ILog) *TcpBusServer {
	return &TcpBusServer{
		log:      log,
		IP:       "0.0.0.0",
		Port:     port,
		MaxConn:  50,
		pack:     NewDataPack(),
		connPool: NewTcpPool(log),
		handler:  NewMsgHandle(),
	}
}

func (this *TcpBusServer) AddRouter(msgID uint32, router busface.IRouter) {
	this.handler.AddRouter(msgID, router)
}

func (this *TcpBusServer) Broadcast(data busface.IMessage) {
	this.connPool.Broadcast(data)
}

func (this *TcpBusServer) Send(ip string, data busface.IMessage) {
	this.connPool.Send(ip, data)
}

func (this *TcpBusServer) Start() {
	if this.pack == nil {
		panic("未设置 packFunc")
	}

	if this.handler == nil {
		panic("未设置 handler")
	}

	//开启一个go去做服务端Linster业务
	go func() {
		this.handler.StartWorkerPool()
		//1 获取一个TCP的Addr
		addr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", this.IP, this.Port))
		if err != nil {
			panic(err)
		}

		//2 监听服务器地址
		listener, err := net.ListenTCP("tcp4", addr)
		if err != nil {
			panic(err)
		}

		//TODO server.go 应该有一个自动生成ID的方法
		var cID uint32
		cID = 0

		this.log.Write(busface.LOG_LEVEL_INFO, "[服务端] 服务器正在监听.")

		//3 启动server网络连接业务
		for {
			//3.1 阻塞等待客户端建立连接请求
			conn, err := listener.AcceptTCP()
			if err != nil {
				this.log.Write(busface.LOG_LEVEL_ERROR,
					fmt.Sprintf("[服务端] 接收请求时发生错误: [%s]", err.Error()))
				continue
			}

			this.log.Write(busface.LOG_LEVEL_INFO,
				fmt.Sprintf("[服务端] 客户端建立连接: [%s]", conn.RemoteAddr().String()))

			//3.2 设置服务器最大连接控制,如果超过最大连接，那么则关闭此新的连接
			// if s.ConnMgr.Len() >= this.MaxConn {
			// 	conn.Close()
			// 	continue
			// }

			cID++
			//3.3 处理该新连接请求的 业务 方法， 此时应该有 handler 和 conn是绑定的
			dealConn := NewTcpConn(context.Background(), this.log, this, cID, conn, conn.RemoteAddr().String())

			//3.4 启动当前链接的处理业务
			go dealConn.Start()
			this.connPool.Join(cID, dealConn)
		}
	}()
}

func (this *TcpBusServer) GetHandler() busface.IHandler {
	return this.handler
}

func (this *TcpBusServer) GetPackFunc() busface.IPack {
	return this.pack
}
