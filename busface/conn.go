package busface

type IConn interface {
	Write(IMessage) error
	GetConnID() uint32
}

type IRequest interface {
	GetConnection() IConn //获取请求连接信息
	GetData() []byte      //获取请求消息的数据
	GetMsgID() uint32     //获取请求的消息ID
}

type IRouter interface {
	Handle(request IRequest) //处理conn业务的方法
}
