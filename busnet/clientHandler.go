package busnet

import (
	"sync"

	"github.com/cmpeax/tcpbus/busface"
)

// ClientHandle -
type ClientHandle struct {
	sync.RWMutex
	Apis map[uint32]busface.IRouter //存放每个MsgID 所对应的处理方法的map属性
}

//NewClientHandle 创建ClientHandle
func NewClientHandle() *ClientHandle {
	return &ClientHandle{
		Apis: make(map[uint32]busface.IRouter),
	}
}

//DoClientHandler 马上以非阻塞方式处理消息
func (mh *ClientHandle) DoHandler(request busface.IRequest) {
	mh.RLock()
	defer mh.RUnlock()
	handler, ok := mh.Apis[request.GetMsgID()]
	if !ok {
		return
	}

	//执行对应处理方法
	handler.Handle(request)
}

//AddRouter 为消息添加具体的处理逻辑
func (mh *ClientHandle) AddRouter(msgID uint32, router busface.IRouter) {
	mh.Lock()
	defer mh.Unlock()
	//2 添加msg与api的绑定关系
	mh.Apis[msgID] = router
}

//AddRouter 为消息添加具体的处理逻辑
func (mh *ClientHandle) RemoveRouter(msgID uint32) {
	mh.Lock()
	defer mh.Unlock()
	//2 添加msg与api的绑定关系
	delete(mh.Apis, msgID)
}
