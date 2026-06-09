// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, test, vi } from 'vitest';

const executeMock = vi.fn();
const createStoreByModelMock = vi.fn();
const handoffSetMock = vi.fn();

vi.mock('@/web/web/query/context', () => ({
  buildBrowseContext: vi.fn(() => ({ kind: 'browse_ctx' })),
}));

vi.mock('@/web/web/query/planner', () => ({
  buildPlan: vi.fn(() => ({ kind: 'browse_plan' })),
}));

vi.mock('@/web/web/query/executor', () => ({
  execute: (...args: any[]) => executeMock(...args),
}));

vi.mock('@/web/web/stores/registry', () => ({
  createStoreByModel: (...args: any[]) => createStoreByModelMock(...args),
}));

vi.mock('@/web/web/query/utils/handoff', () => ({
  handoffCache: {
    set: (...args: any[]) => handoffSetMock(...args),
  },
  flashRead: vi.fn(() => undefined),
}));

vi.mock('@/web/web/query/utils/registry/fieldReady', () => ({
  awaitFieldSelection: vi.fn(async () => {}),
}));

import { createFormController } from './formController';

function newStore(
  updateResult: Record<string, unknown>,
  fieldsMetadata: Record<string, { type: string }> = {
    Avatar: { type: 'image' },
    Username: { type: 'varchar' },
  }
) {
  return {
    fullModelName: 'auth.User',
    storeId: 'auth.User',
    fieldsMetadata,
    state: {},
    getContext: () => ({}),
    UpdateById: vi.fn(async () => updateResult),
  } as any;
}

function newAttachmentService() {
  return {
    PrepareUpload: vi.fn(async () => ({ uploadId: 'up_1', uploadTarget: { method: 'PUT', url: '/_document/uploads/up_1' } })),
    FinalizeUpload: vi.fn(async () => ({ attachmentObjectId: 'ao_1' })),
  };
}

describe('formController submit attachment refresh', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    createStoreByModelMock.mockReturnValue(newAttachmentService());

    executeMock.mockResolvedValue({
      kind: 'search',
      rows: [
        {
          kind: 'record',
          key: 'u1',
          payload: {
            Id: 'u1',
            Avatar: {
              attachmentBindingId: 'bind-new',
              previewUrl: '/_document/bindings/bind-new/content?token=t',
              fileName: 'avatar.jpg',
            },
            Username: 'admin',
          },
          raw: {},
        },
      ],
      total: 1,
      ts: Date.now(),
    });
  });

  test('refreshes record after update when attachment field changed', async () => {
    const store = newStore({ Id: 'u1', Avatar: 'bind-new' });
    const controller = createFormController(store);

    controller.vm.mode = 'edit';
    controller.vm.original = { Id: 'u1', Avatar: 'bind-old', Username: 'admin' } as any;
    controller.vm.draft = { Id: 'u1', Avatar: 'bind-new', Username: 'admin' } as any;

    await controller.submit();

    expect(store.UpdateById).toHaveBeenCalledTimes(1);
    expect(executeMock).toHaveBeenCalledTimes(1);
    expect((controller.vm.original as any)?.Avatar?.attachmentBindingId).toBe('bind-new');
    expect((controller.vm.original as any)?.Avatar?.previewUrl).toContain('/_document/bindings/bind-new/content');
    expect(handoffSetMock).toHaveBeenCalledWith('u1', expect.objectContaining({ Id: 'u1' }));
  });

  test('refreshes record after update when binary attachment field changed', async () => {
    executeMock.mockResolvedValueOnce({
      kind: 'search',
      rows: [
        {
          kind: 'record',
          key: 'u2',
          payload: {
            Id: 'u2',
            IdentityDoc: {
              attachmentBindingId: 'bind-doc-new',
              fileName: 'passport.pdf',
              displayName: 'passport.pdf',
            },
            Username: 'admin',
          },
          raw: {},
        },
      ],
      total: 1,
      ts: Date.now(),
    });

    const store = newStore(
      { Id: 'u2', IdentityDoc: 'bind-doc-new' },
      {
        IdentityDoc: { type: 'binary' },
        Username: { type: 'varchar' },
      }
    );
    const controller = createFormController(store);

    controller.vm.mode = 'edit';
    controller.vm.original = { Id: 'u2', IdentityDoc: 'bind-doc-old', Username: 'admin' } as any;
    controller.vm.draft = { Id: 'u2', IdentityDoc: 'bind-doc-new', Username: 'admin' } as any;

    await controller.submit();

    expect(store.UpdateById).toHaveBeenCalledTimes(1);
    expect(executeMock).toHaveBeenCalledTimes(1);
    expect((controller.vm.original as any)?.IdentityDoc?.attachmentBindingId).toBe('bind-doc-new');
    expect((controller.vm.original as any)?.IdentityDoc?.fileName).toBe('passport.pdf');
    expect((controller.vm.original as any)?.IdentityDoc?.previewUrl).toBeUndefined();
    expect(handoffSetMock).toHaveBeenCalledWith('u2', expect.objectContaining({ Id: 'u2' }));
  });

  test('passes displayFileName in normalized update payload for attachment set envelope', async () => {
    executeMock.mockResolvedValueOnce({
      kind: 'search',
      rows: [
        {
          kind: 'record',
          key: 'u3',
          payload: {
            Id: 'u3',
            Avatar: {
              attachmentBindingId: 'bind-new',
              fileName: 'avatar-original.jpg',
              previewUrl: '/_document/bindings/bind-new/content?token=t',
            },
            Username: 'admin',
          },
          raw: {},
        },
      ],
      total: 1,
      ts: Date.now(),
    });

    const store = newStore({ Id: 'u3', Avatar: 'bind-new' });
    const controller = createFormController(store);

    controller.vm.mode = 'edit';
    controller.vm.original = { Id: 'u3', Avatar: 'bind-old', Username: 'admin' } as any;
    controller.vm.draft = {
      Id: 'u3',
      Avatar: {
        kind: 'set',
        attachmentObjectId: 'ao-new',
        displayName: 'avatar-original.jpg',
      },
      Username: 'admin',
    } as any;

    await controller.submit();

    expect(store.UpdateById).toHaveBeenCalledTimes(1);
    const normalizedPatch = store.UpdateById.mock.calls[0]?.[1] as Record<string, unknown>;
    expect(normalizedPatch?.Avatar).toEqual({
      attachmentObjectId: 'ao-new',
      displayFileName: 'avatar-original.jpg',
    });
  });

  test('does not refresh record when attachment field not changed', async () => {
    const store = newStore({ Id: 'u1', Username: 'root' });
    const controller = createFormController(store);

    controller.vm.mode = 'edit';
    controller.vm.original = { Id: 'u1', Avatar: 'bind-old', Username: 'admin' } as any;
    controller.vm.draft = { Id: 'u1', Avatar: 'bind-old', Username: 'root' } as any;

    await controller.submit();

    expect(store.UpdateById).toHaveBeenCalledTimes(1);
    expect(executeMock).not.toHaveBeenCalled();
    expect((controller.vm.original as any)?.Avatar).toBe('bind-old');
    expect((controller.vm.original as any)?.Username).toBe('root');
  });
});
