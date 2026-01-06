package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	mrand "math/rand"
	"strconv"
	"sync"
)

//本地消息池（模拟持久化层），只有确认提交成功后才会存入此池
var localMessagePool = []CmdStructMessage{}

// 保存节点信息，节点的ID，地址和对应的公钥私钥对
type SructNode struct {
	nodeID     string //节点ID
	addr       string //节点监听地址,也就是自己的IP地址
	rsaPrivKey []byte //RSA私钥
	rsaPubKey  []byte //RSA公钥
}

/*****************************************************************************
map知识小科普，再看发现自己忘了
 map[keyType]valueType
*****************************************************************************/
// pbft的主结构体，里面保存了每个节点共识过程中所需要的所有信息
type StructPbft struct {
	node                SructNode                //节点信息，包括节点的ID，地址，私钥和公钥
	sequenceID          int                      //每笔请求自增序号，从0开始
	lock                sync.Mutex               //互斥锁
	requestMessagePool  map[string]StructRequest //这里面保存的是请求信息和客户端的地址信息，也就是request阶段所需要的所有信息
	prePareConfirmCount map[string]int           //存放收到的prepare数量(至少需要收到并确认2f个)，根据摘要来对应
	commitConfirmCount  map[string]int           //存放收到的commit数量（至少需要收到并确认2f+1个），根据摘要来对应
	isCommitBordcast    map[string]bool          //该笔消息是否已进行Commit广播
	isReply             map[string]bool          //该笔消息是否已对客户端进行Reply
	// 下面是2023年2.14添加的，为了加入门限签名
	// partThresholdValue string                       // 自身的部分门限签名
}

// 初始化节点StructPbft，也就是创建一个StructPbft对象当成一个共识节点
func NewPBFT(nodeID, addr string) *StructPbft {
	p := new(StructPbft)
	p.node.nodeID = nodeID
	p.node.addr = addr
	p.node.rsaPrivKey = p.getPivKey(nodeID) //从生成的私钥文件处读取
	p.sequenceID = 0
	p.requestMessagePool = make(map[string]StructRequest)
	p.prePareConfirmCount = make(map[string]int)
	p.commitConfirmCount = make(map[string]int)
	p.isCommitBordcast = make(map[string]bool)
	p.isReply = make(map[string]bool)
	return p
}

func (p *StructPbft) handleRequest(data []byte) {
	//切割消息，把消息切割为命令和内容，根据消息命令调用不同的功能
	cmd, content := splitMessage(data)
	switch cmd {
	case cRequest: // 搞定
		p.handlePreprepare(content)
	case cPrePrepare_vote:
		p.handlePrePrepare_vote(content)
	case cPreCommit:
		p.handlePrecommit(content)
	case cPreCommitVote:
		p.handlePreCommitVote(content)
	case cCommit:
		p.handleCommit(content)
	case cCommitVote:
		p.handleCommitVote(content)
	}
}

//处理客户端发来的请求，因为设定N0位主节点，所以这个消息其实只有N0能接收到，对消息进行处理，发出的PrePrepare消息
func (p *StructPbft) handlePreprepare(content []byte) {

	//使用json解析出Request结构体
	requsetMSG := new(StructRequest)
	err := json.Unmarshal(content, requsetMSG) // 解析得到客户端发送过来的RequestMSG
	if err != nil {
		log.Panic(err)
	}
	//fmt.Println(*requsetMSG)
	//添加信息序号
	p.sequenceIDAdd() // 也就是sequenceID的值加1
	//获取消息摘要
	hash := sha256.Sum256(content)
	//进行十六进制字符串编码
	digest := hex.EncodeToString(hash[:]) // 摘要其实就是字符串形式的hash值
	// digest := getDigest(*requsetMSG) // 得到的是hash值的字符串,这个20221129这个地方之所以注释掉，是因为现在content本身就是marshal之后的信息，要是调用这个函数，饶了弯路，所以给删除了
	//把请求消息，放入到请求消息池，存入临时消息池
	p.requestMessagePool[digest] = *requsetMSG
	//fmt.Println("摘要信息为", digest)
	//主节点对消息摘要进行签名
	// digestByte, _ := hex.DecodeString(digest) 	// 任永旺2022年11月29注释掉，这个如果把代码给修改后，digestByte其实就是hash值，更节约时间了
	//	p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)	// 任永旺2022年11月29注释掉，这个和下面一行代码都是签名，应该是重复进行了
	signInfo := p.RsaSignWithSha256(hash[:], p.node.rsaPrivKey)
	//拼接成PrePrepare，准备发往follower节点
	pp := PrePrepare{*requsetMSG, digest, p.sequenceID, signInfo}
	// 这个地方把N0的部分门限签名都包含进去
	set_1_PartThresholdSign(digest, "N0", allPartSig["N0"])
	set_2_PartThresholdSign(digest, "N0", allPartSig["N0"])
	set_3_PartThresholdSign(digest, "N0", allPartSig["N0"])
	b, err := json.Marshal(pp)
	if err != nil {
		log.Panic(err)
	}

	fmt.Println("正在向其他节点进行进行PrePrepare广播 ...")
	//进行PrePrepare广播
	randSleep()
	p.broadcast(cPrePrepare_vote, b) // 向除了自己之外的所有节点广播CPrePrepare消息
	//fmt.Println("PrePrepare广播完成")
}

