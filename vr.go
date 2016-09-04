package vr

import "context"

// VR is Viewstamped Replication state mnet/achine.
type VR interface {
	Process(ctx context.Context, m Msg) error
}
