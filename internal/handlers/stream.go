package handlers

import (
	"context"
	_ "net/http/pprof"

	pb "gorsovet/cmd/proto"
)

func (gk *GkeeperService) UpLoadFile(ctx context.Context, req *pb.UpLoadRequest) (resp *pb.UpLoadResponse, err error) {

}
