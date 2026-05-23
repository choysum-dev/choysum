// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

import type { GenFile } from '@bufbuild/protobuf/codegenv2';
import { fileDesc, serviceDesc } from '@bufbuild/protobuf/codegenv2';

export const file_minimal_service: GenFile = fileDesc('Cg1taW5pbWFsX3NlcnZpY2UucHJvdG8=');

export const DemoService = serviceDesc(file_minimal_service, 0);
