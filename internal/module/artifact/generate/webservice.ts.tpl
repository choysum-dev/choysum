// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/* eslint-disable */
{{$app := .}}
import { CreateWebClient, CreateWebApiService } from '@/core/web/rpc';
import * as {{$app.Name}} from './pb/{{$app.Name}}_pb';

{{range $model := .Models}}
//{{$model.Name}} Service - {{ConvertPath $model.Path}}
const {{$model.Name}}Client = CreateWebClient({{$app.Name}}.{{$model.Name}});
import type {{$model.Name}} from '{{ConvertPath $model.Path}}';
export const {{$model.Name}}Service = {
  {{- range $service := $model.Services }}
  {{$service.Name}}: CreateWebApiService<typeof {{$model.Name}}.{{$service.Name}}{{ConvertTypeParam $model $service}}>(
    {{$model.Name}}Client,
    '{{$app.Name}}.{{$model.Name}}',
    '{{ToCamel $service.Name}}',
    (...args) => {
      return {{ConvertArgs $service}};
    },
    {{ConvertReturnType $service}}
  ),
  {{- end}}
};

{{end}}

