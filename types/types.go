package types

import "time"

const (
	G_SELF         = "00"
	G_OTHER        = "01"
	G_MINTERACTIVE = "02" // 主动发起方缓存交互 key id value ipport_pubkey  跟对方谁在通信
	G_SINTERACTIVE = "03" // 被动发送方缓存交互 key id value para请求数据
	G_HASHDATA     = "04" // 链上数据 key id value peerSet序列化  peers数据  此类数据两个来源，1来自用户输入,2来自master

	T_CacheExpire = time.Second * 20
)
