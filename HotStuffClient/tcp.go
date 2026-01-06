package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strconv"
)

// // 在用。客户端监听计时器，如果客户端在100ms内没有接收到反馈，则说明协议没有达成，数据要重新发送
// func clientTcpListenTimer(timer *time.Timer) {
// 	for {
// 		fmt.Println("tcp.go clientTcpListenTimer是否还在工作")
// 		<-timer.C
// 		timer.Reset(time.Millisecond * resendDataTime)
// 		ChanNum <- "failget"
// 		fmt.Println("wo没有发送数据，计时器开始工作了")
// 	}
// }

//在用。客户端使用的tcp监听,就是看你是否在客户端输入了信息,把监听到的信息存入到管道ChanNum中
func clientTcpListen() {
	// 建立tcp服务
	listen, err := net.Listen("tcp", "0.0.0.0:8888") // 监听所有给这个客户端发送的信息
	if err != nil {
		log.Panic(err)
	}
	defer listen.Close()
	for { // 无线循环，一直监听
		conn, err := listen.Accept() // 监听，等待客户端建立连接
		if err != nil {
			log.Panic(err)
		}
		b, err := ioutil.ReadAll(conn) // 如果监听到内容，就读取出来，内容包括id，时间戳，和客户端地址等
		if err != nil {
			log.Panic(err)
		}
		strB := string(b)
		fmt.Println("tcp.go  clientTcpListen任永旺啥情况" + strB)
		ChanNum <- strB
		if bStopMark {
			return
		}
	}
}

// 在用的函数，计算收到了多少确认信息
func countChanNum() {
	Mapmark := make(map[string]int)
	nCount := 0
	for {
		strB, ok := <-ChanNum
		if !ok {
			break
		}
		if _, ok := Mapmark[strB]; ok {
			Mapmark[strB]++
			fmt.Println(strB, Mapmark[strB])
			if Mapmark[strB] == nodeCount*computerCount/3+1 {
				nCount++
				str := "成功接收区块数" + strconv.Itoa(nCount)
				sendMSGToPrimary(str, nCount)
				if nCount == stopNum {
					bStopMark = true
					chanStop <- bStopMark
					return
				}
			}
		} else {
			Mapmark[strB] = 1
		}
	}
}

//在用，使用tcp发送消息
func tcpDial(context []byte, addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println("connect error", err)
		return
	}

	_, err = conn.Write(context)
	if err != nil {
		log.Fatal(err)
	}
	conn.Close()
}
