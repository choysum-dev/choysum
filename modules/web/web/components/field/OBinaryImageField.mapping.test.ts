// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

function source(name: string): string {
  return readFileSync(resolve(__dirname, name), 'utf8');
}

describe('OBinaryField/OImageField mapping contract', () => {
  test('OBinaryField renders descriptor text from attachmentBindingId', () => {
    const s = source('OBinaryField.vue');

    expect(s).toContain("defineOptions({ name: 'OBinaryField' });");
    expect(s).toContain('raw.attachmentBindingId ?? raw.bindingId ?? raw.Id ?? raw.id');
    expect(s).toContain(
      "type UploadPassthroughProps = Partial<Pick<UploadProps, 'drag' | 'multiple' | 'limit' | 'disabled' | 'showFileList' | 'listType' | 'onExceed'>>;"
    );
    expect(s).toContain(':drag="shouldUseDragMode(fieldValue().value)"');
    expect(s).toContain('UploadFilled');
    expect(s).toContain('@click="removeBinary(fieldValue, onFieldChange)"');
    expect(s).toContain('function shouldShowUploadTrigger(raw: unknown): boolean');
    expect(s).toContain('return !hasAttachment(raw);');
    expect(s).not.toContain('if (fileName && bindingId) return `${fileName} (${bindingId})`;');
    expect(s).toContain("return '[binary]';");
  });

  test('OBinaryField keeps upload envelope and remove protocol for binary attachments', () => {
    const s = source('OBinaryField.vue');

    expect(s).toContain("kind: 'set'");
    expect(s).toContain('const fileName = normalizeOptionalString(file.name)');
    expect(s).toContain('const contentType = normalizeOptionalString(file.type)');
    expect(s).toContain('valueRef.value = null;');
    expect(s).toContain('async function removeBinary(');
    expect(s).toContain('await onFieldChange();');
  });

  test('OImageField supports descriptor previewUrl and fallback text', () => {
    const s = source('OImageField.vue');

    expect(s).toContain("defineOptions({ name: 'OImageField' });");
    expect(s).toContain(
      "type UploadPassthroughProps = Partial<Pick<UploadProps, 'drag' | 'multiple' | 'limit' | 'disabled' | 'showFileList' | 'listType' | 'onExceed'>>;"
    );
    expect(s).toContain('resolvePreviewUrl(raw: unknown): string | undefined');
    expect(s).toContain('const fromPreviewUrl = normalizeOptionalString(raw.previewUrl)');
    expect(s).toContain('const fromThumbnailUrl = normalizeOptionalString(raw.thumbnailUrl)');
    expect(s).toContain(':drag="shouldUseDragMode(fieldValue().value)"');
    expect(s).toContain('UploadFilled');
    expect(s).toContain('@click="removeImage(fieldValue, onFieldChange)"');
    expect(s).toContain('function shouldShowUploadTrigger(raw: unknown): boolean');
    expect(s).toContain('return !hasAttachment(raw);');
    expect(s).not.toContain('if (fileName && bindingId) return `${fileName} (${bindingId})`;');
    expect(s).toContain("return '[image]';");
  });
});
