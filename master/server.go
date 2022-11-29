package master

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Master struct {
	hbTimeout time.Duration

	listen net.Listener
	users  []net.Conn

	onConnect        func(*Conn)
	onDisconnect     func(*Conn, error)
	sendMsgBoardCast func(master *Master)

	stopCh chan error
	send   chan Users
}

type Users struct {
	Addr string
	Date []byte
}

var Server *Master

func MasterInit() *Master {
	MasterServer := &Master{
		hbTimeout:        1000,
		stopCh:           make(chan error),
		send:             make(chan Users),
		onConnect:        defaultConnectHandler,
		onDisconnect:     defaultDisconnectHandler,
		sendMsgBoardCast: defaultSendMsgBoardCast,
	}
	return MasterServer
}

func defaultConnectHandler(conn *Conn) {
	log.Println("Connect coming:", conn.addr)
}

func defaultDisconnectHandler(conn *Conn, err error) {
	log.Println("Disconnect", conn.addr, err.Error())

}

func defaultSendMsgBoardCast(m *Master) {
	for {
		select {
		case msg := <-m.send:
			log.Println(msg.Addr, msg.Date)
			for _, conn := range m.users {
				if conn.RemoteAddr().String() != msg.Addr {
					addr := strings.Split(msg.Addr, ":")
					addr0 := []byte(addr[0])
					ipLen := uint16(len(addr0))
					l := uint16(len(msg.Date)) + ipLen
					buffer := new(bytes.Buffer)
					_ = binary.Write(buffer, binary.BigEndian, l)
					_ = binary.Write(buffer, binary.BigEndian, ipLen)
					_ = binary.Write(buffer, binary.BigEndian, addr0)
					_ = binary.Write(buffer, binary.LittleEndian, msg.Date)
					msg.Date = buffer.Bytes()
					if _, err := conn.Write(msg.Date); err != nil {
						log.Println("send msg error,", err.Error())
						continue
					}
					log.Println("send msg to", msg.Addr, msg.Date)
				}
			}
		default:
			continue
		}
	}
}

func (m *Master) Start() {
	listen, err := net.Listen("tcp", os.Args[2])
	if err != nil {
		fmt.Println("listen failed, err:", err)
		return
	}
	m.listen = listen
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		cancel()
		listen.Close()
	}()

	go m.acceptHandler(ctx)
	go m.sendMsgBoardCast(m)

	for {
		select {
		case <-m.stopCh:
			continue
		}
	}
}

func (m *Master) acceptHandler(ctx context.Context) {
	for {
		c, err := m.listen.Accept()
		if err != nil {
			m.stopCh <- err
			fmt.Println(err)
			return
		}

		go m.connectHandler(ctx, c)
	}
}

func (m *Master) connectHandler(ctx context.Context, c net.Conn) {
	old := false
	for i, conn := range m.users {
		if conn.RemoteAddr().String() == c.RemoteAddr().String() {
			m.users[i] = c
			old = true
		}
	}

	if !old {
		m.users = append(m.users, c)
	}

	connCtx, cancel := context.WithCancel(ctx)
	conn := NewConn(c, m.hbTimeout)
	defer func() {
		cancel()
		conn.Close()
		m.deleteUsers(conn.addr)
	}()

	go conn.readCoroutine(connCtx, m)
	if m.onConnect != nil {
		m.onConnect(conn)
	}

	for {
		select {
		case err := <-conn.done:
			if m.onDisconnect != nil {
				m.onDisconnect(conn, err)
				m.deleteUsers(conn.addr)
				return
			}
		default:
			continue
		}
	}
}

func (m *Master) deleteUsers(addr string) {
	for i, conn := range m.users {
		if conn.RemoteAddr().String() == addr {
			m.users = append(m.users[:i], m.users[i+1:]...)
		}
	}
}
