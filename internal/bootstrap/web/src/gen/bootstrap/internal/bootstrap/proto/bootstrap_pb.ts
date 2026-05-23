// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/* eslint-disable */

import type { GenEnum, GenFile, GenMessage, GenService } from "@bufbuild/protobuf/codegenv2";
import { enumDesc, fileDesc, messageDesc, serviceDesc } from "@bufbuild/protobuf/codegenv2";
import type { Message } from "@bufbuild/protobuf";

/**
 * Describes the file internal/bootstrap/proto/bootstrap.proto.
 */
export const file_internal_bootstrap_proto_bootstrap: GenFile = /*@__PURE__*/
  fileDesc("CihpbnRlcm5hbC9ib290c3RyYXAvcHJvdG8vYm9vdHN0cmFwLnByb3RvEglib290c3RyYXAifQoYV29ya3NwYWNlX0luaXRpYWxpemVfUmVxEhYKDmFkbWluX3VzZXJuYW1lGAEgASgJEhAKCHBhc3N3b3JkGAIgASgJEh4KFmNsaWVudF9oYXNoaW5nX2VuYWJsZWQYAyABKAgSFwoPaWRlbXBvdGVuY3lfa2V5GAQgASgJIo4BChlXb3Jrc3BhY2VfSW5pdGlhbGl6ZV9SZXNwEhAKCGFjY2VwdGVkGAEgASgIEhQKDG9wZXJhdGlvbl9pZBgCIAEoCRIaChJuZXh0X3BvbGxfYWZ0ZXJfbXMYAyABKAMSLQoFc3RhdGUYBCABKA4yHi5ib290c3RyYXAuSW5pdGlhbGl6YXRpb25TdGF0ZSI9CiVXb3Jrc3BhY2VfR2V0SW5pdGlhbGl6YXRpb25TdGF0dXNfUmVxEhQKDG9wZXJhdGlvbl9pZBgBIAEoCSKsAgomV29ya3NwYWNlX0dldEluaXRpYWxpemF0aW9uU3RhdHVzX1Jlc3ASFAoMb3BlcmF0aW9uX2lkGAEgASgJEi0KBXN0YXRlGAIgASgOMh4uYm9vdHN0cmFwLkluaXRpYWxpemF0aW9uU3RhdGUSLQoFc3RhZ2UYAyABKA4yHi5ib290c3RyYXAuSW5pdGlhbGl6YXRpb25TdGFnZRIYChBwcm9ncmVzc19wZXJjZW50GAQgASgFEhcKD3JlYWR5X2Zvcl9sb2dpbhgFIAEoCBIUCgxyZWRpcmVjdF91cmwYBiABKAkSEgoKZXJyb3JfY29kZRgHIAEoCRIVCg1lcnJvcl9tZXNzYWdlGAggASgJEhoKEm5leHRfcG9sbF9hZnRlcl9tcxgJIAEoAyrEAQoTSW5pdGlhbGl6YXRpb25TdGF0ZRIkCiBJTklUSUFMSVpBVElPTl9TVEFURV9VTlNQRUNJRklFRBAAEiAKHElOSVRJQUxJWkFUSU9OX1NUQVRFX1BFTkRJTkcQARIgChxJTklUSUFMSVpBVElPTl9TVEFURV9SVU5OSU5HEAISIgoeSU5JVElBTElaQVRJT05fU1RBVEVfU1VDQ0VFREVEEAMSHwobSU5JVElBTElaQVRJT05fU1RBVEVfRkFJTEVEEAQq5AIKE0luaXRpYWxpemF0aW9uU3RhZ2USJAogSU5JVElBTElaQVRJT05fU1RBR0VfVU5TUEVDSUZJRUQQABIlCiFJTklUSUFMSVpBVElPTl9TVEFHRV9BQ1FVSVJFX0xPQ0sQARIyCi5JTklUSUFMSVpBVElPTl9TVEFHRV9DSEVDS19XT1JLU1BBQ0VfRlJFU0hORVNTEAISLworSU5JVElBTElaQVRJT05fU1RBR0VfRU5TVVJFX01JTklNQUxfUlVOVElNRRADEi8KK0lOSVRJQUxJWkFUSU9OX1NUQUdFX1ZBTElEQVRFX1JVTlRJTUVfUkVBRFkQBBIlCiFJTklUSUFMSVpBVElPTl9TVEFHRV9VUERBVEVfQURNSU4QBRIkCiBJTklUSUFMSVpBVElPTl9TVEFHRV9TV0lUQ0hfTU9ERRAGEh0KGUlOSVRJQUxJWkFUSU9OX1NUQUdFX0RPTkUQBzLkAQoJV29ya3NwYWNlElcKCkluaXRpYWxpemUSIy5ib290c3RyYXAuV29ya3NwYWNlX0luaXRpYWxpemVfUmVxGiQuYm9vdHN0cmFwLldvcmtzcGFjZV9Jbml0aWFsaXplX1Jlc3ASfgoXR2V0SW5pdGlhbGl6YXRpb25TdGF0dXMSMC5ib290c3RyYXAuV29ya3NwYWNlX0dldEluaXRpYWxpemF0aW9uU3RhdHVzX1JlcRoxLmJvb3RzdHJhcC5Xb3Jrc3BhY2VfR2V0SW5pdGlhbGl6YXRpb25TdGF0dXNfUmVzcEJRWk9naXRodWIuY29tL3Byb2plY3Qtb3J5b24vb3J5b24vaW50ZXJuYWwvYm9vdHN0cmFwL3Byb3RvL2Jvb3RzdHJhcHBiO2Jvb3RzdHJhcHBiYgZwcm90bzM");

