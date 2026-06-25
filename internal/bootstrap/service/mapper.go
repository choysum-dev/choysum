// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package service

import (
	bootstrappb "github.com/choysum-dev/choysum/internal/bootstrap/proto/bootstrappb"
)

func toInitializeResponse(op operationSnapshot) *bootstrappb.Workspace_Initialize_Resp {
	return &bootstrappb.Workspace_Initialize_Resp{
		Accepted:        true,
		OperationId:     op.OperationID,
		NextPollAfterMs: op.NextPollAfterMs,
		State:           op.State,
	}
}

func toStatusResponse(op operationSnapshot) *bootstrappb.Workspace_GetInitializationStatus_Resp {
	return &bootstrappb.Workspace_GetInitializationStatus_Resp{
		OperationId:     op.OperationID,
		State:           op.State,
		Stage:           op.Stage,
		ProgressPercent: op.ProgressPercent,
		ReadyForLogin:   op.ReadyForLogin,
		RedirectUrl:     op.RedirectURL,
		ErrorCode:       op.ErrorCode,
		ErrorMessage:    op.ErrorMessage,
		NextPollAfterMs: op.NextPollAfterMs,
		StageDetail:     op.StageDetail,
	}
}
