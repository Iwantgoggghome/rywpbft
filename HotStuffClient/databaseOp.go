package main

// import (
// 	"database/sql"
// 	"fmt"
// 	"time"

// 	_ "github.com/go-sql-driver/mysql"
// )

// // 查询客户端发送的数据，每隔5ms检查一次，查看是否有数据更新
// func queryClientSendData() {
// 	conn, err := sql.Open("mysql", g_dsn)
// 	if err != nil {
// 		fmt.Println("queryClientSendData我没有打开数据库")
// 		panic(err)
// 	}
// 	defer conn.Close()
// 	fmt.Println("queryClientSendData我已经打开了数据库")
// 	for i := 0; i < stopNum; {
// 		data, err := conn.Query("select * from clientsendmsg where nMsgID=?", i)
// 		msg := Request{}
// 		if err == nil {
// 			for data.Next() {
// 				data.Scan(&msg.Message.ID, &msg.Message.Content, &msg.Timestamp, &msg.ClientAddr)
// 				fmt.Println(msg)
// 				sendMSGToPrimary(&msg)
// 				i++
// 			}
// 		}
// 		time.Sleep(time.Millisecond * 10)
// 	}
// }

// func insertData(msg *Request) {
// 	conn, err := sql.Open("mysql", g_dsn)
// 	if err != nil {
// 		fmt.Println("insertData我没有打开数据库")
// 		panic(err)
// 	}
// 	defer conn.Close()
// 	// fmt.Println("insertData我已经打开了数据库")
// 	// fmt.Println("insert clientsendmsg(nMsgID,sMsgContent,nTimeStamp,ClientAddr) values(?,?,?,?)", *&msg.Message.ID, *&msg.Message.Content, *&msg.Timestamp, *&msg.ClientAddr)
// 	_, err = conn.Exec("insert clientsendmsg(nMsgID,sMsgContent,nTimeStamp,ClientAddr) values(?,?,?,?)", *&msg.Message.ID, *&msg.Message.Content, *&msg.Timestamp, *&msg.ClientAddr)
// 	if err != nil {
// 		fmt.Println("insertData插入操作没有完成")
// 		panic(err)
// 	} else {
// 		fmt.Println("数据插入完成")
// 	}
// }

// func insertCommitMSG(MType int, MNum int) {
// 	conn, err := sql.Open("mysql", g_dsn)
// 	if err != nil {
// 		fmt.Println("insertData我没有打开数据库")
// 		panic(err)
// 	}
// 	defer conn.Close()

// 	//_, err = conn.Exec("insert ?(MsgSourceID,MsgType,MsgNum) values(?,?,?)", g_databaseListName, g_sourcID, MType, MNum)
// 	_, err = conn.Exec("insert ip0(MsgSourceID,MsgType,MsgNum) values(?,?,?)", g_sourcID, MType, MNum)
// 	if err != nil {
// 		fmt.Println("insertData插入操作没有完成")
// 		panic(err)
// 	} else {
// 		fmt.Println("数据插入完成")
// 	}
// }
