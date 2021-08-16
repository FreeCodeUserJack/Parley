package context_utils

import (
	"context"

	"github.com/google/uuid"
)

type TraceId string
type ClientId string

func GetTraceAndClientIds(ctx context.Context) []string {
	var traceId, clientId string
	if buf := ctx.Value(TraceId("traceId")); buf != nil {
		if uuid, ok := buf.(uuid.UUID); !ok {
			traceId = ""
		} else {
			traceId = uuid.String()
		}

	}
	if buf := ctx.Value(ClientId("clientId")); buf != nil {
		clientId = buf.(string)
	}

	return []string{"traceId:" + traceId, "cliendId:" + clientId}
}
