/*
	created by yiqi Jiang
	2016.08.03
*/

//创建10个协程，一次发送10张图片

package main

import (
	//"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"io/ioutil"
	"os/exec"
	"runtime"
	"strconv"
	"time"
	"log"
)

func checkError(err error) {
    if err != nil {
        log.Fatal(err)
    }
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	var (
		host   = "192.168.5.239"     //服务端IP
		port   = "9090"            //服务端端口
		remote = host + ":" + port //构造连接串

		fileName      = "in.mp4" //待发送文件名称
		//mergeFileName = "frame.jpg"   //待合并文件名称
		coroutine     = 50         //协程数量或拆分文件的数量
		bufsize       = 230400     //单次发送数据的大小
	)

	//获取参数信息。
	//参数顺序：
	// 1：待发送文件名
	// 2：待合并文件名
	// 3：单次发送数据大小
	// 4：协程数量或拆分文件数量
	for index, sargs := range os.Args {
		switch index {
		case 1:
			fileName = sargs
			//mergeFileName = sargs
		case 2:
			bufsize, _ = strconv.Atoi(sargs)
		case 3:
			coroutine, _ = strconv.Atoi(sargs)
		}

	}
	//用户输入Ip模块
	/*
	fmt.Printf("请输入服务端IP: ")
	reader := bufio.NewReader(os.Stdin)
	ipdata, _, _ := reader.ReadLine()

	host = string(ipdata)
	remote = host + ":" + port
	*/
    len := 2998
    count := 0
    end := 0
    //var name string
    begintime := time.Now().Unix()
    terminal := len / coroutine	//终止条件
	for count <= terminal {
		c := make(chan string)	//锁机制
		//timeout := make(chan bool, 1)	//初始化timeout，缓冲为1

		if(count == terminal) {
			end = len % coroutine;
		}else {
			end = coroutine
		}
		for i := 0; i < end; i++ {
			index := count * coroutine + i
			go sendFile(fileName, remote, c, index, bufsize)
		}
		//同步等待发送文件的协程，锁机制
		for j := 0; j < end; j++ {
			fmt.Println(<-c)
		}
		/*
		//启动timeout协程
		go func() {
			//监听通道，设置超时防止协程泄露
			select {
			case <-c:
				//同步等待发送文件的协程，锁机制
				for j := 0; j < end; j++ {
					fmt.Println(<-c)
				}
			case <-time.After(time.Duration(3) * time.Second):    //设置超时时间为３ｓ，如果channel　3s钟没有响应，一直阻塞，则报告超时，进行超时处理．
				fmt.Println("超时")
				timeout<-true
				break
			}	
		}()
		*/
		count++
    }
 
	midtime := time.Now().Unix()
	sendtime := midtime - begintime
	fmt.Printf("发送耗时：%d 分 %d 秒 \n", sendtime/60, sendtime%60)

	endtime := time.Now().Unix()

	tot := endtime - begintime
	fmt.Printf("总计耗时：%d 分 %d 秒 \n", tot/60, tot%60)
	

}

/*
*	文件发送方法
*	filename 	视频文件名
*	remote 		服务端IP及端口号（如：192.168.1.8:9090）
*	c			channel,用于同步协程
*	times		图片序号
	size 		图像的大小
 */
