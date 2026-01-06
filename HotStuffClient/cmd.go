package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
)

//<REQUEST,o,t,c>
type Request struct {
	Message         // Message自定义结构体，信息包含内容和ID
	Timestamp int64 // 时间戳
	//相当于clientID
	ClientAddr string // 客户端地址
}

type Message struct {
	Content string
	ID      int
}

const prefixCMDLength = 1

const (
	cRequest    byte = 'a' // 用a指请求
	cPrePrepare byte = 'b' // 用b指预准备
	cPrepare    byte = 'c' // 用c指准备
	cCommit     byte = 'd' // 用d指commit
)

//默认前十二位为命令名称，返回的消息为前12位为命令类型，后面为json编码的信息，也就是所传参数cmd是所传命令类型，content是json编码后的客户端数据
func jointMessage(cmd byte, content []byte) []byte {
	b := make([]byte, prefixCMDLength)
	b[0] = cmd
	joint := make([]byte, 0)
	joint = append(b, content...)
	return joint
}

//默认前十二位为命令名称，把命令和内容分隔开开来，进行返回，这个正好是jointMessage的反向操作，返回的是命令类型cmd和json编码数据content
func splitMessage(message []byte) (cmd byte, content []byte) {
	cmdBytes := message[:prefixCMDLength]
	cmd = byte(cmdBytes[0])
	content = message[prefixCMDLength:]
	return
}

//对消息详情进行摘要，获取数据的hash值
func getDigest(request Request) string {
	b, err := json.Marshal(request)
	if err != nil {
		log.Panic(err)
	}
	hash := sha256.Sum256(b) // 返回的是sha256的校验和
	//进行十六进制字符串编码
	return hex.EncodeToString(hash[:]) // 获取hash值
}
