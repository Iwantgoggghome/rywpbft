package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
)

/****************
任永旺2021年11月21号修改的版本，用来实现自己的大论文和第三篇英文文章的数据
其中用公共数据库来代替多个不同电脑间的数据传输
*****************/

const nodeCount = 56                                            // 共识节点的数量
const sleepTime = 2                                             // 随机休眠的最大时间为5ms
const waitTimeStop = 300                                        // 等待多久如果没有收到客户端发来的信息，这结束进程
const MaxNodeNumber = 100                                       // 最大节点数，这个如果超出了，要增加
const computerNO = 0                                            // 电脑编号，我的电脑设置为0，不同的编号设置不同
var computerCount = 2                                           // 主机数
var clientAddr = "192.168.124.12:8888"                          //客户端的监听地址
var leastConsensusNodeCount = nodeCount * computerCount / 3 * 2 // 最少共识节点数，这儿没加1，是为了在程序中使用方便
var primaryID = "N0"                                            //设置主节点的ID
var resetTime = false                                           // 重置计时，以方便结束当前应用
var sendDataSuccessProbably = 101                               // pp消息和p消息发送成功率
var nodeTable map[string]string                                 // 其实就是节点池，主要用来存储监听地址，也就是相应的IP地址和端口号
var nodeTable00 map[string]string                               // 这里保存的是以0.0.0.0开头的，和nodeTable对的端口号相同的东西
var allNodeRsaPubKey map[string][]byte                          //20221129保存所有节点的公钥信息
var allNodeRsaPrivateKey map[string][]byte                      //20221129保存所有节点的私钥信息
var allPartSig map[string]string                                // 保存所有部分签名，根据节点编号进行索引
var all_1_RcvPartSignPool map[string]map[string]string          // 部分门限签名池，记录自身受到的门限签名信息
var all_2_RcvPartSignPool map[string]map[string]string          // 部分门限签名池，记录自身受到的门限签名信息
var all_3_RcvPartSignPool map[string]map[string]string          // 部分门限签名池，记录自身受到的门限签名信息
var allRcvPartSignVerifyPool map[string]map[string]bool         // 门限签名的值是否已经验证，省的重复验证

//传入节点编号， 获取对应的公钥
func getPubKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("../public/Keys/" + nodeID + "/" + nodeID + "_RSA_PUB")
	if err != nil {
		log.Panic(err)
	}
	return key
}

//传入节点编号， 获取对应的私钥
func getPrivateKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("../public/Keys/" + nodeID + "/" + nodeID + "_RSA_PIV")
	if err != nil {
		log.Panic(err)
	}
	return key
}

// 根据节点数自动生成端口号和对应的IP地址，对应下面的nodeTable = map[string]string，这里的生成会根据nodeCount的数量自动的调整，所以只需要配置nodecount的值就行了
//var endTime time.Time
// 根据节点数自动生成端口号和对应的IP地址，对应下面的nodeTable = map[string]string
func initNodeTable() {
	nodeTable = make(map[string]string, 2)
	nodeTable00 = make(map[string]string, 2)
	// "N0": "127.0.0.1:8000",

	for i := 0; i < nodeCount; i++ {
		{
			numberS := fmt.Sprintf("N%d", i)
			IPs := fmt.Sprintf("192.168.124.12:80%02d", i)
			nodeTable[numberS] = IPs

			IP0 := fmt.Sprintf("0.0.0.0:80%02d", i)
			nodeTable00[numberS] = IP0
		}

		// 选择发给谁的，这不同的主机对应不同的端口号和IP地址，所以这边要全部列出来，这样整体节点数就是nodeCount*主机数了，这个是我的台式机的IP地址
		if computerCount > 1 {
			numberS1 := fmt.Sprintf("N%d", nodeCount*1+i)
			IPs1 := fmt.Sprintf("192.168.124.14:80%02d", nodeCount*1+i)
			IP1 := fmt.Sprintf("0.0.0.0:80%02d", nodeCount*1+i)
			nodeTable[numberS1] = IPs1

			nodeTable00[numberS1] = IP1
		}

		if computerCount > 2 { // 这个是17的
			numberS2 := fmt.Sprintf("N%d", nodeCount*2+i)
			IPs2 := fmt.Sprintf("192.168.124.94:80%02d", nodeCount*2+i) // 不同电脑的IP地址不同，17电脑
			nodeTable[numberS2] = IPs2

			IP2 := fmt.Sprintf("0.0.0.0:80%02d", nodeCount*2+i)
			nodeTable00[numberS2] = IP2
		}

		if computerCount > 3 { // 这个是puge的
			numberS3 := fmt.Sprintf("N%d", nodeCount*3+i)
			IPs3 := fmt.Sprintf("192.168.124.21:80%02d", nodeCount*3+i) // 不同电脑的IP地址不同，璞哥电脑
			nodeTable[numberS3] = IPs3

			IP3 := fmt.Sprintf("0.0.0.0:80%02d", nodeCount*3+i)
			nodeTable00[numberS3] = IP3
		}

		if computerCount > 4 { // 这个是马哥的
			numberS4 := fmt.Sprintf("N%d", nodeCount*4+i)
			IPs4 := fmt.Sprintf("192.168.124.32:80%02d", nodeCount*4+i) // 不同电脑的IP地址不同，璞哥电脑
			nodeTable[numberS4] = IPs4

			IP4 := fmt.Sprintf("0.0.0.0:80%02d", nodeCount*4+i)
			nodeTable00[numberS4] = IP4
		}

		if computerCount > 5 { // 这个是马哥的
			numberS5 := fmt.Sprintf("N%d", nodeCount*5+i)
			IPs5 := fmt.Sprintf("192.168.124.42:80%02d", nodeCount*5+i) // 不同电脑的IP地址不同，璞哥电脑
			nodeTable[numberS5] = IPs5

			IP5 := fmt.Sprintf("0.0.0.0:80%02d", nodeCount*5+i)
			nodeTable00[numberS5] = IP5
		}

	}
}

