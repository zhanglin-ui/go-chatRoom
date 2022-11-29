package client

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Client struct {
	conn net.Conn

	msg     chan Msg
	sendMsg chan string
	quit    chan int
	errChan chan error
}

type Msg struct {
	addr []byte
	msg  []byte
}

var Clients *Client

func Initialize() *Client {
	client := &Client{
		msg:     make(chan Msg),
		quit:    make(chan int),
		sendMsg: make(chan string),
		errChan: make(chan error),
	}

	return client
}

func (c *Client) Start() {
	conn, err := net.Dial("tcp", os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}
	c.conn = conn
	defer func() {
		c.conn.Close()
	}()

	go c.ListenKeyBoard()
	go c.SendMsg()
	go c.ReceiveMsg()
	go c.PrintMsg()

	for {
		select {
		case <-c.quit:
			return
		case <-c.errChan:
			return
		}
	}
}

func (c *Client) ListenKeyBoard() {
	inputReader := bufio.NewReader(os.Stdin)
	for {
		input, _ := inputReader.ReadString('\n')
		inputInfo := strings.Trim(input, "\r\n")
		if strings.ToUpper(inputInfo) == "Q" {
			c.quit <- 1
			return
		} else {
			if len(inputInfo) == 0 {
				continue
			}
			log.Printf("                              【我】%s\n", inputInfo)
			c.sendMsg <- inputInfo
		}
		time.Sleep(time.Millisecond * 10)
	}
}

func (c *Client) SendMsg() {
	for {
		select {
		case info := <-c.sendMsg:
			pkt, err := Encode(info)
			if err != nil {
				log.Println("encode msg error", err.Error())
				c.errChan <- err
			}

			if _, err = c.conn.Write(pkt); err != nil {
				log.Println("send msg to server error", err.Error())
				c.errChan <- err
			}
		default:
			time.Sleep(time.Millisecond * 10)
		}
	}
}

func (c *Client) ReceiveMsg() {
	for {
		var l uint16
		var ipLen uint16
		buf := make([]byte, 2)
		n, err := io.ReadFull(c.conn, buf)
		if n == 0 && err == io.EOF {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		if err != nil {
			log.Println("read len error", err.Error())
			c.errChan <- err
		}

		r := bytes.NewBuffer(buf)
		err = binary.Read(r, binary.BigEndian, &l)
		if err != nil {
			log.Println("read length from buf error")
			c.errChan <- err
		}

		ipBuf := make([]byte, 2)
		n, err = io.ReadFull(c.conn, ipBuf)
		if n == 0 && err == io.EOF {
			log.Println("read ip length error")
			c.errChan <- err
		}

		ipBuffer := bytes.NewBuffer(ipBuf)
		err = binary.Read(ipBuffer, binary.BigEndian, &ipLen)
		if err != nil {
			log.Println("read ip str error", err.Error())
			c.errChan <- err
		}

		ipStrBuf := make([]byte, ipLen)
		headBuf := make([]byte, 5)
		dateBuf := make([]byte, l-10-ipLen)
		tailBuf := make([]byte, 5)

		n, err = io.ReadFull(c.conn, ipStrBuf)
		if err != nil {
			log.Println("read ip error", err.Error())
			c.errChan <- err
		}

		n, err = io.ReadFull(c.conn, headBuf)
		if err != nil {
			log.Println("read head error", err.Error())
			c.errChan <- err
		}
		n, err = io.ReadFull(c.conn, dateBuf)
		if err != nil {
			log.Println("read date error", err.Error())
			c.errChan <- err
		}
		n, err = io.ReadFull(c.conn, tailBuf)
		if err != nil {
			log.Println("read tail error", err.Error())
			c.errChan <- err
		}
		c.msg <- Msg{addr: ipStrBuf, msg: dateBuf}
	}
}

func (c *Client) PrintMsg() {
	for {
		select {
		case str := <-c.msg:
			log.Printf("【%s】%s", string(str.addr), string(str.msg))
		default:
			continue
		}
	}
}

func Encode(msg string) ([]byte, error) {
	buffer := new(bytes.Buffer)

	var l uint16
	l = uint16(len(msg)) + 5 + 5

	Magic := [5]byte{'w', 'm', 's', 'g', 'b'}
	End := [5]byte{'w', 'm', 's', 'g', 'e'}

	err := binary.Write(buffer, binary.BigEndian, l)
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, Magic)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buffer, binary.BigEndian, []byte(msg))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.BigEndian, End)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
