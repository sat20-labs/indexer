package common

// get读取优先级
type ReadPriority int

const (
	NetworkFirst ReadPriority = iota // 0 网络优先
	LocalFirst   ReadPriority = iota // 1 本地优先
)
