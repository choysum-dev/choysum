// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  __collectCreateAttachmentWriteActionsForTest,
  __isCreateAttachmentWritePipelineEnabledForTest,
  __rewriteCreateInputForAttachmentsForTest,
} from './model_create';
import {
  __clearAttachmentBindingForTest,
  __collectUpdateAttachmentWriteActionsForTest,
  __isUpdateAttachmentWritePipelineEnabledForTest,
  __rewriteUpdateInputForAttachmentsForTest,
} from './model_update';

test('create attachment pipeline collects set/clear/noop actions and rewrites payload', () => {
  const fields = new Map<string, { type?: string }>([
    ['Avatar', { type: 'binary' }],
    ['Banner', { type: 'image' }],
    ['Memo', { type: 'varchar' }],
  ]);

  const input = {
    Avatar: {
      kind: 'set',
      attachmentObjectId: 'ao-create',
      mutationId: 'mut-set',
      displayName: 'avatar.png',
      downloadDisposition: 'inline',
    },
    Banner: { kind: 'clear', mutationId: 'mut-clear' },
    Memo: 'unchanged',
  } as any;

  const actions = __collectCreateAttachmentWriteActionsForTest(fields as any, input);

  expect(actions.get('Avatar')).toMatchObject({
    kind: 'set',
    attachmentObjectId: 'ao-create',
    displayFileName: 'avatar.png',
    downloadDisposition: 'inline',
  });
  expect(actions.get('Banner')).toMatchObject({ kind: 'clear' });

  const rewritten = __rewriteCreateInputForAttachmentsForTest(input, actions);

  expect(rewritten.Avatar).toBeNull();
  expect(rewritten.Banner).toBeNull();
  expect(rewritten.Memo).toBe('unchanged');
});

test('update attachment pipeline collects noop and drops noop field from rewritten payload', () => {
  const fields = new Map<string, { type?: string }>([
    ['Avatar', { type: 'binary' }],
    ['Icon', { type: 'image' }],
    ['Name', { type: 'varchar' }],
  ]);

  const input = {
    Avatar: { kind: 'noop' },
    Icon: {
      kind: 'set',
      attachmentObjectId: 'ao-update',
      mutationId: 'mut-icon',
      fileName: 'updated-icon.jpg',
      downloadDisposition: 'attachment',
    },
    Name: 'updated',
  } as any;

  const actions = __collectUpdateAttachmentWriteActionsForTest(fields as any, input);

  expect(actions.get('Avatar')).toEqual({ kind: 'noop' });
  expect(actions.get('Icon')).toMatchObject({
    kind: 'set',
    attachmentObjectId: 'ao-update',
    displayFileName: 'updated-icon.jpg',
    downloadDisposition: 'attachment',
  });

  const rewritten = __rewriteUpdateInputForAttachmentsForTest(input, actions);

  expect(Object.prototype.hasOwnProperty.call(rewritten, 'Avatar')).toBe(false);
  expect(rewritten.Icon).toBeNull();
  expect(rewritten.Name).toBe('updated');
});

test('update clear pipeline falls back to active binding search when direct unbind is not found', async () => {
  const unbindCalls: any[] = [];
  const searchCalls: any[] = [];

  const service = {
    async Bind() {
      return { attachmentBindingId: 'unused' };
    },
    async Unbind(req: any) {
      unbindCalls.push(req);
      if (req.attachmentBindingId === 'bind-old') {
        throw { code: 'not_found', message: 'not found' };
      }
      return { ok: true };
    },
    async Search(condition: any, options: any) {
      searchCalls.push({ condition, options });
      return [{ Id: 'bind-active' }];
    },
  };

  await __clearAttachmentBindingForTest(service as any, 'demo.Asset', 'RID-42', 'Avatar', 'mut-clear', 'bind-old');

  expect(unbindCalls).toEqual([
    { attachmentBindingId: 'bind-old', mutationId: 'mut-clear', reason: 'clear' },
    { attachmentBindingId: 'bind-active', mutationId: 'mut-clear', reason: 'clear' },
  ]);
  expect(searchCalls.length).toBe(1);
});

test('update clear pipeline skips fallback search when direct unbind succeeds', async () => {
  const calls = {
    unbind: 0,
    search: 0,
  };

  const service = {
    async Bind() {
      return { attachmentBindingId: 'unused' };
    },
    Unbind: async (_req: any) => {
      calls.unbind += 1;
      return { ok: true };
    },
    Search: async () => {
      calls.search += 1;
      return [{ Id: 'bind-active' }];
    },
  };

  await __clearAttachmentBindingForTest(service as any, 'demo.Asset', 'RID-43', 'Avatar', 'mut-clear', 'bind-direct');

  expect(calls.unbind).toBe(1);
  expect(calls.search).toBe(0);
});

test('attachment write pipeline model exclusions remain explicit for storage internals', () => {
  expect(__isCreateAttachmentWritePipelineEnabledForTest('document.AttachmentObject')).toBe(false);
  expect(__isCreateAttachmentWritePipelineEnabledForTest('document.UploadSession')).toBe(false);
  expect(__isCreateAttachmentWritePipelineEnabledForTest('demo.Asset')).toBe(true);

  expect(__isUpdateAttachmentWritePipelineEnabledForTest('document.AttachmentObject')).toBe(false);
  expect(__isUpdateAttachmentWritePipelineEnabledForTest('document.UploadSession')).toBe(false);
  expect(__isUpdateAttachmentWritePipelineEnabledForTest('demo.Asset')).toBe(true);
});
