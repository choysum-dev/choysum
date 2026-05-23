// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package grpcclient

import (
	"context"

	"github.com/choysum-dev/choysum/pkg/auth/grpcclient/authpb"
	"github.com/choysum-dev/choysum/pkg/grpc/client"
	"google.golang.org/grpc/metadata"
)

func cloneMetadata(md metadata.MD) metadata.MD {
	if md == nil {
		return metadata.MD{}
	}
	out := metadata.MD{}
	for k, vv := range md {
		cp := make([]string, len(vv))
		copy(cp, vv)
		out[k] = cp
	}
	return out
}

// CheckMethodAccess calls auth.User.CheckMethodAccess over gRPC using typed request/response.
//
// Fail-closed is enforced by the caller: errors are returned for the caller to treat as deny.
func CheckMethodAccess(ctx context.Context, companyId, serviceFullName string) (bool, error) {
	conn, err := client.Dial(ctx, "auth.User")
	if err != nil {
		return false, err
	}
	grpcClient := authpb.NewUserClient(conn)

	req := &authpb.User_CheckMethodAccess_Req{
		CompanyId:       companyId,
		ServiceFullName: serviceFullName,
	}
	outCtx := ctx
	if in, ok := metadata.FromIncomingContext(ctx); ok {
		outCtx = metadata.NewOutgoingContext(ctx, cloneMetadata(in))
	}

	resp, err := grpcClient.CheckMethodAccess(outCtx, req)
	if err != nil {
		return false, err
	}
	return resp.GetResult(), nil
}