//处理预准备消息
func (p *StructPbft) handlePrePrepare_vote(content []byte) {
	//	//使用json解析出PrePrepare结构体
	pp := new(PrePrepare)
	err := json.Unmarshal(content, pp)
	if err != nil {
		log.Panic(err)
	}
	//获取主节点的公钥，用于数字签名验证
	//primaryNodePubKey := p.getPubKey("N0") // 20221129这个要注释掉，每次都从文件中读物主节点的公钥和私钥太傻了
	digestByte, _ := hex.DecodeString(pp.Digest)
	digest := getDigest(pp.RequestMessage)
	// fmt.Println("digest你怎么样了")
	if digest != pp.Digest {
		fmt.Println("信息摘要对不上，拒绝进行prepare广播")
	} else if p.sequenceID+1 != pp.SequenceID {
		fmt.Println("消息序号对不上，拒绝进行prepare广播")
	} else if !p.RsaVerySignWithSha256(digestByte, pp.Sign, allNodeRsaPubKey[primaryID]) {
		fmt.Println("主节点签名验证失败！,拒绝进行prepare广播")
	} else {
		//fmt.Println("wo daole zhe 儿了")
		//序号赋值，这儿值应该变为1了
		p.sequenceID = pp.SequenceID
		//将信息存入临时消息池
		//fmt.Println("已将消息存入临时节点池")
		p.requestMessagePool[pp.Digest] = pp.RequestMessage
		//fmt.Println(p.node.nodeID, "这个节点保存了", pp.Digest, len(p.requestMessagePool))
		//fmt.Println("摘要为：", pp.Digest, pp.RequestMessage.Content)
		//节点使用私钥对其签名
		sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
		//拼接成Prepare
		pre := Prepare{pp.Digest, pp.SequenceID, p.node.nodeID, sign}
		bPre, err := json.Marshal(pre)
		if err != nil {
			log.Panic(err)
		}
		//进行准备阶段的广播
		// fmt.Println("正在进行Prepare广播 ...")
		randSleep()
		p.broadcastToPrimary(cPreCommit, bPre)
		//fmt.Println("Prepare广播完成")
	}
}

