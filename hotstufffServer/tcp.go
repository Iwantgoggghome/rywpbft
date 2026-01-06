package main

/***
这个文件中的内容主要是接收客户端发来的消息，接收节点间互相发送的消息和发送消息给指定IP
****/
import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
)

//客户端使用的tcp监听,接受客户端发送过来的信息，然后分发给服务器进行处理
func clientTcpListen() {
	// 建立tcp服务
	listen, err := net.Listen("tcp", clientAddr)
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
		fmt.Println(string(b)) // 打印出监听内容
	}
}

//节点使用的tcp监听，这也就是节点接受发给自己的信息
func (p *StructPbft) tcpListen() {
	listen, err := net.Listen("tcp", p.node.addr)
	//fmt.Println("这是我要看的内容", p.node.addr)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("节点开启监听，地址：%s\n", p.node.addr)
	defer listen.Close()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Panic(err)
		}
		b, err := ioutil.ReadAll(conn)
		if err != nil {
			log.Panic(err)
		}
		p.handleRequest(b)
	}

}

//使用tcp发送消息，把内容context发送到addr地址
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
