// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package injectappmodel unifies C2 inject for FieldDefault and AppSetting (and
// future app-scoped models) behind a Registry of Specs with Decide + Materialize,
// Supersede, and multi-app Bundle entry points. Sessions bind a read-only
// BuildCtx to a Registry and return Effects for the caller to apply.
package injectappmodel
