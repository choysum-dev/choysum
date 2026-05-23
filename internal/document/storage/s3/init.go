// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package s3

import pkgstorage "github.com/choysum-dev/choysum/pkg/storage"

func init() {
	pkgstorage.Register("s3", NewStoredContentDriver)
}
