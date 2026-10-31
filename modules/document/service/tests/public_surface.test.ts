// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import AttachmentContent from '../models/attachment_object';
import AttachmentBinding from '../models/attachment_binding';
import AttachmentUploadSession from '../models/upload_session';
import AttachmentMutationLedger from '../models/attachment_mutation_ledger';

function listDeclaredStaticAsyncPascalMethods(modelCtor: any): string[] {
  const source = Function.prototype.toString.call(modelCtor);
  // TS erases the `public` keyword; scan static async PascalCase and treat the
  // interactive whitelist separately from worker-only methods.
  const methodMatches = source.matchAll(/\bstatic\s+async\s+([A-Z][A-Za-z0-9_]*)\s*\(/g);
  const names = new Set<string>();
  for (const match of methodMatches) {
    if (match[1]) {
      names.add(match[1]);
    }
  }
  return Array.from(names).sort();
}

test('document model conventional public method whitelist', () => {
  const attachmentContentMethods = listDeclaredStaticAsyncPascalMethods(AttachmentContent);
  expect(attachmentContentMethods.filter(name => name !== 'RunGarbageCollection')).toEqual([
    'AuthorizeUploadPut',
    'CommitUploadPut',
    'FinalizeUpload',
    'PrepareUpload',
  ]);
  // Worker routing target: present on the class, not a public interactive RPC.
  expect(typeof (AttachmentContent as { RunGarbageCollection?: unknown }).RunGarbageCollection).toBe('function');
  expect(listDeclaredStaticAsyncPascalMethods(AttachmentBinding)).toEqual([
    'BatchDescribe',
    'Bind',
    'ResolveDownloadContent',
    'Unbind',
  ]);
  expect(listDeclaredStaticAsyncPascalMethods(AttachmentUploadSession)).toEqual([]);
  expect(listDeclaredStaticAsyncPascalMethods(AttachmentMutationLedger)).toEqual([]);
});
