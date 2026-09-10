// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { __normalizeAttachmentFieldsInPayloadForTest } from './formController';
import { execute } from '@/web/web/query/executor';
import { asyncFnRecorder, type AsyncFnRecorder } from '@/web/web/__tests__/mountApp';

type AttachmentContentServiceStub = {
  PrepareUpload: AsyncFnRecorder;
  FinalizeUpload: AsyncFnRecorder;
};

class TestFile extends Blob {
  name: string;

  constructor(parts: BlobPart[], fileName: string, options?: BlobPropertyBag) {
    super(parts, options);
    this.name = fileName;
  }
}

describe('attachment end-to-end chain regression', () => {
  const originalFetch = globalThis.fetch;
  const originalFile = (globalThis as any).File;
  let fetchMock: AsyncFnRecorder;

  beforeEach(() => {
    fetchMock = asyncFnRecorder(async () => ({ ok: true, status: 200 }));
    globalThis.fetch = fetchMock as any;
    (globalThis as any).File = TestFile;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    if (originalFile === undefined) {
      delete (globalThis as any).File;
    } else {
      (globalThis as any).File = originalFile;
    }
  });

  test('covers Create(Blob/File) -> Read(descriptor enrich) -> Update(clear/noop/set)', async () => {
    const finalizedQueue = ['ao-create-blob', 'ao-create-file', 'ao-update-file'];
    const attachmentContentService: AttachmentContentServiceStub = {
      PrepareUpload: asyncFnRecorder(async () => ({
        uploadId: `upload-${Math.random().toString(16).slice(2)}`,
        uploadTarget: {
          method: 'PUT',
          url: 'https://example.com/upload',
          headers: {
            'content-type': 'application/octet-stream',
          },
        },
      })),
      FinalizeUpload: asyncFnRecorder(async () => ({
        attachmentObjectId: finalizedQueue.shift(),
      })),
    };

    const createStoreByModel = (modelName: string) => {
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
                displayName: 'Avatar Binding Display',
              },
            ],
          }),
        } as any;
      }
      if (modelName === 'document.AttachmentContent') {
        return attachmentContentService;
      }
      throw new Error(`unexpected model: ${modelName}`);
    };

    const store = {
      fullModelName: 'auth.User',
      fieldsMetadata: {
        Avatar: { type: 'image' },
      },
      getContext: () => ({}),
    } as any;
    const normalizeDeps = { createStoreByModel };

    const createBlobPayload = await __normalizeAttachmentFieldsInPayloadForTest(
      store,
      {
        Avatar: new Blob([new Uint8Array([1, 2, 3])], { type: 'image/png' }),
      },
      {
        operation: 'create',
        ownerModel: 'auth.User',
        fields: ['Avatar'],
      },
      normalizeDeps
    );
    expect(createBlobPayload).toEqual({ Avatar: 'ao-create-blob' });

    const createFilePayload = await __normalizeAttachmentFieldsInPayloadForTest(
      store,
      {
        Avatar: new TestFile([new Uint8Array([4, 5, 6])], 'avatar-create.png', { type: 'image/png' }),
      },
      {
        operation: 'create',
        ownerModel: 'auth.User',
        fields: ['Avatar'],
      },
      normalizeDeps
    );
    expect(createFilePayload).toEqual({
      Avatar: {
        attachmentObjectId: 'ao-create-file',
        displayFileName: 'avatar-create.png',
      },
    });

    const readStore = {
      storeId: 'auth.User',
      fullModelName: 'auth.User',
      fieldsMetadata: {
        Avatar: { type: 'image' },
      },
      Search: async () => [
        {
          Id: 'USR-1',
          Avatar: 'bind-avatar',
        },
      ],
      Count: async () => 1,
    } as any;

    const readSnapshot = await execute(
      {
        main: {
          kind: 'search',
          params: {
            condition: ['Id', '=', 'USR-1'],
          },
          hash: 'attachment-read-main',
        },
        auxiliary: [
          {
            kind: 'count',
            params: {
              condition: ['Id', '=', 'USR-1'],
            },
            hash: 'attachment-read-count',
          },
        ],
      } as any,
      readStore,
      undefined,
      { createStoreByModel }
    );

    const readRow = (readSnapshot.rows[0] as any)?.payload;
    expect(readRow?.Avatar).toMatchObject({
      kind: 'attachment',
      fieldType: 'image',
      attachmentBindingId: 'bind-avatar',
      fileName: 'avatar.png',
      displayName: 'Avatar Binding Display',
      previewUrl: '/_document/bindings/bind-avatar/content',
      ownerModel: 'auth.User',
      ownerRecordId: 'USR-1',
    });

    const updateClearPayload = await __normalizeAttachmentFieldsInPayloadForTest(
      store,
      {
        Avatar: { kind: 'clear' },
      },
      {
        operation: 'update',
        ownerModel: 'auth.User',
        ownerRecordId: 'USR-1',
        fields: ['Avatar'],
      },
      normalizeDeps
    );
    expect(updateClearPayload).toEqual({ Avatar: null });

    const updateNoopPayload = await __normalizeAttachmentFieldsInPayloadForTest(
      store,
      {
        Avatar: { kind: 'noop' },
        Username: 'alice',
      },
      {
        operation: 'update',
        ownerModel: 'auth.User',
        ownerRecordId: 'USR-1',
        fields: ['Avatar'],
      },
      normalizeDeps
    );
    expect(updateNoopPayload).toEqual({ Username: 'alice' });

    const updateSetPayload = await __normalizeAttachmentFieldsInPayloadForTest(
      store,
      {
        Avatar: { kind: 'set', attachmentObjectId: 'ao-direct-set' },
      },
      {
        operation: 'update',
        ownerModel: 'auth.User',
        ownerRecordId: 'USR-1',
        fields: ['Avatar'],
      },
      normalizeDeps
    );
    expect(updateSetPayload).toEqual({ Avatar: 'ao-direct-set' });

    const updateFilePayload = await __normalizeAttachmentFieldsInPayloadForTest(
      store,
      {
        Avatar: new TestFile([new Uint8Array([7, 8, 9])], 'avatar-update.png', { type: 'image/png' }),
      },
      {
        operation: 'update',
        ownerModel: 'auth.User',
        ownerRecordId: 'USR-1',
        fields: ['Avatar'],
      },
      normalizeDeps
    );
    expect(updateFilePayload).toEqual({
      Avatar: {
        attachmentObjectId: 'ao-update-file',
        displayFileName: 'avatar-update.png',
      },
    });

    expect(attachmentContentService.PrepareUpload.calls.length).toBe(3);
    expect(attachmentContentService.FinalizeUpload.calls.length).toBe(3);
    expect(fetchMock.calls.length).toBe(3);
  });
});
