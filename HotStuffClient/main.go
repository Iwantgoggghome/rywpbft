package main

import (
	"fmt"
	"time"
)

const nodeCount = 56
const sleepTime = 5 // 随机休眠的最大时间为5ms,现在改为非随机休眠

//客户端的监听地址
var computerCount = 2                         // 有多少台主机参与到共识当中
var clientAddr = "192.168.124.12:8888"        // 任永旺 客户端地址因为要多台电脑，所以改成自己的IP地址了
var ChanNum = make(chan string, nodeCount*10) //获取得到的确认的节点数
var bStopMark = false                         // 是否停止发送数据
var chanStop = make(chan bool, 1)             // 主程序是否终止
var stopNum = 10                              // 达成多少次共识后主程序退出

// var g_dsn = "root:password@tcp(222.22.65.217)/test11" // 设置要访问的数据库的相关信息
// var g_databaseListName = "ip0"                        // 这个是自己填写信息的数据库的列表名，不同的机器，这个列表名应该不同
// var g_sourcID = 0                                     //  和上面一样，需要根据不同的机器做出相应的修改，两台机器不能相同

//节点池，主要用来存储监听地址
var nodeTable map[string]string // 其实就是
var startTime time.Time

const counttime = 300     // 计时，单位为秒，程序统计多长时间的数据
const resendDataTime = 20 // 如果200ms没有收到数据，则任务共识没有达成，重新发送数据
const timeResentWait = 15 // 这个必须小于上面那个数值，因为电脑算力有限，所以节点数越多，这个值就要越大，这里是在34个节点时我测试的不死的，最后得结果要减去这个

//var endTime time.Time
// 根据节点数自动生成端口号和对应的IP地址，对应下面的nodeTable = map[string]string
func initNodeTable() {
	nodeTable = make(map[string]string, 2)
	// "N0": "127.0.0.1:8000",

	for i := 0; i < nodeCount; i++ {
		numberS := fmt.Sprintf("N%d", i)
		IPs := fmt.Sprintf("192.168.124.12:80%02d", i) // 不同电脑的IP地址不同
		nodeTable[numberS] = IPs

		// 选择发给谁的，这不同的主机对应不同的端口号和IP地址，所以这边要全部列出来，这样整体节点数就是nodeCount*主机数了，这个是我的台式机的IP地址
		if computerCount > 1 {
			numberS1 := fmt.Sprintf("N%d", nodeCount*1+i)
			IPs1 := fmt.Sprintf("192.168.124.14:80%02d", nodeCount*1+i) // 不同电脑的IP地址不同，我的台式机
			nodeTable[numberS1] = IPs1
		}
		if computerCount > 2 {
			numberS2 := fmt.Sprintf("N%d", nodeCount*2+i)
			IPs2 := fmt.Sprintf("192.168.124.94:80%02d", nodeCount*2+i) // 不同电脑的IP地址不同，17电脑
			nodeTable[numberS2] = IPs2
		}
		if computerCount > 3 {
			numberS3 := fmt.Sprintf("N%d", nodeCount*3+i)
			IPs3 := fmt.Sprintf("192.168.124.21:80%02d", nodeCount*3+i) // 不同电脑的IP地址不同，璞哥电脑
			nodeTable[numberS3] = IPs3
		}
		if computerCount > 4 {
			numberS4 := fmt.Sprintf("N%d", nodeCount*4+i)
			IPs4 := fmt.Sprintf("192.168.124.32:80%02d", nodeCount*4+i) // 不同电脑的IP地址不同，璞哥电脑
			nodeTable[numberS4] = IPs4
		}
		if computerCount > 5 {
			numberS5 := fmt.Sprintf("N%d", nodeCount*5+i)
			IPs5 := fmt.Sprintf("192.168.124.42:80%02d", nodeCount*5+i) // 不同电脑的IP地址不同，璞哥电脑
			nodeTable[numberS5] = IPs5
		}

	}
	// 遍历nodetable中的所有信息
	// for i := 0; i < len(nodeTable); i++ {
	// 	numberS1 := fmt.Sprintf("N%d", i)
	// 	fmt.Println(nodeTable[numberS1])
	// }
}

func main() {
	initNodeTable()
	clientSendMessageAndListen()
	//go queryClientSendData()

	select {
	case <-chanStop:
		time.Sleep(time.Second)
		break
	}
}