// 模拟的(t,n)门限签名，t是阈值，n是总数，secret是签名用的验证秘密
func initPartThresholdSig(t int, n int, secret string) ([]string, error) {
	fmt.Println(t, n)
	created, err := Create(t, n, secret)
	if err != nil {
		fmt.Println("Fatal: created PartThresholdSig: ", err)
		return []string{""}, errors.New("cuole")
	}
	allPartSig = make(map[string]string, 2)
	for i := 0; i < nodeCount*computerCount; i++ {
		nID := "N" + strconv.Itoa(i)
		allPartSig[nID] = created[i]
	}

	combined, err := ThresholdSig(created[0:75], "renyongwang") // 这个是秘密共享的东西，要是门限签名还要修改
	if err != nil {
		fmt.Println("wo cuo le 呢") // 到这儿了
	} else {
		fmt.Println(combined)
	}

	b, er := VerifyThresholdSig("renyongwang", combined)
	if b && er == nil {
		fmt.Println("wo yannzheng tongugo le hahahahaaha ")
	} else {
		fmt.Println("tainanle") // 这儿是通过的
	}

	b = IsValidShare(created[0])
	if b {
		fmt.Println("wo you xiao")
	}

	return created, nil
}

// 读取所有用到的公钥和私钥信息，并把他们保存到相应的map内
func initKeys() {
	allNodeRsaPubKey = make(map[string][]byte)
	allNodeRsaPrivateKey = make(map[string][]byte)
	for i := 0; i < nodeCount*computerCount; i++ {
		nID := "N" + strconv.Itoa(i)
		allNodeRsaPubKey[nID] = getPubKey(nID)
		allNodeRsaPrivateKey[nID] = getPivKey(nID)
	}
}

func main() {
	genRsaKeys()    //为节点生成公私钥，并把信息保存在public/keys目录下
	initNodeTable() // 根据nodeCount数，初始化节点信息
	// 为每个节点开辟一个线程，每个线程就是一个共识节点
	initKeys()
	all_1_RcvPartSignPool = make(map[string]map[string]string, 2)                                        // 初始化所有的用到的公钥和私钥
	all_2_RcvPartSignPool = make(map[string]map[string]string, 2)                                        // 初始化所有的用到的公钥和私钥
	all_3_RcvPartSignPool = make(map[string]map[string]string, 2)                                        // 初始化所有的用到的公钥和私钥
	_, er := initPartThresholdSig(nodeCount*computerCount/3*2+1, nodeCount*computerCount, "renyongwang") // 确定门限值和加密内容
	// 这个里面只包含nodecount的共识节点，所以不同的电脑启动的共识节点是不同的
	if er == nil {
		for i := nodeCount * computerNO; i < nodeCount*(computerNO+1); i++ {
			nID := "N" + strconv.Itoa(i)
			p := NewPBFT(nID, nodeTable00[nID]) // 初始化节点信息
			//fmt.Println(nID, nodeTable00[nID])
			go p.tcpListen() // 开启nodeCount个线程，每个线程代表一个共识节点
		}
	} else {
		fmt.Println("main func初始化部分门限签名出错")
	}

	select {} // 为了保证程序在你手动结束之前，一直不会结束
}
