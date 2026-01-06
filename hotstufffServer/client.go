package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	mrand "math/rand"
	"os"
	"strings"
	"time"
)

// 随机休眠1-10ms
func randSleep() {
	num := mrand.Intn(sleepTime) + 1
	for i := 0; i < num; i++ {
		time.Sleep(time.Millisecond)
	}
}

func clientSendMessageAndListen() {
	//开启客户端的本地监听（主要用来接收节点的reply信息）
	go clientTcpListen()
	fmt.Printf("客户端开启监听，地址：%s\n", clientAddr)

	fmt.Println(" ---------------------------------------------------------------------------------")
	fmt.Println("|  已进入PBFT测试Demo客户端，请启动全部节点后再发送消息！ :)  |")
	fmt.Println(" ---------------------------------------------------------------------------------")
	fmt.Println("请在下方输入要存入节点的信息：")
	//首先通过命令行获取用户输入
	stdReader := bufio.NewReader(os.Stdin)
	for {
		data, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}
		r := new(StructRequest) // Request是一个自定义结构体，分别保存下面赋值内容的信息
		r.Timestamp = time.Now().UnixNano()
		r.ClientAddr = clientAddr
		r.CmdStructMessage.ID = getRandom()
		//消息内容就是用户的输入
		r.CmdStructMessage.Content = strings.TrimSpace(data)
		br, err := json.Marshal(r) // 对r用json进行编码
		if err != nil {
			log.Panic(err)
		}
		fmt.Println(string(br))               // 输出编码信息{"Content":"renyongwangshigedabendan","ID":4687201663,"Timestamp":1622769567507361000,"ClientAddr":"127.0.0.1:8888"}
		content := jointMessage(cRequest, br) // 合成请求信息
		//默认N0为主节点，直接把请求信息发送至N0
		randSleep()
		tcpDial(content, nodeTable["N0"])
	}
}

//返回一个十位数的随机数，作为msgid
func getRandom() int {
	x := big.NewInt(10000000000)
	for {
		result, err := rand.Int(rand.Reader, x)
		if err != nil {
			log.Panic(err)
		}
		if result.Int64() > 1000000000 {
			return int(result.Int64())
		}
	}
}
