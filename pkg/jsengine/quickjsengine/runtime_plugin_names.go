// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import "strings"

const (
	RuntimePluginConsole          = "console"
	RuntimePluginXid              = "xid"
	RuntimePluginDB               = "db"
	RuntimePluginAuth             = "auth"
	RuntimePluginCrypto           = "crypto"
	RuntimePluginFS               = "fs"
	RuntimePluginI18n             = "i18n"
	RuntimePluginGRPC             = "grpc"
	RuntimePluginDocumentStorage  = "document-storage"
	RuntimePluginModuleManagement = "module-management"
	RuntimePluginScriptVueSFC     = "script:vuesfc"
	RuntimePluginScriptChoysumRPC = "script:choysumrpc"
)

var defaultRuntimePluginNames = []string{
	RuntimePluginConsole,
	RuntimePluginXid,
	RuntimePluginDB,
	RuntimePluginAuth,
	RuntimePluginCrypto,
	RuntimePluginFS,
	RuntimePluginI18n,
	RuntimePluginGRPC,
	RuntimePluginDocumentStorage,
	RuntimePluginModuleManagement,
	RuntimePluginScriptVueSFC,
	RuntimePluginScriptChoysumRPC,
}

var replaceableRuntimePluginNames = []string{
	RuntimePluginConsole,
	RuntimePluginXid,
	RuntimePluginDB,
	RuntimePluginAuth,
	RuntimePluginCrypto,
	RuntimePluginFS,
	RuntimePluginI18n,
	RuntimePluginGRPC,
	RuntimePluginDocumentStorage,
	RuntimePluginModuleManagement,
}

var defaultOnlyRuntimePluginNames = []string{
	RuntimePluginScriptVueSFC,
	RuntimePluginScriptChoysumRPC,
}

var replaceableRuntimePluginNameSet = map[string]struct{}{
	RuntimePluginConsole:          {},
	RuntimePluginXid:              {},
	RuntimePluginDB:               {},
	RuntimePluginAuth:             {},
	RuntimePluginCrypto:           {},
	RuntimePluginFS:               {},
	RuntimePluginI18n:             {},
	RuntimePluginGRPC:             {},
	RuntimePluginDocumentStorage:  {},
	RuntimePluginModuleManagement: {},
}

var defaultOnlyRuntimePluginNameSet = map[string]struct{}{
	RuntimePluginScriptVueSFC:     {},
	RuntimePluginScriptChoysumRPC: {},
}

// DefaultRuntimePluginNames returns the ordered default QuickJS runtime plugin names
// used by the built-in default engine assembly.
func DefaultRuntimePluginNames() []string {
	return append([]string(nil), defaultRuntimePluginNames...)
}

// ReplaceableRuntimePluginNames returns the ordered plugin names that callers may
// replace while keeping the same runtime slot.
func ReplaceableRuntimePluginNames() []string {
	return append([]string(nil), replaceableRuntimePluginNames...)
}

// DefaultOnlyRuntimePluginNames returns the ordered plugin names that belong only
// to the built-in default assembly and should not be replaced as stable runtime slots.
func DefaultOnlyRuntimePluginNames() []string {
	return append([]string(nil), defaultOnlyRuntimePluginNames...)
}

func IsReplaceableRuntimePlugin(name string) bool {
	_, ok := replaceableRuntimePluginNameSet[strings.TrimSpace(name)]
	return ok
}

func IsDefaultOnlyRuntimePlugin(name string) bool {
	_, ok := defaultOnlyRuntimePluginNameSet[strings.TrimSpace(name)]
	return ok
}
