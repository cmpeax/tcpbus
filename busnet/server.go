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
}

func NewTcpPool() *tcpPool {
	return &tcpPool{}
}

func (this *tcpPool) Join(id uint32, conn *TcpConn) {
	// 加入时,监听. 以便删除

	this.conns.Store(id, conn)
	go func(cn *TcpConn, tid uint32) {
		<-cn.GetContext().Done()
		// 检测到连接失效时,删除该链接。
		this.conns.Delete(tid)
	}(conn, id)
}

func (this *tcpPool) Broadcast(data busface.IMessage) {
	this.conns.Range(func(key interface{}, value interface{}) bool {
		matchConn := value.(*TcpConn)
		matchConn.Write(data)
		return true
	})
}

func (this *tcpPool) Send(ip string, data busface.IMessage) {
	this.conns.Range(func(key interface{}, value interface{}) bool {
		matchConn := value.(*TcpConn)
		connAddr := strings.Split(matchConn.addr, ":")
		if len(connAddr) == 2 {
			if connAddr[0] == ip {
				matchConn.Write(data)
			}
		}
		return true
	})
}

// 服务端
type TcpBusServer struct {
	IP       string
	Port     int
	MaxConn  int
	handler  busface.IHandler
	pack     busface.IPack
	connPool *tcpPool
}

func NewServer(port int) *TcpBusServer {
	return &TcpBusServer{
		IP:       "0.0.0.0",
		Port:     port,
		MaxConn:  50,
		pack:     NewDataPack(),
		connPool: NewTcpPool(),
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

		fmt.Println("服务端正在监听中... ")
		//3 启动server网络连接业务
		for {
			//3.1 阻塞等待客户端建立连接请求
			conn, err := listener.AcceptTCP()
			if err != nil {
				fmt.Println("Accept err ", err)
				continue
			}
			fmt.Println("Get conn remote addr = ", conn.RemoteAddr().String())

			//3.2 设置服务器最大连接控制,如果超过最大连接，那么则关闭此新的连接
			// if s.ConnMgr.Len() >= this.MaxConn {
			// 	conn.Close()
			// 	continue
			// }

			cID++
			//3.3 处理该新连接请求的 业务 方法， 此时应该有 handler 和 conn是绑定的
			dealConn := NewTcpConn(context.Background(), this, cID, conn, conn.RemoteAddr().String())

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
