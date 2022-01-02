package net

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tfes-dev/tfes/pkg/schemas"
	"io"
	"log"
	"net"
)

type PeerServer struct {
	Peers  []*schemas.PeerConnection
	config *schemas.Config
}

func NewPeerListener(config *schemas.Config) *PeerServer {
	return &PeerServer{config: config, Peers: make([]*schemas.PeerConnection, 0)}
}

func (p *PeerServer) Start() error {
	go p.dialPeers()

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", p.config.Cluster.Address, p.config.Cluster.Port))
	if err != nil {
		return err
	}

	for {
		c, err := listener.Accept()
		if err != nil {
			// If current connection didn't succeed to establish, move on
			continue
		}
		go p.handleConnection(c)
	}
}

func (p *PeerServer) dialPeers() {
	for _, peerUri := range p.config.Cluster.Routes {
		if len(peerUri) > 0 {
			log.Println("Dialing peer:", peerUri)
			conn, err := net.Dial("tcp", peerUri)
			if err == nil {
				pc := &schemas.PeerConnection{
					PeerUri:       peerUri,
					TcpConnection: conn,
				}
				p.Peers = append(p.Peers, pc)
				log.Println("Connected to peer", peerUri)
				p.sendPeerConnectPacket(pc)
				go p.handleConnection(pc.TcpConnection)
			}
		}
	}
}

func (p *PeerServer) sendPeerConnectPacket(connection *schemas.PeerConnection) {
	msg := &schemas.Message{
		Kind: schemas.KindPeerConnect,
		PeerConnect: &schemas.PeerConnect{
			PeerName:      p.config.Server.Name,
			AdvertiseAddr: fmt.Sprintf("%s:%d", p.config.Cluster.Address, p.config.Cluster.Port),
		},
	}
	r, err := json.Marshal(msg)
	if err != nil {
		log.Println(err)
	}
	writer := bufio.NewWriter(connection.TcpConnection)
	writer.Write(append(r, '\n'))
	writer.Flush()
}

func (p *PeerServer) handleConnection(conn net.Conn) {
	reader := bufio.NewReader(conn)
	cc := &schemas.PeerConnection{
		TcpConnection: conn,
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

		p.handleIncomingMessage(data, cc)
	}
}

// Should send back ACK packets
func (p *PeerServer) handleIncomingMessage(data string, cc *schemas.PeerConnection) *schemas.Message {
	log.Println("Received incoming message from peer")
	var msg schemas.Message
	err := json.Unmarshal([]byte(data), &msg)
	if err != nil {
		return returnErrorAck(err)
	}

	var fn func(*schemas.Message, *schemas.PeerConnection) *schemas.Message

	switch msg.Kind {
	case schemas.KindPeerConnect:
		fn = p.handlePeerConnect
		break
	case schemas.KindPublish:
		fn = p.handlePublish
		break
	case schemas.KindSubscribe:
		fn = p.handleSubscribe
		break
	default:
		return returnErrorAck(errors.New("unknown message kind"))
	}

	return fn(&msg, cc)
}

func (p *PeerServer) notifyPub(msg *schemas.Message) {
	for _, peer := range p.Peers {
		log.Println("Notifying peer:", peer.PeerUri)
		r, err := json.Marshal(msg)
		if err != nil {
			log.Println(err)
		}
		writer := bufio.NewWriter(peer.TcpConnection)
		_, err = writer.Write(append(r, '\n'))
		if err != nil {
			log.Println(err)
		}
		err = writer.Flush()
		if err != nil {
			log.Println(err)
		}
	}
}

func (p *PeerServer) handlePublish(message *schemas.Message, connection *schemas.PeerConnection) *schemas.Message {
	log.Println("Received peer publish")
	return returnSuccessAck()
}

func (p *PeerServer) handleSubscribe(message *schemas.Message, connection *schemas.PeerConnection) *schemas.Message {
	return returnSuccessAck()
}

func (p *PeerServer) handlePeerConnect(message *schemas.Message, connection *schemas.PeerConnection) *schemas.Message {
	log.Println("Received incoming peer connection")
	pc := message.PeerConnect
	p.Peers = append(p.Peers, &schemas.PeerConnection{
		PeerName:      pc.PeerName,
		PeerUri:       pc.AdvertiseAddr,
		TcpConnection: connection.TcpConnection,
	})
	return returnSuccessAck()
}
