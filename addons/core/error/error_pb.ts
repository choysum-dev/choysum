// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/* eslint-disable */

import type { GenFile, GenMessage } from '@bufbuild/protobuf/codegenv2';
import { fileDesc, messageDesc } from '@bufbuild/protobuf/codegenv2';
import type { Message } from '@bufbuild/protobuf';

/**
 * Describes the file error.proto.
 */
export const file_error: GenFile =
  /*@__PURE__*/
  fileDesc(
    'CgtlcnJvci5wcm90bxIHb2Vycm9ycyLEAQoJRXJyb3JJbmZvEhAKCGVycm9yX2lkGAEgASgJEg4KBmRvbWFpbhgCIAEoCRIMCgRjb2RlGAMgASgJEg8KB21lc3NhZ2UYBCABKAkSEQoJZ3JwY19jb2RlGAUgASgFEjIKCG1ldGFkYXRhGAYgAygLMiAub2Vycm9ycy5FcnJvckluZm8uTWV0YWRhdGFFbnRyeRovCg1NZXRhZGF0YUVudHJ5EgsKA2tleRgBIAEoCRINCgV2YWx1ZRgCIAEoCToCOAFCNFoyZ2l0aHViLmNvbS9wcm9qZWN0LW9yeW9uL29yeW9uL3BrZy9vZXJyb3JzO29lcnJvcnNiBnByb3RvMw'
  );

/**
 * ErrorInfo is a structure that carries error details.
 *
 * @generated from message oerrors.ErrorInfo
 */
export type ErrorInfo = Message<'oerrors.ErrorInfo'> & {
  /**
   * Error ID used for tracing and log correlation.
   *
   * @generated from field: string error_id = 1;
   */
  errorId: string;

  /**
   * System or service domain where the error occurred.
   * For example, "auth.choysum" or "sale.choysum".
   *
   * @generated from field: string domain = 2;
   */
  domain: string;

  /**
   * Machine-readable error reason code as a stable constant.
   * For example, "USERNAME_TAKEN" or "EMAIL_ALREADY_EXISTS".
   *
   * @generated from field: string code = 3;
   */
  code: string;

  /**
   * Detailed error message that can be shown to users.
   * It may contain formatted content such as "username {username} is already taken".
   *
   * @generated from field: string message = 4;
   */
  message: string;

  /**
   * gRPC status code matching google.golang.org/grpc/codes directly.
   *
   * @generated from field: int32 grpc_code = 5;
   */
  grpcCode: number;

  /**
   * Key-value pairs with extra error-related details.
   * For example, {"username": "user123", "field": "username"}.
   *
   * @generated from field: map<string, string> metadata = 6;
   */
  metadata: { [key: string]: string };
};

/**
 * Describes the message oerrors.ErrorInfo.
 * Use `create(ErrorInfoSchema)` to create a new message.
 */
export const ErrorInfoSchema: GenMessage<ErrorInfo> = /*@__PURE__*/ messageDesc(file_error, 0);
