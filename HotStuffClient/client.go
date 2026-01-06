package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// 在用，给主节点发送信息
// 给主节点发送信息
func sendMSGToPrimary(data string, nID int) {
	r := new(Request) // Request是一个自定义结构体，分别保存下面赋值内容的信息
	r.Timestamp = time.Now().UnixNano()
	r.ClientAddr = clientAddr // "127.0.0.1:8888"
	r.Message.ID = nID

	//消息内容就是用户的输入
	r.Message.Content = strings.TrimSpace(data)
	br, err := json.Marshal(r) // 对r用json进行编码
	if err != nil {
		log.Panic(err)
	}
	//fmt.Println(string(br))               // 输出编码信息{"Content":"renyongwangshigedabendan","ID":4687201663,"Timestamp":1622769567507361000,"ClientAddr":"127.0.0.1:8888"}
	content := jointMessage(cRequest, br) // 合成请求信息
	//默认N0为主节点，直接把请求信息发送至N0
	tcpDial(content, nodeTable["N0"])
	//tcpDial(content, nodeTable["N0"])

}

// 在用，随机休眠1-10ms
func randSleep() {
	time.Sleep(time.Millisecond * sleepTime)
}

/*** 用到的，开启多个线程，1是启动tcp中的线程clientTcpListen()监听客户端收到的信息，把信息保存到管道ChanNum中
	2是
***/
func clientSendMessageAndListen() {

	go clientTcpListen() //把收到信息放入ChanNum
	go countChanNum()
	startTime = time.Now() // 获取当前时间
	sendMSGToPrimary("ryw直接在这儿启动就发送数据了，所以呢，要先启动服务器，再启动这个客户端", 0)

	go func() {
		for {
			t2 := time.Now()
			time.Sleep(time.Millisecond * 10)
			if bStopMark {
				fmt.Printf("&&&&&&&&&&&&&&&&&&时间到了,总用时%v，单位是s哈&&&&&&&&&&&&&&&&&&\n", (t2.Sub(startTime)/1000)/1000)
				return
			}
		}
	}()
}
