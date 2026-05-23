// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import AttachmentContent from './attachment_object';
import AttachmentBinding from './attachment_binding';
import AttachmentUploadSession from './upload_session';
import AttachmentMutationLedger from './attachment_mutation_ledger';

function listDeclaredConventionalServices(modelCtor: any): string[] {
  const source = Function.prototype.toString.call(modelCtor);
  const methodMatches = source.matchAll(/\bstatic\s+async\s+([A-Z][A-Za-z0-9_]*)\s*\(/g);
  const names = new Set<string>();
  for (const match of methodMatches) {
    if (match[1]) {
      names.add(match[1]);
    }
  }
  return Array.from(names).sort();
}

test('document model conventional public method whitelist includes task-driven GC surface', () => {
  expect(listDeclaredConventionalServices(AttachmentContent)).toEqual([
    'AuthorizeUploadPut',
    'CommitUploadPut',
    'FinalizeUpload',
    'PrepareUpload',
    'RunGarbageCollection',
  ]);
  expect(listDeclaredConventionalServices(AttachmentBinding)).toEqual(['BatchDescribe', 'Bind', 'ResolveDownloadContent', 'Unbind']);
  expect(listDeclaredConventionalServices(AttachmentUploadSession)).toEqual([]);
  expect(listDeclaredConventionalServices(AttachmentMutationLedger)).toEqual([]);
});