//处理准备消息
func (p *StructPbft) handlePrecommit(content []byte) {
	//使用json解析出Prepare结构体
	prepareMSG := new(Prepare)
	err := json.Unmarshal(content, prepareMSG)
	if err != nil {
		log.Panic(err)
	}
	//fmt.Printf("本节点已接收到%s节点发来的Prepare ... \n", pre.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	// MessageNodePubKey := p.getPubKey(pre.NodeID) 20221129删除，下面是用allNodeRsaPubKey[pre.NodeID]替换
	digestByte, _ := hex.DecodeString(prepareMSG.Digest)
	//fmt.Println(prepareMSG.Digest)
	if _, ok := p.requestMessagePool[prepareMSG.Digest]; !ok {
		fmt.Println("当前临时消息池无此摘要，拒绝执行commit广播")
	} else if p.sequenceID != prepareMSG.SequenceID {
		fmt.Println("消息序号对不上，拒绝执行commit广播")
	} else if !p.RsaVerySignWithSha256(digestByte, prepareMSG.Sign, allNodeRsaPubKey[prepareMSG.NodeID]) {
		fmt.Println("节点签名验证失败！,拒绝执行commit广播")
	} else {
		p.lock.Lock()
		set_1_PartThresholdSign(prepareMSG.Digest, prepareMSG.NodeID, allPartSig[prepareMSG.NodeID])

		//因为主节点不会发送Prepare，所以不包含自己
		//如果节点至少收到了2f个prepare的消息（包括自己）,并且没有进行过commit广播，则进行commit广播
		//获取消息源节点的公钥，用于数字签名验证
		if len(all_1_RcvPartSignPool[prepareMSG.Digest]) == leastConsensusNodeCount+1 {
			//fmt.Println("测试成功，本节点已收到至少2f个节点(包括本地节点)发来的Prepare信息 ...")
			//节点使用私钥对其签名
			var created [MaxNodeNumber]string
			km := 0
			for _, v := range all_1_RcvPartSignPool[prepareMSG.Digest] {
				created[km] = v
				km++
			}
			_, err := ThresholdSig(created[0:leastConsensusNodeCount+1], "renyongwang") // 这个是秘密共享的东西，要是门限签名还要修改
			if err != nil {
				fmt.Println("*******************************************************wo zheng guou cuo le")
			} else {
				sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
				c := Commit{prepareMSG.Digest, prepareMSG.SequenceID, p.node.nodeID, sign}
				bc, err := json.Marshal(c)
				if err != nil {
					log.Panic(err)
				}
				randSleep()
				p.broadcast(cPreCommitVote, bc)
			}

			//p.isCommitBordcast[prepareMSG.Digest] = true
		}
		p.lock.Unlock()
	}
}

//处理提交确认消息
func (p *StructPbft) handlePreCommitVote(content []byte) {
	//使用json解析出Commit结构体
	commitMSG := new(Commit)
	err := json.Unmarshal(content, commitMSG)
	if err != nil {
		log.Panic(err)
	}
	//fmt.Printf("本节点已接收到%s节点发来的Commit ... \n", c.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	// MessageNodePubKey := p.getPubKey(c.NodeID)20221129删除，下面是用allNodeRsaPubKey[pre.NodeID]替换
	digestByte, _ := hex.DecodeString(commitMSG.Digest)
	if p.sequenceID != commitMSG.SequenceID {
		fmt.Println("消息序号对不上，拒绝将信息持久化到本地消息池")
	} else if !p.RsaVerySignWithSha256(digestByte, commitMSG.Sign, allNodeRsaPubKey[commitMSG.NodeID]) {
		fmt.Println("节点签名验证失败！,拒绝将信息持久化到本地消息池")
	} else {
		p.lock.Lock()
		sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
		c := Commit{commitMSG.Digest, commitMSG.SequenceID, p.node.nodeID, sign}
		bc, err := json.Marshal(c)
		if err != nil {
			log.Panic(err)
		}
		p.broadcastToPrimary(cCommit, bc)
		p.lock.Unlock()
	}
}