/**
 * @generated from message bootstrap.Workspace_Initialize_Req
 */
export type Workspace_Initialize_Req = Message<"bootstrap.Workspace_Initialize_Req"> & {
  /**
   * @generated from field: string admin_username = 1;
   */
  adminUsername: string;

  /**
   * @generated from field: string password = 2;
   */
  password: string;

  /**
   * @generated from field: bool client_hashing_enabled = 3;
   */
  clientHashingEnabled: boolean;

  /**
   * @generated from field: string idempotency_key = 4;
   */
  idempotencyKey: string;
};

/**
 * Describes the message bootstrap.Workspace_Initialize_Req.
 * Use `create(Workspace_Initialize_ReqSchema)` to create a new message.
 */
export const Workspace_Initialize_ReqSchema: GenMessage<Workspace_Initialize_Req> = /*@__PURE__*/
  messageDesc(file_internal_bootstrap_proto_bootstrap, 0);

/**
 * @generated from message bootstrap.Workspace_Initialize_Resp
 */
export type Workspace_Initialize_Resp = Message<"bootstrap.Workspace_Initialize_Resp"> & {
  /**
   * @generated from field: bool accepted = 1;
   */
  accepted: boolean;

  /**
   * @generated from field: string operation_id = 2;
   */
  operationId: string;

  /**
   * @generated from field: int64 next_poll_after_ms = 3;
   */
  nextPollAfterMs: bigint;

  /**
   * @generated from field: bootstrap.InitializationState state = 4;
   */
  state: InitializationState;
};

/**
 * Describes the message bootstrap.Workspace_Initialize_Resp.
 * Use `create(Workspace_Initialize_RespSchema)` to create a new message.
 */
export const Workspace_Initialize_RespSchema: GenMessage<Workspace_Initialize_Resp> = /*@__PURE__*/
  messageDesc(file_internal_bootstrap_proto_bootstrap, 1);

/**
 * @generated from message bootstrap.Workspace_GetInitializationStatus_Req
 */
export type Workspace_GetInitializationStatus_Req = Message<"bootstrap.Workspace_GetInitializationStatus_Req"> & {
  /**
   * @generated from field: string operation_id = 1;
   */
  operationId: string;
};

/**
 * Describes the message bootstrap.Workspace_GetInitializationStatus_Req.
 * Use `create(Workspace_GetInitializationStatus_ReqSchema)` to create a new message.
 */
export const Workspace_GetInitializationStatus_ReqSchema: GenMessage<Workspace_GetInitializationStatus_Req> = /*@__PURE__*/
  messageDesc(file_internal_bootstrap_proto_bootstrap, 2);

/**
 * @generated from message bootstrap.Workspace_GetInitializationStatus_Resp
 */
