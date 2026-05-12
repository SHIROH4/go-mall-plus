// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"context"
	"errors"

	"ecommerce-demo/app/gateway/internal/svc"
	"ecommerce-demo/app/gateway/internal/types"
	"ecommerce-demo/app/user/pb"
	"ecommerce-demo/common/ctxutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserInfoLogic) GetUserInfo() (resp *types.GetUserInfoResp, err error) {
	// 使用统一的 ctxutil 获取用户 ID
	userId, err := ctxutil.GetUserId(l.ctx)
	if err != nil {
		return nil, errors.New("用户未登录")
	}

	// 调用 User RPC 获取信息
	rpcResp, err := l.svcCtx.UserRpc.GetUserInfo(l.ctx, &pb.GetUserInfoReq{
		Id: userId,
	})
	if err != nil {
		return nil, err
	}

	return &types.GetUserInfoResp{
		Id:       rpcResp.Id,
		Username: rpcResp.Username,
	}, nil
}
