// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage } from '@/core/service/orm/metadata/storage';
import { resolveModelConstructor } from '@/core/service/orm/model/model_registry';
import type { FieldMetadata } from '@/core/service/orm/metadata/field';
import { createTranslate } from '@/core/service/i18n';
import { DocumentErrCode, GrpcCode, throwDocumentError } from '../error';
import type AttachmentContent from './attachment_object';
import { DEFAULT_GLOBAL_MAX_UPLOAD_BYTES } from './_upload';

const { _t } = createTranslate('document');

function resolveOwnerFieldMetadata(ownerModel: string, fieldName: string): FieldMetadata | undefined {
  const ModelCtor = resolveModelConstructor(ownerModel);
  if (!ModelCtor) {
    return undefined;
  }
  const modelMeta = MetadataStorage.instance.getModelMetadata(ModelCtor);
  if (!modelMeta) {
    return undefined;
  }
  return modelMeta.fields.get(fieldName);
}

function resolveEffectiveMaxUploadBytes(maxUploadBytes: number): number {
  return Math.min(maxUploadBytes, DEFAULT_GLOBAL_MAX_UPLOAD_BYTES);
}

/**
 * Validates finalized attachment content against per-field upload and dimension limits.
 * Dimension probe is skipped when ImageWidth/ImageHeight are absent.
 */
export function validateAttachmentContentFieldLimits(
  ownerModel: string,
  fieldName: string,
  attachmentContent: AttachmentContent
): void {
  const fieldMeta = resolveOwnerFieldMetadata(ownerModel, fieldName);
  if (!fieldMeta) {
    return;
  }

  const hasByteLimit = typeof fieldMeta.maxUploadBytes === 'number' && fieldMeta.maxUploadBytes > 0;
  const hasWidthLimit = fieldMeta.type === 'image' && typeof fieldMeta.maxWidth === 'number' && fieldMeta.maxWidth > 0;
  const hasHeightLimit = fieldMeta.type === 'image' && typeof fieldMeta.maxHeight === 'number' && fieldMeta.maxHeight > 0;
  if (!hasByteLimit && !hasWidthLimit && !hasHeightLimit) {
    return;
  }

  if (hasByteLimit) {
    const effectiveMaxBytes = resolveEffectiveMaxUploadBytes(fieldMeta.maxUploadBytes as number);
    const sizeBytes = Number(attachmentContent.SizeBytes ?? 0);
    if (Number.isFinite(sizeBytes) && sizeBytes > effectiveMaxBytes) {
      throwDocumentError(
        DocumentErrCode.INVALID_ARGUMENT,
        _t('upload exceeds field maxUploadBytes', { scope: 'service/models/_binding_field_limits' }),
        GrpcCode.InvalidArgument,
        {
          ownerModel,
          fieldName,
          sizeBytes: String(sizeBytes),
          maxUploadBytes: String(effectiveMaxBytes),
        }
      );
    }
  }

  if (fieldMeta.type === 'image' && (hasWidthLimit || hasHeightLimit)) {
    const imageWidth = attachmentContent.ImageWidth;
    const imageHeight = attachmentContent.ImageHeight;
    const width = typeof imageWidth === 'number' && Number.isFinite(imageWidth) ? Math.trunc(imageWidth) : undefined;
    const height = typeof imageHeight === 'number' && Number.isFinite(imageHeight) ? Math.trunc(imageHeight) : undefined;
    if (width === undefined && height === undefined) {
      return;
    }
    if (hasWidthLimit && width !== undefined && width > (fieldMeta.maxWidth as number)) {
      throwDocumentError(
        DocumentErrCode.INVALID_ARGUMENT,
        _t('upload exceeds field maxWidth', { scope: 'service/models/_binding_field_limits' }),
        GrpcCode.InvalidArgument,
        {
          ownerModel,
          fieldName,
          imageWidth: String(width),
          maxWidth: String(fieldMeta.maxWidth),
        }
      );
    }
    if (hasHeightLimit && height !== undefined && height > (fieldMeta.maxHeight as number)) {
      throwDocumentError(
        DocumentErrCode.INVALID_ARGUMENT,
        _t('upload exceeds field maxHeight', { scope: 'service/models/_binding_field_limits' }),
        GrpcCode.InvalidArgument,
        {
          ownerModel,
          fieldName,
          imageHeight: String(height),
          maxHeight: String(fieldMeta.maxHeight),
        }
      );
    }
  }
}
