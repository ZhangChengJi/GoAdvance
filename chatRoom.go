package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

/*
聊天室模块划分：
主go程：
创建监听socket。for循环Accept0客户端连接-conn。启动go程HandlerConnect：

HandlerConnect:
创建用户结构体对象。存入onlineMap。发送用户登录广播、聊天消息。处理查询在线用户、改名、下线、超时提出，

Manager:
监听全局channel message，将读到的消息广播给onlineMap中的所有用户。

WriteMsgToClient:
读取每个用户自带channelC上消息（由Manager发送该消息）。回写给用户。

全局数据模块：
用户结构体：Client{C、Name. Addr string}
在现用户列表：onlineMap【string】Client key：客户端IP+port value： Client

消息通道：message
*/
/** 							1.广播上线用户：
1.主go程中，创建监听狄接字。记得defer
2. for征环监听密户端连接请求。Accept（）
3.有一个密户端连接，创建新go程处理嵇户端数据HandlerConnet（conn）defer
4.定义全局结构体类型C、Name、Addr
5.创建全局map. channel
6.实现HandlerConnet，获取密户端IP+port一RemoteAddr0。初始化新用户结构体信息。name ==Addr
7.创建Manager实现管理go程。Accept0之前。
8.实现Manager。初始化在线用户map。征环读取全局channel，如果无数据，阻立。如果有数据，遍历在线用户map，将数据写到用户的C里9.将新用户添加到在线用户map中。Key==IP+port value=新用户结构体
10. 创建WriteMsgToClient go程，专门给当前用户写数据。
来源于用户自带的C中
11.实现WriteMsgTolient （cnt，conn）。遍历自带的C，读数据，conn.Write到密户端
12.HandlerConnet中，结束位置，组织用户上线信息，将用户上线化息写到全局 channel
-Manager的读就被激活（原来一直阻立）
13. HandlerConnet中，结尼加for{；}
*/
/**
								2.广播用户消息：
1.封装函数MakeMsg0来处理广播、用户消息
2. HandlerConnet中，创建匿名go程，读取用户socket上发送来的聊天内容。写到全局channel
3. for循环conn.Read n==0 err!= nil
4.写给全局message一后续的事，原来广播用户上线模块完成。（Manager.WriteMsgToClient
*/

/*
							  	3.查询在线用户：
1.将读取到的用户消息msg结尾的“\n”去掉。
2.判断是否是“who”命令
3.如果是，遍历在线用户列表，组织显示信息。写到socket中。
4.如果不是。写给全局message
*/

/*
								4.修改用户名：
1.将读取到的用户消息msg判断是否包含“rename|"
2.提取“”后面的字符串。存入到Client的Name成员中
3.更新在线用户列表。onlineMap. key - IP+port
4.提示用户更新完成。conn.Write
*/
/*
								5.用户退出：
1.在用户成功登陆之后，创建监听用户退出的channel一isQuit
2.当conn.Read ==0，isQuit 《-true
3.在HandlerConnet结几for中，添加select 监听《-isQuitI
4.条件满足。将用户从在结列表移除。组织用户下线消息，写入message（广播）
*/

/*
								6.超时强踢：
1.在select中监听定时器。（time.After（）计时到达。将用户从在线列表移除。组织用户下线消患，写入message（广插） 2.创建监听用户活跃的channel 一 hasData
3.只用户执行：聊天、改名、who任意一个操作，hasData《- true
4.在select中添加监听《-hasData。条件满足，不做任何事情。目的是重量计时器。
*/

// Client 创建用户结构体类型
type Client struct {
	C    chan string
	Name string
	Addr string
}

//创建全局map,存储在线用户
var onlineMap map[string]*Client

//创建全局channal 传递用户消息
var message = make(chan string)

func WriteMsgToClient(clnt *Client, conn net.Conn) {
	//监听用户自带channel是否有消息
	for s := range clnt.C {
		conn.Write([]byte(s + "\n"))
	}
}
func MakeMsg(clnt *Client, msg string) string {
	buf := "[" + clnt.Addr + "]" + clnt.Name + ": " + msg
	return buf
}
func HandlerConnect(conn net.Conn) {

	defer conn.Close()

	//创建channel判断，用户是否活跃
	hasData := make(chan bool)
	//获取用户网络地址IPaddr
	netAddr := conn.RemoteAddr().String()
	//创建新连接用户结构体
	clnt := &Client{
		C:    make(chan string),
		Name: netAddr,
		Addr: netAddr,
	}
	//将新连接用户，添加到在线用户map中
	onlineMap[netAddr] = clnt

	//创建专门用来给当前用户发送消息的go程
	go WriteMsgToClient(clnt, conn)

	//发送用户上线消息到全局channel中
	message <- MakeMsg(clnt, "login")

	//创建一个channel，用来判断退出状态
	isQuit := make(chan bool)

	//创建一个匿名go程，专门处理用户发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := conn.Read(buf)
			if n == 0 {
				isQuit <- true
				fmt.Printf("检测到客户端：%s退出", clnt.Name)
				return
			}
			if err != nil {
				return
			}
			//将读取的用户消息，保存到msg中，string类型
			msg := string(buf[:n-1]) //-1 回车去掉

			//提取在线用户列表
			if msg == "who" && len(msg) == 3 {
				conn.Write([]byte("online user list:\n"))
				for _, user := range onlineMap {
					userInfo := user.Addr + ":" + user.Name + "\n"
					conn.Write([]byte(userInfo))
				}
				//判断用户发送了改名命令
			} else if len(msg) >= 8 && msg[:6] == "rename" {
				newName := strings.Split(msg, "|")[1]
				clnt.Name = newName       //修改结构体Name
				onlineMap[netAddr] = clnt //更新onlineMap
				conn.Write([]byte("rename successful\n"))
			} else {
				//将读取的用户消息，写入到message
				message <- MakeMsg(clnt, msg)
			}
			hasData <- true
		}
	}()
	//保证不退出
	for {
		select {
		case <-isQuit:
			close(clnt.C)
			delete(onlineMap, netAddr)         //将用户从onlineMap移除
			message <- MakeMsg(clnt, "logout") //写入用户退出到全局channel
			return
		case <-hasData:
			//什么都不做，目的是重制下面case的计时器
		case <-time.After(time.Second * 60): //超时退出
			close(clnt.C)
			delete(onlineMap, netAddr)
			message <- MakeMsg(clnt, "time out leaved")
			return

		}
	}
}

func Manager() {
	//初始化onlineMap
	onlineMap = make(map[string]*Client)

	//循环从message中读取
	for {
		//监听全局channel 是否有数据,有数据存储msg,无数据阻塞
		msg := <-message
		//循环发送消息给所以用户
		for _, clnt := range onlineMap {
			clnt.C <- msg
		}
	}

}
func main() {
	//创建监听套接字
	listener, err := net.Listen("tcp", "127.0.0.1:8000")
	if err != nil {
		fmt.Println("Listen err", err)
		return
	}
	defer listener.Close()

	//创建管理者go程，管理map和全局channel
	go Manager()

	//循环监听客户端连接请求
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept err", err)
			return
		}
		//启动go程处理客户端数据请求
		go HandlerConnect(conn)
	}

}
