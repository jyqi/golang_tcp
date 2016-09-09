
//对每个请求由一个单独的协程进行处理，文件接收完成后由一个协负责将所有接收的数据合并成一个有效文件
package main
import (
	"log"
	"bytes"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"net/http"
	_"net/http/pprof"
	"math/rand"
	//"time"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	var (
		// host   = "192.168.1.5"	//如果写locahost或127.0.0.1则只能本地访问。
		port = "9090"
		// remote = host + ":" + port

		remote = ":" + port //此方式本地与非本地都可访问
	)

	fmt.Println("服务器初始化... (Ctrl-C 停止)")

	lis, err := net.Listen("tcp", remote)
	defer lis.Close()

	if err != nil {
		fmt.Println("监听端口发生错误: ", remote)
		os.Exit(-1)
	}

	pool := make([][]byte, 50)
	buffer := make(chan []byte, 5)
	var b []byte
	for {

		conn, err := lis.Accept()
		if err != nil {
			fmt.Println("客户端连接错误: ", err.Error())
			// os.Exit(0)
			continue
		}
		select {
		case b = <-buffer:
			fmt.Println("读buffer")
		default:
			b = make([]byte, 1024 * 1024)
			fmt.Println("buffer为空，创建")
		}
		i := rand.Intn(len(pool))
		if pool[i] != nil {
			select {
			case buffer <- pool[i]:
				pool[i] = nil
				fmt.Println("写buffer", i)
			default:
				fmt.Println("直接使用buffer")
			}
		}
		pool[i] = b
		//调用文件接收方法
		go receiveFile(conn, b)
	}
}

/*
*	文件接收方法
*	con 连接成功的客户端连接
 */
func receiveFile(con net.Conn, data []byte) {
	var (
		res          string
		tempFileName string                    //保存临时文件名称
		//data         = make([]byte, 1024*1024) //用于保存接收的数据的切片
		by           []byte
		databuf      = bytes.NewBuffer(by) //数据缓冲变量
		fileNum      int                   //当前协程接收的数据在原文件中的位置
	)
    defer con.Close()
	fmt.Println("新建立连接: ", con.RemoteAddr())
	j := 0 //标记接收数据的次数
	for {
		length, err := con.Read(data)
		if err != nil {
			//res = strconv.Itoa(fileNum) + " 接收完成"
			//con.Write([]byte(res))
			// writeend(tempFileName, databuf.Bytes())
			da := databuf.Bytes()
			// fmt.Println("over", fileNum, len(da))
			fmt.Printf("客户端 %v 已断开. %2d %d \n", con.RemoteAddr(), fileNum, len(da))
			//if data.alloc != nil {
			//	cmem.Free(data.alloc)
			//	data.alloc = nil
			//}
			return
		}
		if 0 == j {
			res = string(data[0:8])
			//fmt.Println("%s\n", res)
			fileNum = int(data[0])
			tempFileName = string(data[1:length])
			fmt.Println("创建文件：", tempFileName)
			fout, err := os.Create(tempFileName)
			if err != nil {
				fmt.Println("创建文件错误", tempFileName)
				return
			}
			fout.Close()
		} else {
			writeTempFileEnd(tempFileName, data[0:length])
		}
		res = strconv.Itoa(fileNum) + " 接收完成"
		con.Write([]byte(res))
		j++
	}

}

/*
*	把数据写入指定的文件中
*
*	fileName	文件名
*	data 		接收的数据
 */
func writeTempFileEnd(fileName string, data []byte) {
	// fmt.Println("追加：", name)
	tempFile, err := os.OpenFile(fileName, os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		// panic(err)
		fmt.Println("打开文件错误", err)
		return
	}
	defer tempFile.Close()
	tempFile.Write(data)
}