func sendFile(filename string, remote string, c chan string, times int, size int) {
	//fmt.Println(remote)
	con, err := net.Dial("tcp", remote)
	//fmt.Println("=========", times, c)
	if err != nil {
		//fmt.Println("get out!")
		fmt.Println("服务器连接失败.")
		os.Exit(-1)
		return
	}
	defer con.Close()
    width := 640
    height := 360
    //use buffer to record frame
    cmd := exec.Command("ffmpeg", "-ss", strconv.Itoa(times), "-i", filename, "-vframes", "1", "-s", fmt.Sprintf("%dx%d", width, height), "-f", "singlejpeg", "-");
    //capture frame from video
    //cmd := exec.Command("ffmpeg", "-i", filename, "-vf", "fps=1", "-s", fmt.Sprintf("%dx%d", width, height), "img11%04d.jpg") 

    //record video
    //cmd := exec.Command("ffmpeg", "-r", "30", "-f", "avfoundation", "-i", "0", "out.mp4")   

    //capture frame from camera 
    //cmd := exec.Command("ffmpeg", "-i", "rtsp://192.168.5.141:8554/all", "-f", "image2", "-vf", "fps=fps=1", "img%3d.jpg")

    var buffer bytes.Buffer
    cmd.Stdout = &buffer
    if cmd.Run() != nil {
        panic("could not generate frame")
    }
    
    l := make([]byte, width * height)
    buffer.Read(l)
    name := "frame" + strconv.Itoa(times) + ".jpg"
    //fmt.Printf("%s\n", name)
    err = ioutil.WriteFile(name, l, 0777)
    checkError(err)	//检测写入错误
    fileName := name
    fl, err := os.OpenFile(fileName, os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("userFile", err)
		return
	}
	stat, err := fl.Stat() //获取文件状态
	if err != nil {
		panic(err)
	}
	var end int64
	end = stat.Size()
	fl.Close()

	fmt.Println(name, "连接已建立.文件发送中...")
	var by [1]byte
	by[0] = byte(times)
	var bys []byte
	databuf := bytes.NewBuffer(bys) //数据缓冲变量
	databuf.Write(by[:])
	databuf.WriteString(name)
	bb := databuf.Bytes()
	// bb := by[:]
	//fmt.Println("-----", bb)
	in, err := con.Write(bb) //向服务器发送当前协程的顺序，代表发送文件的顺序
	if err != nil {
		fmt.Printf("向服务器发送数据错误: %d\n", in)
		os.Exit(0)
	}

	var msg = make([]byte, 1024)  //创建读取服务端信息的切片
	lengthh, err := con.Read(msg) //确认服务器已收到顺序数据
	if err != nil {
		fmt.Printf("读取服务器数据错误.\n", lengthh)
		os.Exit(0)
	}
	//打开待发送文件，准备发送文件数据
	file, err := os.OpenFile(fileName, os.O_RDWR, 0666)
	defer file.Close()
	if err != nil {
		fmt.Println(fileName, "-文件打开错误.")
		os.Exit(0)
	}
	var begin int64 = 0
	file.Seek(begin, 0) //设定读取文件的位置

	buf := make([]byte, size) //创建用于保存读取文件数据的切片

	var sendDtaTolNum int = 0 //记录发送成功的数据量（Byte）
	//读取并发送数据
	for i := begin; int64(i) < end; i += int64(size) {
		length, err := file.Read(buf) //读取数据到切片中
		if err != nil {
			fmt.Println("读文件错误", i, times, end)
		}

		//判断读取的数据长度与切片的长度是否相等，如果不相等，表明文件读取已到末尾
		if length == size {
			//判断此次读取的数据是否在当前协程读取的数据范围内，如果超出，则去除多余数据，否则全部发送
			if int64(i)+int64(size) >= end {
				sendDataNum, err := con.Write(buf[:size-int((int64(i)+int64(size)-end))])
				if err != nil {
					fmt.Printf("向服务器发送数据错误: %d\n", sendDataNum)
					os.Exit(0)
				}
				sendDtaTolNum += sendDataNum
			} else {
				sendDataNum, err := con.Write(buf)
				if err != nil {
					fmt.Printf("向服务器发送数据错误: %d\n", sendDataNum)
					os.Exit(0)
				}
				sendDtaTolNum += sendDataNum
			}

		} else {
			sendDataNum, err := con.Write(buf[:length])
			if err != nil {
				fmt.Printf("向服务器发送数据错误: %d\n", sendDataNum)
				os.Exit(0)
			}
			sendDtaTolNum += sendDataNum
		}
	}
	//读取服务器端信息，确认服务端已接收数据
	lengths, err := con.Read(msg)
	if err != nil {
		fmt.Printf("读取服务器数据错误.\n", lengths)
		os.Exit(0)
	}
	//str := string(msg[0:lengths])
	//fmt.Println("传输完成（服务端信息）： ", str)
	fmt.Println(name, "发送数据(Byte)：", sendDtaTolNum)
	c <- strconv.Itoa(times) + " 协程退出"
	
}
