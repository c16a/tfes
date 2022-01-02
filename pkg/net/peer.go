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

type PeerServer struct {
	Peers           []*schemas.PeerConnection
	config          *schemas.Config
	msgsFromClients chan *schemas.Message
	msgsToClients   chan *schemas.Message
}

func NewPeerListener(config *schemas.Config, msgsFromClients chan *schemas.Message, msgsToClients chan *schemas.Message) *PeerServer {
	return &PeerServer{
		config:          config,
		Peers:           make([]*schemas.PeerConnection, 0),
		msgsFromClients: msgsFromClients,
		msgsToClients:   msgsToClients,
	}
}

func (p *PeerServer) Start() error {
	go p.dialPeers()
	go p.listenToInbox()

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

func (p *PeerServer) listenToInbox() {
	for {
		select {
		case msg := <-p.msgsFromClients:
			log.Println("Received peer inbox msg:", msg.Kind)
			p.notifyPeers(msg)
		}
	}
}

func (p *PeerServer) dialPeers() {
	for _, route := range p.config.Cluster.Routes {
		if len(route.Url) > 0 {
			log.Println("Dialing peer:", route)
			conn, err := net.Dial("tcp", route.Url)
			if err == nil {
				pc := &schemas.PeerConnection{
					PeerName:      route.Name,
					PeerUri:       route.Url,
					TcpConnection: conn,
				}
				p.Peers = append(p.Peers, pc)
				log.Println("Connected to peer", route)
				p.sendPeerConnectPacket(pc)
				go p.handleDialedUpConnection(pc)
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
	utils.WriteToIo(connection.TcpConnection, msg)
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

func (p *PeerServer) handleDialedUpConnection(pc *schemas.PeerConnection) {
	reader := bufio.NewReader(pc.TcpConnection)
	cc := &schemas.PeerConnection{
		PeerName:      pc.PeerName,
		PeerUri:       pc.PeerUri,
		TcpConnection: pc.TcpConnection,
	}
	for {
		data, err := reader.ReadString('\n')

		if err != nil {
			if err == io.EOF {
				pc.TcpConnection.Close()
				return
			}
			continue
		}

		p.handleIncomingMessage(data, cc)
	}
}

// Should send back ACK packets
func (p *PeerServer) handleIncomingMessage(data string, cc *schemas.PeerConnection) *schemas.Message {
	var msg schemas.Message
	err := json.Unmarshal([]byte(data), &msg)
	if err != nil {
		return utils.ReturnErrorAck(err)
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
	case schemas.KindUnsubscribe:
		fn = p.handleUnsubscribe
		break
	default:
		return utils.ReturnErrorAck(errors.New("unknown message kind"))
	}

	return fn(&msg, cc)
}

func (p *PeerServer) notifyPeers(msg *schemas.Message) {
	for _, peer := range p.Peers {
		if msg.Kind == schemas.KindSubscribe || msg.Kind == schemas.KindUnsubscribe {
			log.Println("Notifying peer:", peer.PeerUri)
			utils.WriteToIo(peer.TcpConnection, msg)
		} else {
			log.Printf("Checking %s for subs: %v\n", peer.PeerName, peer.InterestedSubjects)
			for _, sub := range peer.InterestedSubjects {
				if routing.MatchSubject(msg.Publish.Subject, sub) {
					log.Println("Notifying peer:", peer.PeerUri)
					utils.WriteToIo(peer.TcpConnection, msg)
				}
			}
		}
	}
}

func (p *PeerServer) handlePublish(message *schemas.Message, connection *schemas.PeerConnection) *schemas.Message {
	p.msgsToClients <- message
	return utils.ReturnSuccessAck()
}

func (p *PeerServer) handleSubscribe(message *schemas.Message, connection *schemas.PeerConnection) *schemas.Message {
	subscribe := message.Subscribe
	log.Println("Received peer subscribe for:", message.Subscribe.Subject)
	for _, peer := range p.Peers {
		if peer.PeerName == connection.PeerName {
			peer.InterestedSubjects = append(peer.InterestedSubjects, subscribe.Subject)
		}
	}
	return utils.ReturnSuccessAck()
}

func (p *PeerServer) handleUnsubscribe(message *schemas.Message, connection *schemas.PeerConnection) *schemas.Message {
	subscribe := message.Subscribe
	peerUrl := connection.TcpConnection.RemoteAddr().String()
	for _, peer := range p.Peers {
		if peer.PeerUri == peerUrl {
			peer.InterestedSubjects = utils.RemoveItem(peer.InterestedSubjects, subscribe.Subject)
		}
	}
	return utils.ReturnSuccessAck()
}

func (p *PeerServer) handlePeerConnect(message *schemas.Message, connection *schemas.PeerConnection) *schemas.Message {
	log.Println("Received incoming peer connection")
	pc := message.PeerConnect
	connection.PeerName = pc.PeerName
	connection.PeerUri = pc.AdvertiseAddr

	p.Peers = append(p.Peers, connection)
	return utils.ReturnSuccessAck()
}
