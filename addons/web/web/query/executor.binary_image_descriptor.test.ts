// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, test } from 'vitest';
import { __decodeBinaryImageFieldsForTest, __enrichBinaryImageDescriptorsForTest } from './executor';
import { setTokenProvider } from '@/core/web/rpc';

function resetTokenProvider() {
  setTokenProvider({
    getToken: async () => null,
    refreshToken: async () => false,
    shouldRefreshToken: async () => false,
  });
}

describe('query executor binary/image descriptor decode', () => {
  beforeEach(() => {
    resetTokenProvider();
  });

  afterEach(() => {
    resetTokenProvider();
  });

  test('decodes binding-id strings into attachment descriptors', () => {
    const store = {
      fullModelName: 'demo.Asset',
      fieldsMetadata: {
        Avatar: { type: 'binary' },
        Cover: { type: 'image' },
        Name: { type: 'varchar' },
      },
    } as any;

    const rows = [
      {
        Id: 'RID-100',
        Avatar: 'bind-avatar',
        Cover: 'bind-cover',
        Name: 'Asset A',
      },
    ];

    __decodeBinaryImageFieldsForTest(store, rows);

    expect(rows[0]?.Avatar).toEqual({
      kind: 'attachment',
      fieldType: 'binary',
      fieldName: 'Avatar',
      attachmentBindingId: 'bind-avatar',
      ownerModel: 'demo.Asset',
      ownerRecordId: 'RID-100',
    });
    expect(rows[0]?.Cover).toEqual({
      kind: 'attachment',
      fieldType: 'image',
      fieldName: 'Cover',
      attachmentBindingId: 'bind-cover',
      ownerModel: 'demo.Asset',
      ownerRecordId: 'RID-100',
    });
    expect(rows[0]?.Name).toBe('Asset A');
  });

  test('normalizes object payload into descriptor while preserving metadata fields', () => {
    const store = {
      fullModelName: 'demo.Asset',
      fieldsMetadata: {
        Avatar: { type: 'binary' },
      },
    } as any;

    const rows = [
      {
        id: 'rid-obj',
        Avatar: {
          attachmentBindingId: 'bind-obj',
          fileName: 'avatar.png',
          displayName: 'Avatar PNG',
          previewUrl: 'https://example.com/p.png',
        },
      },
    ];

    __decodeBinaryImageFieldsForTest(store, rows);

    expect(rows[0]?.Avatar).toEqual({
      kind: 'attachment',
      fieldType: 'binary',
      fieldName: 'Avatar',
      attachmentBindingId: 'bind-obj',
      ownerModel: 'demo.Asset',
      ownerRecordId: 'rid-obj',
      fileName: 'avatar.png',
      displayName: 'Avatar PNG',
      previewUrl: 'https://example.com/p.png',
    });
  });

  test('enriches descriptors by calling document.AttachmentBinding.BatchDescribe', async () => {
    const store = {
      fullModelName: 'demo.Asset',
      fieldsMetadata: {
        Avatar: { type: 'binary' },
        Cover: { type: 'image' },
      },
    } as any;

    const rows = [
      {
        Id: 'RID-200',
        Avatar: 'bind-avatar',
        Cover: 'bind-cover',
      },
    ];

    __decodeBinaryImageFieldsForTest(store, rows);

    await __enrichBinaryImageDescriptorsForTest(store, rows, {
      createStoreByModel: (modelName: string) => {
        if (modelName === 'document.AttachmentBinding') {
          return {
            BatchDescribe: async () => ({
              items: [
                {
                  attachmentBindingId: 'bind-avatar',
                  descriptor: {
                    fileName: 'avatar.png',
                    mimeType: 'image/png',
                    downloadUrl: '/_document/bindings/bind-avatar/content',
                  },
                  displayName: 'Avatar Binding',
                },
                {
                  attachmentBindingId: 'bind-cover',
                  descriptor: {
                    fileName: 'cover.jpg',
                    mimeType: 'image/jpeg',
                    downloadUrl: '/_document/bindings/bind-cover/content',
                  },
                  displayName: 'Cover Binding',
                },
              ],
            }),
          } as any;
        }

        throw new Error(`unexpected model: ${modelName}`);
      },
    } as any);

    expect(rows[0]?.Avatar).toMatchObject({
      attachmentBindingId: 'bind-avatar',
      fileName: 'avatar.png',
      displayName: 'Avatar Binding',
      previewUrl: '/_document/bindings/bind-avatar/content',
    });
    expect(rows[0]?.Cover).toMatchObject({
      attachmentBindingId: 'bind-cover',
      displayName: 'Cover Binding',
      previewUrl: '/_document/bindings/bind-cover/content',
    });
  });

  test('appends query token for internal document binding preview url', async () => {
    setTokenProvider({
      getToken: async () => 'preview-token',
      refreshToken: async () => true,
      shouldRefreshToken: async () => true,
    });

    const store = {
      fullModelName: 'demo.Asset',
      fieldsMetadata: {
        Cover: { type: 'image' },
      },
    } as any;

    const rows = [
      {
        Id: 'RID-300',
        Cover: 'bind-cover',
      },
    ];

    __decodeBinaryImageFieldsForTest(store, rows);

    await __enrichBinaryImageDescriptorsForTest(store, rows, {
      createStoreByModel: (modelName: string) => {
        if (modelName === 'document.AttachmentBinding') {
          return {
            BatchDescribe: async () => ({
              items: [
                {
                  attachmentBindingId: 'bind-cover',
                  descriptor: {
                    fileName: 'cover.jpg',
                    mimeType: 'image/jpeg',
                    downloadUrl: '/_document/bindings/bind-cover/content',
                  },
                  displayName: 'Cover Binding',
                },
              ],
            }),
          } as any;
        }

        throw new Error(`unexpected model: ${modelName}`);
      },
    } as any);

    expect(rows[0]?.Cover).toMatchObject({
      attachmentBindingId: 'bind-cover',
      previewUrl: '/_document/bindings/bind-cover/content?token=preview-token',
    });
  });
});
