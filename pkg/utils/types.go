package utils

import "github.com/tfes-dev/tfes/pkg/schemas"

func ReturnErrorAck(err error) *schemas.Message {
	return &schemas.Message{
		Kind: schemas.KindAck,
		Ack:  &schemas.Ack{Ok: false, Description: err.Error()},
	}
}

func ReturnSuccessAck() *schemas.Message {
	return &schemas.Message{
		Kind: schemas.KindAck,
		Ack:  &schemas.Ack{Ok: true},
	}
}
