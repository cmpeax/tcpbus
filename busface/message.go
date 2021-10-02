package busface

type IMessage interface {
	GetDataLen() uint32 //获取消息数据段长度
	GetMsgID() uint32   //获取消息ID
	GetData() []byte    //获取消息内容

	SetMsgID(uint32)   //设计消息ID
	SetData([]byte)    //设计消息内容
	SetDataLen(uint32) //设置消息数据段长度
}

type IPack interface {
	Unpack([]byte) (IMessage, error)
	Pack(IMessage) ([]byte, error)
	GetHeadLen() uint32
}

type IHandler interface {
	DoMsgHandler(IRequest)       //马上以非阻塞方式处理消息
	AddRouter(uint32, IRouter)   //为消息添加具体的处理逻辑
	StartWorkerPool()            //启动worker工作池
	SendMsgToTaskQueue(IRequest) //将消息交给TaskQueue,由worker进行处理
}

type IClientHandler interface {
	DoHandler(IRequest)        //马上以非阻塞方式处理消息
	AddRouter(uint32, IRouter) //为消息添加具体的处理逻辑
	RemoveRouter(uint32)       // 移除消息
}
