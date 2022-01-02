package net

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tfes-dev/tfes/pkg/routing"
	"github.com/tfes-dev/tfes/pkg/schemas"
	"io"
	"log"
	"net"
)

type TcpHandlerPool struct {
	// Clients only has a list of clients who have subscribed to at least one topic
	// Other clients who are simply connected need not be tracked, for now.
	Clients []*schemas.ClientConnection
}

func NewTcpHandlerPool() *TcpHandlerPool {
	return &TcpHandlerPool{
		Clients: make([]*schemas.ClientConnection, 0),
	}
}

func (pool *TcpHandlerPool) Start() error {
	listener, err := net.Listen("tcp", ":5555")
	if err != nil {
		return err
	}

	for {
		c, err := listener.Accept()
		if err != nil {
			// If current connection didn't succeed to establish, move on
			continue
		}
		go pool.handleConnection(c)
	}
}

func (pool *TcpHandlerPool) handleConnection(conn net.Conn) {
	reader, writer := bufio.NewReader(conn), bufio.NewWriter(conn)
	cc := &schemas.ClientConnection{
		ConnectionType: schemas.ConnectionTypeTcp,
		TcpConnection:  conn,
	}
	for {
		data, err := reader.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				conn.Close()
				return
			}
			continue
		}

		response := pool.handleIncomingMessage(data, cc)

		if !cc.SuppressAcks {
			r, err := json.Marshal(response)
			if err != nil {
				log.Println(err)
			}
			writer.Write(append(r, '\n'))
			writer.Flush()
		}
	}
}

// Should send back ACK packets
func (pool *TcpHandlerPool) handleIncomingMessage(data string, cc *schemas.ClientConnection) *schemas.Message {
	var msg schemas.Message
	err := json.Unmarshal([]byte(data), &msg)
	if err != nil {
		return returnErrorAck(err)
	}

	var fn func(*schemas.Message, *schemas.ClientConnection) *schemas.Message

	switch msg.Kind {
	case schemas.KindConnect:
		fn = pool.handleConnect
		break
	case schemas.KindPublish:
		fn = pool.handlePublish
		break
	case schemas.KindSubscribe:
		fn = pool.handleSubscribe
		break
	case schemas.KindUnsubscribe:
		fn = pool.handleUnsubscribe
		break
	default:
		return returnErrorAck(errors.New("unknown message kind"))
	}

	return fn(&msg, cc)
}

func (pool *TcpHandlerPool) handleConnect(msg *schemas.Message, cc *schemas.ClientConnection) *schemas.Message {
	connect := msg.Connect
	if len(connect.ClientGroup) > 0 {
		cc.ClientUri = fmt.Sprintf("%s:%s", connect.ClientID, connect.ClientGroup)
	} else {
		cc.ClientUri = connect.ClientID
	}
	cc.SuppressAcks = connect.SuppressAcks
	pool.Clients = append(pool.Clients, cc)
	log.Println("Connected new client")
	return returnSuccessAck()
}

func (pool *TcpHandlerPool) handlePublish(msg *schemas.Message, cc *schemas.ClientConnection) *schemas.Message {
	for _, _cc := range pool.Clients {
		go func(_cc *schemas.ClientConnection) {
			checkAndSendToClient(msg, _cc)
		}(_cc)
	}
	return returnSuccessAck()
}

func checkAndSendToClient(msg *schemas.Message, cc *schemas.ClientConnection) {
	for _, subs := range cc.SubscribedSubjects {
		if routing.MatchSubject(msg.Publish.Subject, subs) {
			m := &schemas.Message{
				Kind:   schemas.KindBounty,
				Header: msg.Header,
				Bounty: msg.Publish.ToBounty(),
			}

			r, _ := json.Marshal(m)
			writer := bufio.NewWriter(cc.TcpConnection)
			writer.Write(append(r, '\n'))
			writer.Flush()
		}
	}
}

func (pool *TcpHandlerPool) handleSubscribe(msg *schemas.Message, cc *schemas.ClientConnection) *schemas.Message {
	subscribe := msg.Subscribe
	if len(cc.SubscribedSubjects) == 0 {
		cc.SubscribedSubjects = make([]string, 0)
	}
	cc.SubscribedSubjects = append(cc.SubscribedSubjects, subscribe.Subject)
	return returnSuccessAck()
}

func (pool *TcpHandlerPool) handleUnsubscribe(msg *schemas.Message, cc *schemas.ClientConnection) *schemas.Message {
	unsubscribe := msg.Unsubscribe
	index := -1
	for i, subject := range cc.SubscribedSubjects {
		if subject == unsubscribe.Subject {
			index = i
		}
	}
	if index >= -1 {
		cc.SubscribedSubjects = remove(cc.SubscribedSubjects, index)
	}
	return returnSuccessAck()
}

func remove(slice []string, i int) []string {
	copy(slice[i:], slice[i+1:])
	return slice[:len(slice)-1]
}

func returnErrorAck(err error) *schemas.Message {
	return &schemas.Message{
		Kind: schemas.KindAck,
		Ack:  &schemas.Ack{Ok: false, Description: err.Error()},
	}
}

func returnSuccessAck() *schemas.Message {
	return &schemas.Message{
		Kind: schemas.KindAck,
		Ack:  &schemas.Ack{Ok: true},
	}
}