//处理准备消息
func (p *StructPbft) handleCommit(content []byte) {
	commitMSG := new(Commit)
	err := json.Unmarshal(content, commitMSG)
	if err != nil {
		log.Panic(err)
	}
	//fmt.Printf("本节点已接收到%s节点发来的Commit ... \n", c.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	// MessageNodePubKey := p.getPubKey(c.NodeID)20221129删除，下面是用allNodeRsaPubKey[pre.NodeID]替换
	digestByte, _ := hex.DecodeString(commitMSG.Digest)
	//fmt.Println(prepareMSG.Digest)
	if p.sequenceID != commitMSG.SequenceID {
		fmt.Println("消息序号对不上，拒绝执行commit广播")
	} else if !p.RsaVerySignWithSha256(digestByte, commitMSG.Sign, allNodeRsaPubKey[commitMSG.NodeID]) {
		fmt.Println("节点签名验证失败！,拒绝执行commit广播")
	} else {
		p.lock.Lock()
		set_2_PartThresholdSign(commitMSG.Digest, commitMSG.NodeID, allPartSig[commitMSG.NodeID])

		//因为主节点不会发送Prepare，所以不包含自己
		//如果节点至少收到了2f个prepare的消息（包括自己）,并且没有进行过commit广播，则进行commit广播
		//获取消息源节点的公钥，用于数字签名验证
		if len(all_2_RcvPartSignPool[commitMSG.Digest]) == leastConsensusNodeCount+1 {
			//fmt.Println("测试成功，本节点已收到至少2f个节点(包括本地节点)发来的Prepare信息 ...")
			//节点使用私钥对其签名

			var created [MaxNodeNumber]string
			km := 0
			for _, v := range all_2_RcvPartSignPool[commitMSG.Digest] {
				created[km] = v
				km++
			}
			_, err := ThresholdSig(created[0:leastConsensusNodeCount+1], "renyongwang") // 这个是秘密共享的东西，要是门限签名还要修改
			if err != nil {
				fmt.Println("*******************************************************wo zheng guou cuo le")
			} else {
				sign := p.RsaSignWithSha256(digestByte, p.node.rsaPrivKey)
				c := Commit{commitMSG.Digest, commitMSG.SequenceID, p.node.nodeID, sign}
				bc, err := json.Marshal(c)
				if err != nil {
					log.Panic(err)
				}

				randSleep()
				p.broadcast(cCommitVote, bc)
			}
			//p.isCommitBordcast[prepareMSG.Digest] = true
		}
		p.lock.Unlock()
	}
}

//处理提交确认消息
func (p *StructPbft) handleCommitVote(content []byte) {
	//使用json解析出Commit结构体
	commitMSG := new(Commit)
	err := json.Unmarshal(content, commitMSG)
	if err != nil {
		log.Panic(err)
	}
	//fmt.Printf("本节点已接收到%s节点发来的Commit ... \n", c.NodeID)
	//获取消息源节点的公钥，用于数字签名验证
	// MessageNodePubKey := p.getPubKey(c.NodeID)20221129删除，下面是用allNodeRsaPubKey[pre.NodeID]替换
	digestByte, _ := hex.DecodeString(commitMSG.Digest)
	if p.sequenceID != commitMSG.SequenceID {
		fmt.Println("消息序号对不上，拒绝将信息持久化到本地消息池")
	} else if !p.RsaVerySignWithSha256(digestByte, commitMSG.Sign, allNodeRsaPubKey[commitMSG.NodeID]) {
		fmt.Println("节点签名验证失败！,拒绝将信息持久化到本地消息池")
	} else {
		p.lock.Lock()
		//p.setCommitConfirmCount(commitMSG.Digest)
		//如果节点至少收到了2f+1个commit消息（包括自己）,并且节点没有回复过,并且已进行过commit广播，则提交信息至本地消息池，并reply成功标志至客户端！
		//if p.commitConfirmCount[commitMSG.Digest] >= leastConsensusNodeCount && !p.isReply[commitMSG.Digest] && p.isCommitBordcast[commitMSG.Digest] {
		//fmt.Println("本节点已收到至少2f + 1 个节点(包括本地节点)发来的Commit信息 ...")
		//将消息信息，提交到本地消息池中！
		localMessagePool = append(localMessagePool, p.requestMessagePool[commitMSG.Digest].CmdStructMessage)
		info := strconv.Itoa(p.requestMessagePool[commitMSG.Digest].ID)
		//fmt.Println("正在reply客户端 ...", info)
		randSleep()
		tcpDial([]byte(info), p.requestMessagePool[commitMSG.Digest].ClientAddr)
		//p.isReply[commitMSG.Digest] = true
		//fmt.Println("reply完毕")
		//}
		p.lock.Unlock()
	}
}

//序号累加
func (p *StructPbft) sequenceIDAdd() {
	p.lock.Lock()
	p.sequenceID++
	p.lock.Unlock()
}

//向除自己外的其他节点进行广播,设置了90%的成功发送率
func (p *StructPbft) broadcast(cmd byte, content []byte) {
	for i := range nodeTable {

		if i != p.node.nodeID {
			message := jointMessage(cmd, content)
			//fmt.Println("我广播来了", nodeTable[i], "个信息")
			tcpDial(message, nodeTable[i]) // 20221129这个我把go去掉了，反倒更快了，启动线程所花费的时间应该比发送完数据还要多，我感觉这儿是没必要的，完成这一步再整下一步挺好的
			//fmt.Println("我广播来了", nodeTable[i], "个信息")
		}
	}
}

