// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

// Package typecheck runs Go-native TypeScript (and later Vue) type checking
// via typescript-go-internal, without invoking Node or vue-tsc.
//
// ScopeService covers app-root and service trees; ScopeNoVue also covers web
// TS/TSX and embeds vite/client plus subpath ambient declarations.
// ScopeAll adds web .vue files via Coder service-script overlays (Strategy B).
package typecheck
