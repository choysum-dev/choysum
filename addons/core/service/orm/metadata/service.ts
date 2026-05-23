// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Value encodings supported by generated service metadata.
 */
export type ValueType = 'Empty' | 'Struct' | 'Value' | 'ListValue' | 'NullValue' | 'double' | 'string' | 'bool';

/**
 * Metadata for one service parameter.
 */
export interface ParamMetadata {
  name?: string;
  type?: ValueType;
  description?: string;
  required?: boolean;
}

/**
 * Metadata recorded for one generated service method.
 */
export interface ServiceMetadata {
  name: string;
  kind: 'Unary' | 'ServerStream';
  type: ValueType;
  description?: string;
  params?: ParamMetadata[];
}
