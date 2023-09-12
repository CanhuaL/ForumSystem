package logic

import (
	"ForumSystem/dao/mysql"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"gopkg.in/fatih/set.v0"
	"log"
	"net"
	"net/http"
	"sync"
)

// 消息发送的类型
const (
	CMD_SINGLE_MSG = 10 //  点对点单聊，dstid是用户ID
	CMD_ROOM_MSG   = 11 //  群聊消息，dstid是群id
	CMD_HEART      = 12 //  心跳消息
)

type Message struct {
	Id      int64  `json:"id,omitempty" form:"id"`           //消息ID
	Userid  int64  `json:"userid,omitempty" form:"userid"`   //谁发的
	Cmd     int    `json:"cmd,omitempty" form:"cmd"`         //群聊还是私聊
	Dstid   int64  `json:"dstid,omitempty" form:"dstid"`     //对端用户ID/群ID
	Media   int    `json:"media,omitempty" form:"media"`     //消息按照什么样式展示
	Content string `json:"content,omitempty" form:"content"` //消息的内容
	Pic     string `json:"pic,omitempty" form:"pic"`         //预览图片
	Url     string `json:"url,omitempty" form:"url"`         //服务的URL
	Memo    string `json:"memo,omitempty" form:"memo"`       //简单描述
	Amount  int    `json:"amount,omitempty" form:"amount"`   //其他和数字相关的
}

// 本核心在于形成userid和Node的映射关系
type Node struct {
	Conn      *websocket.Conn //  保存websocket连接，conn是io型的资源
	DataQueue chan []byte     //  并行的数据转成串行的数据
	GroupSets set.Interface   //  组，线程安全
}

var contactService mysql.ContactService

// 映射关系表(map的键userid值是Node，全局的map，所有协程共享)
var clientMap map[int64]*Node = make(map[int64]*Node, 0)

// 读写锁
var rwlocker sync.RWMutex

func Chat(userID int64, c *gin.Context) error {
	//将当前http连接升级为websocket连接,之后每个协程都有自己的结构体,
	//结构体中保存当前协程的websocket连接,管道,群集合等,并且将自己的协程对应的
	//结构体bind到全局共享的map中,userid=>node,key=>value的形式,
	//并且每个协程都有自己两个子协程,(1)发送子协程:不断的读取当前这个协程对应的结构体
	//的管道,是否有数据通过管道传过来,如果有,则通过当前的node.Conn发送出去(管道:保证发送消息的顺序性)
	//(2)接收子协程:不断的从node.Conn中读取数据
	conn, err := (&websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}).Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		zap.L().Error("websocket.Upgrader has err")
	}

	//  获得conn
	node := &Node{
		Conn:      conn,                    //  Conn 类型表示 WebSocket 连接
		DataQueue: make(chan []byte, 50),   //  并行的数据转成串行的数据，有缓存
		GroupSets: set.New(set.ThreadSafe), //  初始化set，线程安全
	}

	//  获取用户全部群ID，之后放入这个用户的node.GroupSets中，
	//  发送群消息时，根据群id，遍历所有用户，看看每一个用户是否包含这个群id，有则发出送
	comIds := contactService.SearchComunityIds(userID)
	for _, v := range comIds {
		node.GroupSets.Add(v)
	}
	//  userID和node形成绑定关系，由于map操作频率大，用锁保证线程安全
	rwlocker.Lock()
	clientMap[userID] = node
	rwlocker.Unlock()
	//  发送conn
	go Sendproc(node)
	//  接收conn
	go Recvproc(node)
	log.Panicf("<-%d\n", userID)
	SendMsg(userID, []byte("HelloWorld!"))
	return err
}

// 发送消息函数
func SendMsg(userID int64, msg []byte) {
	rwlocker.RLocker()
	node, ok := clientMap[userID]
	rwlocker.Unlock()
	if ok {
		node.DataQueue <- msg
	}
}

// ws发送协程
func Sendproc(node *Node) {
	for {
		select {
		//  一直监听当前这个连接，是否有数据过来，当数据写道node.DataQueue，
		//  之后从node.DataQueue管道中读取数据，通过node.Conn发送到客户端
		case data := <-node.DataQueue:
			err := node.Conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				zap.L().Error(err.Error())
				return
			}
		}
	}
}

