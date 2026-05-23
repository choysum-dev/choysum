// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/* eslint-disable */
{{$app := .App}}
import { CreateServerApiService, registerServiceFactory } from '@/core/service/rpc';

const protoFiles = [
  {{- range $proto := .ProtoFiles }}
  { path: '{{$proto.RegisterPath}}', content: {{$proto.EncodedContent}} },
  {{- end}}
];

let protoFilesRegistered = false;
let protoRegistrationScheduled = false;
let protoRegistrationRetryCount = 0;
const maxProtoRegistrationRetry = 200;

function ensureProtoFilesRegistered(): boolean {
  if (protoFilesRegistered) {
    return true;
  }
  const g = globalThis as any;
  if (!g.$choysum || !g.$choysum.grpc || typeof g.$choysum.grpc.registerProto !== 'function') {
    return false;
  }
  for (const proto of protoFiles) {
    g.$choysum.grpc.registerProto(proto.path, proto.content);
  }
  protoFilesRegistered = true;
  return true;
}

function scheduleProtoRegistration() {
  if (protoFilesRegistered || protoRegistrationScheduled) {
    return;
  }
  if (typeof setTimeout !== 'function') {
    return;
  }
  protoRegistrationScheduled = true;

  const tick = () => {
    if (ensureProtoFilesRegistered()) {
      return;
    }
    protoRegistrationRetryCount += 1;
    if (protoRegistrationRetryCount >= maxProtoRegistrationRetry) {
      protoRegistrationScheduled = false;
      return;
    }
    setTimeout(tick, 10);
  };

  setTimeout(tick, 0);
}

ensureProtoFilesRegistered();
scheduleProtoRegistration();

{{range $model := .App.Models}}
//{{$model.Name}} Service - {{ConvertPath $model.Path}}
import type {{$model.Name}} from '{{ConvertPath $model.Path}}';
export const {{$model.Name}}Service = {
  {{- range $service := $model.Services }}
  {{$service.Name}}: CreateServerApiService<typeof {{$model.Name}}.{{$service.Name}}{{ConvertTypeParam $model $service}}>(
    '{{$app.Name}}.{{$model.Name}}',
    '{{$service.Name}}',
    (...args) => {
      ensureProtoFilesRegistered();
      return {{ConvertArgs $service}};
    },
    {{ConvertReturnType $service}}
  ),
  {{- end}}
};
registerServiceFactory('{{$app.Name}}.{{$model.Name}}', () => {{$model.Name}}Service);

{{end}}
