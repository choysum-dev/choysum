// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, test, vi } from 'vitest';

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: vi.fn(),
}));

import { createStoreByModel } from '@/web/web/stores/registry';
import { setTokenProvider } from '@/core/web/rpc';
import { execute } from './executor';

describe('query executor browse attachment descriptor regression', () => {
  beforeEach(() => {
    (createStoreByModel as any).mockReset();
    setTokenProvider({
      getToken: async () => null,
      refreshToken: async () => false,
      shouldRefreshToken: async () => false,
    });
  });

  test('browse path decodes and enriches image descriptor from binding id', async () => {
    (createStoreByModel as any).mockImplementation((modelName: string) => {
      if (modelName === 'base.Company') {
        return {
          Search: async () => [
            {
              Id: 'C1',
              DisplayName: 'Main Company',
            },
          ],
        } as any;
      }

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
                displayName: 'avatar.png',
              },
            ],
          }),
        } as any;
      }

      throw new Error(`unexpected model: ${modelName}`);
    });

    const store = {
      storeId: 'auth.User',
      fullModelName: 'auth.User',
      fieldsMetadata: {
        Avatar: { type: 'image' },
        CompanyId: { type: 'ManyToOneRef', relationModel: 'base.Company' },
      },
      Browse: async () => ({
        Id: 'USR-1',
        Avatar: 'bind-avatar',
        CompanyId: 'C1',
      }),
    } as any;

    const snapshot = await execute(
      {
        main: {
          kind: 'browse',
          params: { id: 'USR-1' },
          hash: 'browse-hash',
        },
        auxiliary: [],
      } as any,
      store
    );

    const row = snapshot.rows[0] as any;
    expect(row?.payload?.CompanyId).toEqual({
      Id: 'C1',
      DisplayName: 'Main Company',
    });
    expect(row?.payload?.Avatar).toMatchObject({
      kind: 'attachment',
      fieldType: 'image',
      fieldName: 'Avatar',
      attachmentBindingId: 'bind-avatar',
      ownerModel: 'auth.User',
      ownerRecordId: 'USR-1',
      fileName: 'avatar.png',
      displayName: 'avatar.png',
      previewUrl: '/_document/bindings/bind-avatar/content',
    });
  });

  test('browse path decodes and enriches binary descriptor without preview for non-image mime', async () => {
    (createStoreByModel as any).mockImplementation((modelName: string) => {
      if (modelName === 'document.AttachmentBinding') {
        return {
          BatchDescribe: async () => ({
            items: [
              {
                attachmentBindingId: 'bind-doc',
                descriptor: {
                  fileName: 'manual.pdf',
                  mimeType: 'application/pdf',
                  downloadUrl: '/_document/bindings/bind-doc/content',
                },
                displayName: 'manual.pdf',
              },
            ],
          }),
        } as any;
      }

      throw new Error(`unexpected model: ${modelName}`);
    });

    const store = {
      storeId: 'auth.User',
      fullModelName: 'auth.User',
      fieldsMetadata: {
        IdentityDoc: { type: 'binary' },
      },
      Browse: async () => ({
        Id: 'USR-9',
        IdentityDoc: 'bind-doc',
      }),
    } as any;

    const snapshot = await execute(
      {
        main: {
          kind: 'browse',
          params: { id: 'USR-9' },
          hash: 'browse-binary-hash',
        },
        auxiliary: [],
      } as any,
      store
    );

    const row = snapshot.rows[0] as any;
    expect(row?.payload?.IdentityDoc).toMatchObject({
      kind: 'attachment',
      fieldType: 'binary',
      fieldName: 'IdentityDoc',
      attachmentBindingId: 'bind-doc',
      ownerModel: 'auth.User',
      ownerRecordId: 'USR-9',
      fileName: 'manual.pdf',
      displayName: 'manual.pdf',
    });
    expect(row?.payload?.IdentityDoc?.previewUrl).toBeUndefined();
  });

  test('browse path normalizes relation projection payloads', async () => {
    const store = {
      storeId: 'auth.User',
      fullModelName: 'auth.User',
      fieldsMetadata: {
        Username: { type: 'varchar' },
      },
      Browse: async () => ({
        Id: 'USR-2',
        Username: 'alice',
        $rel$_manager_id: JSON.stringify({
          id: 'M-1',
          display_name: 'Manager A',
        }),
      }),
    } as any;

    const snapshot = await execute(
      {
        main: {
          kind: 'browse',
          params: { id: 'USR-2' },
          hash: 'browse-rel-hash',
        },
        auxiliary: [],
      } as any,
      store
    );

    const row = snapshot.rows[0] as any;
    expect(row?.payload?.ManagerId).toEqual({
      Id: 'M-1',
      DisplayName: 'Manager A',
    });
  });
});
