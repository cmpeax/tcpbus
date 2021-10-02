package busface

type IServer interface {
	Start()
	GetPackFunc() IPack
	GetHandler() IHandler
}