// ws接收协程
func Recvproc(node *Node) {
	for {
		//  不断的从websocket连接中读取数据
		_, data, err := node.Conn.ReadMessage()
		//  websocket关闭时，连接回收
		defer node.Conn.Close()
		if err != nil {
			zap.L().Error(err.Error())
			return
		}
		//  把消息广播到局域网
		BroadMsg(data)
		log.Panicf("[ws]<=%s\n", data)
	}
}

// 用来存放udp广播的数据
var udpsendchan chan []byte = make(chan []byte, 1024)

// 讲消息广播到局域网
func BroadMsg(data []byte) {
	udpsendchan <- data
}

func init() {
	go udpsendproc()
	go udprecvproc()
}

// 分布式:把消息广播到局域网:流程为服务端接收协程收到消息后,
// 把消息放到udp的channel中,之后每一台机器启动的时候都会建立一个udp的发送协程,接收协程
// 之后这台机器的udp发送协程从udp的channel中读取数据,将数据写入udp,广播出去,别的所有机器的udp接收协程会收到消息,
// 之后将这条消息发到指定用户,有的机器有这个用户,有的机器没有这个用户,没有就不发,有就发,因为用户userid是唯一的
// 一个用户只会连接到一台服务器上,可以换成rabbitmq则更好一些,确保消息准确到达
// 完成udp接收并处理功能
func udprecvproc() {
	log.Println("start udprecvproc")
	//  监听udp广播端口
	con, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: 3000,
	})
	defer con.Close()
	if err != nil {
		zap.L().Error(err.Error())
	}
	//  处理端口发过来的数据
	for {
		var buf [512]byte
		n, err := con.Read(buf[0:])
		if err != nil {
			zap.L().Error(err.Error())
			return
		}
		//  直接通过udp传过来数据
		dispatch(buf[0:n])
	}
	log.Println("stop udprecvproc")
}

// 完成udp数据的发送协程
func udpsendproc() {
	log.Println("start udpsendproc")
	//  使用udp协议拨号
	//  net.DialUDP函数用于创建一个UDP连接，用于向指定的UDP地址发送数据。
	//  它返回一个*net.UDPConn类型的对象，可以用于发送和接收UDP数据包。
	//  其中，network参数指定网络类型，通常为"udp"。laddr参数是本地地址，
	// 用于绑定本地端口。raddr参数是远程地址，指定要连接的目标UDP地址。
	con, err := net.DialUDP("udp", nil,
		&net.UDPAddr{
			IP:   net.IPv4(192, 168, 0, 255),
			Port: 3000,
		})
	defer con.Close()
	if err != nil {
		zap.L().Error(err.Error())
		return
	}
	//  通过得到的con发送消息
	for {
		select {
		case data := <-udpsendchan:
			_, err = con.Write(data)
			if err != nil {
				zap.L().Error(err.Error())
				return
			}
		}
	}
}

// 后端调度逻辑处理
func dispatch(data []byte) {
	//  解析data为message
	msg := Message{}
	//  将传过来的数据赋值到结构体中
	err := json.Unmarshal(data, &msg)
	if err != nil {
		zap.L().Error(err.Error())
		return
	}
	//  根据cmd对逻辑进行处理
	switch msg.Cmd {
	case CMD_SINGLE_MSG:
		//  私聊
		SendMsg(msg.Dstid, data)
	case CMD_ROOM_MSG:
		//  群聊，遍历所有clienMap
		for userId, v := range clientMap {
			if v.GroupSets.Has(msg.Dstid) {
				//  自己排除， 不发送
				if msg.Userid != userId {
					v.DataQueue <- data
				}
			}
		}
	case CMD_HEART:
		//  心跳，保证websocket长连接一直保持，一般什么都不做
	}
}

// 添加新的群ID到用户的groupset中
func AddGroupId(userId, gid int64) {
	//  取得node
	rwlocker.Lock()
	node, ok := clientMap[userId]
	if ok {
		node.GroupSets.Add(gid)
	}
	rwlocker.Unlock()
}
