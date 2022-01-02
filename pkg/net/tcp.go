package net

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tfes-dev/tfes/pkg/routing"
	"github.com/tfes-dev/tfes/pkg/schemas"
	"github.com/tfes-dev/tfes/pkg/utils"
	"io"
	"log"
	"net"
)

type TcpHandlerPool struct {
	// Clients only has a list of clients who have subscribed to at least one topic
	// Other clients who are simply connected need not be tracked, for now.
	Clients []*schemas.ClientConnection

	msgsToPeers   chan *schemas.Message
	msgsFromPeers chan *schemas.Message
	config        *schemas.Config
}

func NewTcpHandlerPool(config *schemas.Config, msgsToPeers chan *schemas.Message, msgsFromPeers chan *schemas.Message) *TcpHandlerPool {
	return &TcpHandlerPool{
		config:        config,
		msgsToPeers:   msgsToPeers,
		msgsFromPeers: msgsFromPeers,
		Clients:       make([]*schemas.ClientConnection, 0),
	}
}

func (pool *TcpHandlerPool) Start() error {
	go pool.listenToInbox()

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", pool.config.Server.Address, pool.config.Server.Port))
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

func (pool *TcpHandlerPool) listenToInbox() {
	for {
		select {
		case msg := <-pool.msgsFromPeers:
			log.Println("Going to broadcast peer message to clients:", msg.Kind)
			for _, _cc := range pool.Clients {
				go func(_cc *schemas.ClientConnection) {
					checkAndSendToClient(msg, _cc)
				}(_cc)
			}
		}
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
			utils.WriteToBufio(writer, response)
		}
	}
}

// Should send back ACK packets
func (pool *TcpHandlerPool) handleIncomingMessage(data string, cc *schemas.ClientConnection) *schemas.Message {
	var msg schemas.Message
	err := json.Unmarshal([]byte(data), &msg)
	if err != nil {
		return utils.ReturnErrorAck(err)
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
		return utils.ReturnErrorAck(errors.New("unknown message kind"))
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
	return utils.ReturnSuccessAck()
}

func (pool *TcpHandlerPool) handlePublish(msg *schemas.Message, cc *schemas.ClientConnection) *schemas.Message {

	for _, _cc := range pool.Clients {
		go func(_cc *schemas.ClientConnection) {
			checkAndSendToClient(msg, _cc)
		}(_cc)
	}

	pool.msgsToPeers <- msg
	return utils.ReturnSuccessAck()
}

func checkAndSendToClient(msg *schemas.Message, cc *schemas.ClientConnection) {
	for _, subs := range cc.SubscribedSubjects {
		if routing.MatchSubject(msg.Publish.Subject, subs) {
			m := &schemas.Message{
				Kind:   schemas.KindBounty,
				Header: msg.Header,
				Bounty: msg.Publish.ToBounty(),
			}
			utils.WriteToIo(cc.TcpConnection, m)
		}
	}
}

func (pool *TcpHandlerPool) handleSubscribe(msg *schemas.Message, cc *schemas.ClientConnection) *schemas.Message {
	subscribe := msg.Subscribe
	if len(cc.SubscribedSubjects) == 0 {
		cc.SubscribedSubjects = make([]string, 0)
	}
	cc.SubscribedSubjects = append(cc.SubscribedSubjects, subscribe.Subject)
	pool.msgsToPeers <- msg
	return utils.ReturnSuccessAck()
}

func (pool *TcpHandlerPool) handleUnsubscribe(msg *schemas.Message, cc *schemas.ClientConnection) *schemas.Message {
	unsubscribe := msg.Unsubscribe
	cc.SubscribedSubjects = utils.RemoveItem(cc.SubscribedSubjects, unsubscribe.Subject)
	pool.msgsToPeers <- msg
	return utils.ReturnSuccessAck()
}
