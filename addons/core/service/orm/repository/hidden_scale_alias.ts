// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Hidden alias prefix for dynamic decimal scale values to avoid CamelCase or Go normalization.
export const DEC_SCALE_ALIAS_PREFIX = '$dec$';

// Build hidden aliases in the form $dec$<FieldName>__scale.
export const buildHiddenScaleAlias = (fieldName: string) => `${DEC_SCALE_ALIAS_PREFIX}${fieldName}__scale`;