export type Workspace_GetInitializationStatus_Resp = Message<"bootstrap.Workspace_GetInitializationStatus_Resp"> & {
  /**
   * @generated from field: string operation_id = 1;
   */
  operationId: string;

  /**
   * @generated from field: bootstrap.InitializationState state = 2;
   */
  state: InitializationState;

  /**
   * @generated from field: bootstrap.InitializationStage stage = 3;
   */
  stage: InitializationStage;

  /**
   * @generated from field: int32 progress_percent = 4;
   */
  progressPercent: number;

  /**
   * @generated from field: bool ready_for_login = 5;
   */
  readyForLogin: boolean;

  /**
   * @generated from field: string redirect_url = 6;
   */
  redirectUrl: string;

  /**
   * @generated from field: string error_code = 7;
   */
  errorCode: string;

  /**
   * @generated from field: string error_message = 8;
   */
  errorMessage: string;

  /**
   * @generated from field: int64 next_poll_after_ms = 9;
   */
  nextPollAfterMs: bigint;
};

/**
 * Describes the message bootstrap.Workspace_GetInitializationStatus_Resp.
 * Use `create(Workspace_GetInitializationStatus_RespSchema)` to create a new message.
 */
export const Workspace_GetInitializationStatus_RespSchema: GenMessage<Workspace_GetInitializationStatus_Resp> = /*@__PURE__*/
  messageDesc(file_internal_bootstrap_proto_bootstrap, 3);

/**
 * @generated from enum bootstrap.InitializationState
 */
export enum InitializationState {
  /**
   * @generated from enum value: INITIALIZATION_STATE_UNSPECIFIED = 0;
   */
  UNSPECIFIED = 0,

  /**
   * @generated from enum value: INITIALIZATION_STATE_PENDING = 1;
   */
  PENDING = 1,

  /**
   * @generated from enum value: INITIALIZATION_STATE_RUNNING = 2;
   */
  RUNNING = 2,

  /**
   * @generated from enum value: INITIALIZATION_STATE_SUCCEEDED = 3;
   */
  SUCCEEDED = 3,

  /**
   * @generated from enum value: INITIALIZATION_STATE_FAILED = 4;
   */
  FAILED = 4,
}

/**
 * Describes the enum bootstrap.InitializationState.
 */
export const InitializationStateSchema: GenEnum<InitializationState> = /*@__PURE__*/
  enumDesc(file_internal_bootstrap_proto_bootstrap, 0);

/**
 * @generated from enum bootstrap.InitializationStage
 */
export enum InitializationStage {
  /**
   * @generated from enum value: INITIALIZATION_STAGE_UNSPECIFIED = 0;
   */
  UNSPECIFIED = 0,

  /**
   * @generated from enum value: INITIALIZATION_STAGE_ACQUIRE_LOCK = 1;
   */
  ACQUIRE_LOCK = 1,

  /**
   * @generated from enum value: INITIALIZATION_STAGE_CHECK_WORKSPACE_FRESHNESS = 2;
   */
  CHECK_WORKSPACE_FRESHNESS = 2,

  /**
   * @generated from enum value: INITIALIZATION_STAGE_ENSURE_MINIMAL_RUNTIME = 3;
   */
  ENSURE_MINIMAL_RUNTIME = 3,

  /**
   * @generated from enum value: INITIALIZATION_STAGE_VALIDATE_RUNTIME_READY = 4;
   */
  VALIDATE_RUNTIME_READY = 4,

  /**
   * @generated from enum value: INITIALIZATION_STAGE_UPDATE_ADMIN = 5;
   */
  UPDATE_ADMIN = 5,

  /**
   * @generated from enum value: INITIALIZATION_STAGE_SWITCH_MODE = 6;
   */
  SWITCH_MODE = 6,

  /**
   * @generated from enum value: INITIALIZATION_STAGE_DONE = 7;
   */
  DONE = 7,
}

/**
 * Describes the enum bootstrap.InitializationStage.
 */
export const InitializationStageSchema: GenEnum<InitializationStage> = /*@__PURE__*/
  enumDesc(file_internal_bootstrap_proto_bootstrap, 1);

/**
 * @generated from service bootstrap.Workspace
 */
export const Workspace: GenService<{
  /**
   * @generated from rpc bootstrap.Workspace.Initialize
   */
  initialize: {
    methodKind: "unary";
    input: typeof Workspace_Initialize_ReqSchema;
    output: typeof Workspace_Initialize_RespSchema;
  },
  /**
   * @generated from rpc bootstrap.Workspace.GetInitializationStatus
   */
  getInitializationStatus: {
    methodKind: "unary";
    input: typeof Workspace_GetInitializationStatus_ReqSchema;
    output: typeof Workspace_GetInitializationStatus_RespSchema;
  },
}> = /*@__PURE__*/
  serviceDesc(file_internal_bootstrap_proto_bootstrap, 0);

