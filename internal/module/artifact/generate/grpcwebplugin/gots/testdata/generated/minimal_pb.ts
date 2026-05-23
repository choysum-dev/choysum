// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import type { GenFile, GenMessage } from '@bufbuild/protobuf/codegenv2';
import { fileDesc, messageDesc } from '@bufbuild/protobuf/codegenv2';
import type { Message } from '@bufbuild/protobuf';

export const file_minimal: GenFile = fileDesc('Cg1taW5pbWFsLnByb3Rv');

export type Demo = Message<'sample.Demo'> & {
  id: string;
};

export const DemoSchema: GenMessage<Demo> = messageDesc(file_minimal, 0);
