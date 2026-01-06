package main

/***
主客户端充当发送消息给所有分客户端的客户端，消息是从这儿发出来的，分客户端检测到消息更新后，再把消息分发给对应的节点。
最终共识是否达成，也是在这儿进行确认的
*****/

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const nodeCount = 7

var chanSendClientData = make(chan bool, 10) // 判断客户端共识是否达成，继续发送下一个共识数据

var g_dsn = "root:password@tcp(222.22.65.217)/test11" // 设置要访问的数据库的相关信息
var stopNum = 11                                      // 达成多少次共识后主程序退出

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

type StructCommandMSG struct {
	MsgSourceID int // Message自定义结构体，信息包含内容和ID
	MsgType     int // 时间戳
	MsgNum      int // 客户端地址
}

func insertData(msg *Request) {
	conn, err := sql.Open("mysql", g_dsn)
	if err != nil {
		fmt.Println("insertData我没有打开数据库")
		panic(err)
	}
	defer conn.Close()
	fmt.Println("insertData我已经打开了数据库")
	fmt.Println("insert clientsendmsg(nMsgID,sMsgContent,nTimeStamp,ClientAddr) values(?,?,?,?)", *&msg.Message.ID, *&msg.Message.Content, *&msg.Timestamp, *&msg.ClientAddr)
	_, err = conn.Exec("insert clientsendmsg(nMsgID,sMsgContent,nTimeStamp,ClientAddr) values(?,?,?,?)", *&msg.Message.ID, *&msg.Message.Content, *&msg.Timestamp, *&msg.ClientAddr)
	if err != nil {
		fmt.Println("insertData插入操作没有完成")
		panic(err)
	} else {
		fmt.Println("数据插入完成")
	}
}

// 查询客户端发送的数据
func queryClientSendData() Request {
	conn, err := sql.Open("mysql", g_dsn)
	if err != nil {
		fmt.Println("queryClientSendData我没有打开数据库")
		panic(err)
	}
	defer conn.Close()
	fmt.Println("queryClientSendData我已经打开了数据库")
	data, err := conn.Query("select * from clientsendmsg")
	msg := Request{}
	if err == nil {
		for data.Next() {
			data.Scan(&msg.Message.ID, &msg.Message.Content, &msg.Timestamp, &msg.ClientAddr)
			//fmt.Println(msg)

			return msg
		}
	}
	return msg
}

// 查询是否收到消息标号为msgNum的足够的确认信息
func queryCommitMSG() {
	msgNum := 1
	conn, err := sql.Open("mysql", g_dsn)
	if err != nil {
		fmt.Println("queryCommitMSG我没有打开数据库")
		panic(err)
	}
	defer conn.Close()
	num := 0
	for {	
		data, err := conn.Query("select count(*) from ip0 where MsgType = 4 and MsgNum = ?", msgNum)
		if err == nil {
			for data.Next() {
				data.Scan(&num)
				if num > 0 { // ryw20221202这个地方以后要改成nodeCount/3+1,现在先这么测试
					chanSendClientData <- true
					fmt.Println("这就是我想要的结果呀", num)
					msgNum += 1
				} else {
					time.Sleep(time.Millisecond * 10)
				}

			}
		}
	}

}

// 查询客户端发送的数据的最大ID
func queryMaxID() int {
	conn, err := sql.Open("mysql", g_dsn)
	if err != nil {
		fmt.Println("queryMaxID我没有打开数据库")
		panic(err)
	}
	defer conn.Close()
	fmt.Println("queryMaxID我已经打开了数据库")
	data, err := conn.Query("select MAX(nMsgID) from clientsendmsg")
	var maxID int
	if err == nil {
		for data.Next() {
			data.Scan(&maxID)
			return maxID
		}
	}
	return 0
}

func clearAllData() {
	conn, err := sql.Open("mysql", g_dsn)
	if err != nil {
		fmt.Println("queryMaxID我没有打开数据库")
		panic(err)
	}
	defer conn.Close()
	_, err = conn.Exec("delete from clientsendmsg") // 删除clientsendmsg中的消息
	if err != nil {
		fmt.Println("delete操作没有完成")
		panic(err)
	} else {
		fmt.Println("数据删除完成")
	}

	_, err = conn.Exec("delete from ip0") // 删除ip0中的消息
	if err != nil {
		fmt.Println("delete操作没有完成")
		panic(err)
	} else {
		fmt.Println("数据删除完成")
	}
}

func main() {
	//clearAllData()
	go queryCommitMSG() // 不停的查询是否收到确认消息
	for i := 0; i < stopNum; i++ {
		sendClientMSG := new(Request) // Request是一个自定义结构体，分别保存下面赋值内容的信息
		sendClientMSG.Timestamp = time.Now().UnixNano()
		sendClientMSG.ClientAddr = "127.0.0.1:8888" // "127.0.0.1:8888"
		sendClientMSG.Message.ID = i
		sendClientMSG.Message.Content = "任永旺正在对如下编号的区块进行共识" + strconv.Itoa(i)
		if i != 0 {
			if <-chanSendClientData {
				insertData(sendClientMSG)
			}
		} else {
			insertData(sendClientMSG)
		}

	}
	fmt.Println(queryClientSendData())

	// fmt.Println(time.Now().UnixNano())
	// maxid := queryMaxID()
	// if maxid != -1 {
	// 	fmt.Println(maxid)
	// } else {
	// 	fmt.Println("读取最大id失败")
	// }
	// msg := queryClientSendData()
	// fmt.Println(msg)
	// msg.ID = 3
	// insertData(&msg)
}