//向主节点广播消息
func (p *StructPbft) broadcastToPrimary(cmd byte, content []byte) {
	message := jointMessage(cmd, content)
	//fmt.Println("我广播来了", nodeTable[i], "个信息")
	tcpDial(message, nodeTable["N0"]) // 20221129这个我把go去掉了，反倒更快了，启动线程所花费的时间应该比发送完数据还要多，我感觉这儿是没必要的，完成这一步再整下一步挺好的
	//fmt.Println("我广播来了", nodeTable[i], "个信息")
}

//设置一定概率的播放成功率
func (p *StructPbft) broadcastProbably(cmd byte, content []byte) {
	for i := range nodeTable {
		if i == p.node.nodeID {
			continue
		}
		numRand := mrand.Intn(100) // 设置sendDataSuccessProbably的转发成功率
		if numRand < sendDataSuccessProbably {
			message := jointMessage(cmd, content)
			go tcpDial(message, nodeTable[i])
		}
	}
}

//为多重映射开辟赋值
func (p *StructPbft) setPrePareConfirmCount(val string) {
	if _, ok := p.prePareConfirmCount[val]; !ok {
		p.prePareConfirmCount[val] = 1
	} else {
		p.prePareConfirmCount[val] += 1
	}
}

//为多重映射开辟赋值
func (p *StructPbft) setCommitConfirmCount(val string) {
	if _, ok := p.commitConfirmCount[val]; !ok {
		p.commitConfirmCount[val] = 1
	}
	p.commitConfirmCount[val] += 1
}

//传入节点编号， 获取对应的私钥
func (p *StructPbft) getPivKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("../public/Keys/" + nodeID + "/" + nodeID + "_RSA_PIV")
	if err != nil {
		log.Panic(err)
	}
	return key
}

//传入节点编号， 获取对应的私钥
func getPivKey(nodeID string) []byte {
	key, err := ioutil.ReadFile("../public/Keys/" + nodeID + "/" + nodeID + "_RSA_PIV")
	if err != nil {
		log.Panic(err)
	}
	return key
}

// 0707
func set_1_PartThresholdSign(val, val2 string, value string) {
	//fmt.Println(val, val2, value)
	if _, ok := all_1_RcvPartSignPool[val]; !ok {
		all_1_RcvPartSignPool[val] = make(map[string]string)
	}
	//all_1_RcvPartSignPool[val][val2] = value
	// fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&可怜&&&&&&&&&&&&&&&&&&&&&&")
	// fmt.Println(value)
	// fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&可怜&&&&&&&&&&&&&&&&&&&&&&")
	if IsValidShare(value) == true { // 收到部分门限签名，记录其有效信息，防止一个节点在统计时重复验证
		all_1_RcvPartSignPool[val][val2] = value
	}
}

// 0707
func set_2_PartThresholdSign(val, val2 string, value string) {
	if _, ok := all_2_RcvPartSignPool[val]; !ok {
		all_2_RcvPartSignPool[val] = make(map[string]string)
	}
	//all_2_RcvPartSignPool[val][val2] = value
	// fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&可怜&&&&&&&&&&&&&&&&&&&&&&")
	// fmt.Println(value)
	// fmt.Println("&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&&可怜&&&&&&&&&&&&&&&&&&&&&&")
	if IsValidShare(value) == true { // 收到部分门限签名，记录其有效信息，防止一个节点在统计时重复验证
		all_2_RcvPartSignPool[val][val2] = value
	}
}

// 0707
func set_3_PartThresholdSign(val, val2 string, value string) {
	if _, ok := all_3_RcvPartSignPool[val]; !ok {
		all_3_RcvPartSignPool[val] = make(map[string]string)
	}
	//all_3_RcvPartSignPool[val][val2] = value
	if IsValidShare(value) == true { // 收到部分门限签名，记录其有效信息，防止一个节点在统计时重复验证
		all_3_RcvPartSignPool[val][val2] = value
	}
}
