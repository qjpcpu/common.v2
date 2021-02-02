package broadcast

import "context"

const (
	terminateMask = 1
)

type payload struct {
	Ctx  context.Context
	Body interface{}
}

func newPayload(ctx context.Context,body interface{}) payload {
	return payload{Ctx: ctx,Body:body}
}

type message struct {
	Next    chan message
	Payload payload
	Flag    uint32
}

func (b message) isTerminateMessage() bool {
	return b.Flag&terminateMask != 0
}

var (
	terminationMsg = message{Payload: payload{ Ctx:context.TODO()}, Flag:terminateMask}
)