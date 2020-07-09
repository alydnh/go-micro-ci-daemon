package ci

import (
	"context"
	"github.com/alydnh/go-micro-ci-daemon/proto"
)

type Service struct{}

func (s Service) Version(_ context.Context, _ *proto.EmptyRequest, resp *proto.VersionResponse) error {
	resp.Version = GetVersion()
	return nil
}
