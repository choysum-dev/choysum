// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createFormController, type FormControllerDeps } from './formController';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

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
    UpdateById: fnRecorder(async () => updateResult),
  } as any;
}

function newAttachmentService() {
  return {
    PrepareUpload: fnRecorder(async () => ({
      uploadId: 'up_1',
      uploadTarget: { method: 'PUT', url: '/_document/uploads/up_1' },
    })),
    FinalizeUpload: fnRecorder(async () => ({ attachmentObjectId: 'ao_1' })),
  };
}

function refreshSnapshot(payload: Record<string, unknown>) {
  return {
    kind: 'search',
    rows: [
      {
        kind: 'record',
        key: String(payload.Id),
        payload,
        raw: {},
      },
    ],
    total: 1,
    ts: Date.now(),
  };
}

describe('formController submit attachment refresh', () => {
  let executeMock: CallRecorder & ((...args: any[]) => Promise<any>);
  let handoffSetMock: CallRecorder & ((...args: any[]) => void);
  let deps: FormControllerDeps;

  beforeEach(() => {
    executeMock = fnRecorder(async () =>
      refreshSnapshot({
        Id: 'u1',
        Avatar: {
          attachmentBindingId: 'bind-new',
          previewUrl: '/_document/bindings/bind-new/content?token=t',
          fileName: 'avatar.jpg',
        },
        Username: 'admin',
      })
    );
    handoffSetMock = fnRecorder();
    deps = {
      createStoreByModel: (modelName: string) => {
        if (modelName === 'document.AttachmentContent') return newAttachmentService();
        throw new Error(`unexpected model: ${modelName}`);
      },
      execute: executeMock as any,
      handoffSet: handoffSetMock as any,
    };
  });

  test('refreshes record after update when attachment field changed', async () => {
    const store = newStore({ Id: 'u1', Avatar: 'bind-new' });
    const controller = createFormController(store, deps);

    controller.vm.mode = 'edit';
    controller.vm.original = { Id: 'u1', Avatar: 'bind-old', Username: 'admin' } as any;
    controller.vm.draft = { Id: 'u1', Avatar: 'bind-new', Username: 'admin' } as any;

    await controller.submit();

    expect(store.UpdateById.calls.length).toBe(1);
    expect(executeMock.calls.length).toBe(1);
    expect((controller.vm.original as any)?.Avatar?.attachmentBindingId).toBe('bind-new');
    expect((controller.vm.original as any)?.Avatar?.previewUrl).toContain('/_document/bindings/bind-new/content');
    expect(handoffSetMock.calls.length).toBe(1);
    expect(handoffSetMock.calls[0]?.[0]).toBe('u1');
    expect(handoffSetMock.calls[0]?.[1]).toMatchObject({ Id: 'u1' });
  });

  test('refreshes record after update when binary attachment field changed', async () => {
    executeMock = fnRecorder(async () =>
      refreshSnapshot({
        Id: 'u2',
        IdentityDoc: {
          attachmentBindingId: 'bind-doc-new',
          fileName: 'passport.pdf',
          displayName: 'passport.pdf',
        },
        Username: 'admin',
      })
    );
    deps.execute = executeMock as any;

    const store = newStore(
      { Id: 'u2', IdentityDoc: 'bind-doc-new' },
      {
        IdentityDoc: { type: 'binary' },
        Username: { type: 'varchar' },
      }
    );
    const controller = createFormController(store, deps);

    controller.vm.mode = 'edit';
    controller.vm.original = { Id: 'u2', IdentityDoc: 'bind-doc-old', Username: 'admin' } as any;
    controller.vm.draft = { Id: 'u2', IdentityDoc: 'bind-doc-new', Username: 'admin' } as any;

    await controller.submit();

    expect(store.UpdateById.calls.length).toBe(1);
    expect(executeMock.calls.length).toBe(1);
    expect((controller.vm.original as any)?.IdentityDoc?.attachmentBindingId).toBe('bind-doc-new');
    expect((controller.vm.original as any)?.IdentityDoc?.fileName).toBe('passport.pdf');
    expect((controller.vm.original as any)?.IdentityDoc?.previewUrl).toBeUndefined();
    expect(handoffSetMock.calls[0]?.[0]).toBe('u2');
    expect(handoffSetMock.calls[0]?.[1]).toMatchObject({ Id: 'u2' });
  });

  test('passes displayFileName in normalized update payload for attachment set envelope', async () => {
    executeMock = fnRecorder(async () =>
      refreshSnapshot({
        Id: 'u3',
        Avatar: {
          attachmentBindingId: 'bind-new',
          fileName: 'avatar-original.jpg',
          previewUrl: '/_document/bindings/bind-new/content?token=t',
        },
        Username: 'admin',
      })
    );
    deps.execute = executeMock as any;

    const store = newStore({ Id: 'u3', Avatar: 'bind-new' });
    const controller = createFormController(store, deps);

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

    expect(store.UpdateById.calls.length).toBe(1);
    const normalizedPatch = store.UpdateById.calls[0]?.[1] as Record<string, unknown>;
    expect(normalizedPatch?.Avatar).toEqual({
      attachmentObjectId: 'ao-new',
      displayFileName: 'avatar-original.jpg',
    });
  });

  test('does not refresh record when attachment field not changed', async () => {
    const store = newStore({ Id: 'u1', Username: 'root' });
    const controller = createFormController(store, deps);

    controller.vm.mode = 'edit';
    controller.vm.original = { Id: 'u1', Avatar: 'bind-old', Username: 'admin' } as any;
    controller.vm.draft = { Id: 'u1', Avatar: 'bind-old', Username: 'root' } as any;

    await controller.submit();

    expect(store.UpdateById.calls.length).toBe(1);
    expect(executeMock.calls.length).toBe(0);
    expect((controller.vm.original as any)?.Avatar).toBe('bind-old');
    expect((controller.vm.original as any)?.Username).toBe('root');
  });

  test('handoff falls back to updated record when post-submit refresh returns no payload', async () => {
    executeMock = fnRecorder(async () => ({
      kind: 'search',
      rows: [],
      total: 0,
      ts: Date.now(),
    }));
    deps.execute = executeMock as any;

    const store = newStore({ Id: 'u4', Avatar: 'bind-new' });
    const controller = createFormController(store, deps);

    controller.vm.mode = 'edit';
    controller.vm.original = { Id: 'u4', Avatar: 'bind-old', Username: 'admin' } as any;
    controller.vm.draft = { Id: 'u4', Avatar: 'bind-new', Username: 'admin' } as any;

    await controller.submit();

    expect(executeMock.calls.length).toBe(1);
    expect(controller.vm.original).toBeNull();
    expect(handoffSetMock.calls.length).toBe(1);
    expect(handoffSetMock.calls[0]?.[0]).toBe('u4');
    expect(handoffSetMock.calls[0]?.[1]).toMatchObject({ Id: 'u4', Avatar: 'bind-new' });
  });
});
