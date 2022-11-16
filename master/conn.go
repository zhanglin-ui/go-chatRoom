package master

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

const ConnSendQueueSize = 4096
const MsgSize = 64

type Conn struct {
	addr      string //ip + port
	rawConn   net.Conn
	sendCh    chan []byte   //send msg
	done      chan error    //conn err
	hbTimeout time.Duration //connect timeout
}

func NewConn(c net.Conn, hbTimeout time.Duration) *Conn {
	conn := &Conn{
		addr:      c.RemoteAddr().String(),
		rawConn:   c,
		sendCh:    make(chan []byte, ConnSendQueueSize),
		done:      make(chan error),
		hbTimeout: hbTimeout,
	}

	return conn
}

func (c *Conn) Close() {
	c.rawConn.Close()
}

func (c *Conn) GetConnAddr() string {
	return c.addr
}

func (c *Conn) readCoroutine(ctx context.Context, m *Master) {
	for {
		select {
		case <-ctx.Done():
			log.Fatalln("connect done, exit read routine")
			return
		default:
			if c.hbTimeout > 0 {
				err := c.rawConn.SetReadDeadline(time.Now().Add(c.hbTimeout * time.Second))
				if err != nil {
					log.Println("set read dead line error", err.Error())
					return
				}
			}

			var l uint16
			log.Println(c.rawConn.RemoteAddr().String())
			buf := make([]byte, 2)
			n, err := io.ReadFull(c.rawConn, buf)
			if n == 0 && err == io.EOF {
				continue
			}

			log.Println("read success from tcp")

			if err != nil {
				c.done <- err
				log.Println("read len error", err.Error())
				return
			}
			fmt.Println(buf)

			r := bytes.NewBuffer(buf)
			err = binary.Read(r, binary.BigEndian, &l)
			fmt.Println(l)

			dateBuf := make([]byte, l)
			_, err = io.ReadFull(c.rawConn, dateBuf)
			if err != nil {
				log.Println("read date error", err.Error())
				return
			}
			var user Users
			user.Addr = c.addr
			user.Date = dateBuf
			m.send <- user
		}
	}
}
