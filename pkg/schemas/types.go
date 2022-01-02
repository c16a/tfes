package schemas

import "net"

const (
	KindConnect     = "schema.tfes.v1.connect"
	KindAck         = "schema.tfes.v1.ack"
	KindPublish     = "schema.tfes.v1.publish"
	KindSubscribe   = "schema.tfes.v1.subscribe"
	KindUnsubscribe = "schema.tfes.v1.unsubscribe"
	KindBounty      = "schema.tfes.v1.bounty"
)

type Message struct {
	Kind        string       `json:"kind"`
	Header      *Header      `json:"header,omitempty"`
	Publish     *Publish     `json:"publish,omitempty"`
	Subscribe   *Subscribe   `json:"subscribe,omitempty"`
	Unsubscribe *Unsubscribe `json:"unsubscribe,omitempty"`
	Connect     *Connect     `json:"connect,omitempty"`
	Ack         *Ack         `json:"ack,omitempty"`
	Bounty      *Bounty      `json:"bounty,omitempty"`
}

type Connect struct {
	User         string `json:"user"`
	Password     string `json:"password"`
	Token        string `json:"token"`
	SuppressAcks bool   `json:"suppress_acks"`
	ClientID     string `json:"client_id"`
	ClientGroup  string `json:"client_group"`
}

// Ack is the acknowledgement sent by server to client
type Ack struct {
	Ok          bool   `json:"ok"`
	Description string `json:"message,omitempty"`
}

// Publish is sent by client to server
type Publish struct {
	Subject string      `json:"subject"`  // The Subject to which the message must be delivered
	ReplyTo string      `json:"reply_to"` // ReplyTo is the subject to which the reply of the message needs to be sent
	Body    interface{} `json:"body"`     // Body is the custom data that the client wants to send over
}

func (publish *Publish) ToBounty() *Bounty {
	return &Bounty{
		Subject: publish.Subject,
		ReplyTo: publish.ReplyTo,
		Body:    publish.Body,
	}
}

type Subscribe struct {
	Subjects []string `json:"subjects"` // The list of Subjects to subscribe to
}

type Unsubscribe struct {
	Subjects []string `json:"subjects"` // The list of Subjects to unsubscribe from
}

type Bounty struct {
	Subject string      `json:"subject"`            // The Subject to which the message is intended
	ReplyTo string      `json:"reply_to,omitempty"` // The ReplyTo subject
	Body    interface{} `json:"body,omitempty"`
}

const (
	ConnectionTypeTcp = "tcp"
)

type ClientConnection struct {
	ClientUri          string   // ClientUri is a concatenation of ClientID:ClientGroup
	SuppressAcks       bool     // SuppressAcks suppresses acknowledgements if client wants to disable them
	SubscribedSubjects []string // SubscribedSubjects is the list of subjects the connection is subscribed to receive
	ConnectionType     string   // ConnectionType is the type of connection
	TcpConnection      net.Conn // TcpConnection is the net.Conn object for TCP clients. This might be other kinds of connection objects for other connection types.
}
