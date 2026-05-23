// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"fmt"

	"github.com/choysum-dev/choysum/pkg/jsengine"
)

// EngineOption holds configuration options for a QuickJS engine instance.
type EngineOption struct {
	Timeout            uint64 `json:"timeout"`            // Script execution timeout in seconds (0 = no timeout)
	MemoryLimit        uint64 `json:"memoryLimit"`        // Memory limit in bytes (0 = no limit)
	GCThreshold        int64  `json:"gcThreshold"`        // GC threshold in bytes (-1 = disable, 0 = default)
	MaxStackSize       uint64 `json:"maxStackSize"`       // Stack size in bytes (0 = default)
	CanBlock           bool   `json:"canBlock"`           // Whether the runtime can block (for async operations)
	EnableModuleImport bool   `json:"enableModuleImport"` // Enable ES6 module import support
	Strip              int    `json:"strip"`              // Strip level for bytecode compilation
}

// Option is a function that applies a configuration to an Engine instance.
// type Option func(*Engine) error

// WithGCThreshold sets the garbage collection threshold for the engine.
// Use -1 to disable automatic GC, 0 for default, or a positive value for a custom threshold.
func WithGCThreshold(threshold int64) jsengine.JsEngineOption {
	return func(engine jsengine.JsEngine) error {
		e := engine.(*QuickjsEngine)
		if threshold < -1 {
			return fmt.Errorf("invalid GC threshold: %d", threshold)
		}
		e.Option.GCThreshold = threshold
		e.Runtime.SetGCThreshold(threshold)
		return nil
	}
}

// WithMemoryLimit sets the memory limit for the JavaScript runtime in bytes.
// If limit is 0, there is no memory limit.
func WithMemoryLimit(limit uint64) jsengine.JsEngineOption {
	return func(engine jsengine.JsEngine) error {
		e := engine.(*QuickjsEngine)
		// 0 means no limit, so it's allowed
		e.Option.MemoryLimit = limit
		e.Runtime.SetMemoryLimit(limit)
		return nil
	}
}

// WithTimeout sets the script execution timeout in seconds.
// If timeout is 0, there is no timeout.
func WithTimeout(timeout uint64) jsengine.JsEngineOption {
	return func(engine jsengine.JsEngine) error {
		e := engine.(*QuickjsEngine)
		// 0 means no timeout, so it's allowed
		e.Option.Timeout = timeout
		return nil
	}
}

// WithMaxStackSize sets the stack size for the JavaScript runtime in bytes.
// If size is 0, the default stack size is used.
func WithMaxStackSize(size uint64) jsengine.JsEngineOption {
	return func(engine jsengine.JsEngine) error {
		e := engine.(*QuickjsEngine)
		// 0 means use default stack size, so it's allowed
		e.Option.MaxStackSize = size
		e.Runtime.SetMaxStackSize(size)
		return nil
	}
}

// WithCanBlock enables or disables blocking operations in the runtime.
// Enabling this may affect performance and security.
func WithCanBlock(canBlock bool) jsengine.JsEngineOption {
	return func(engine jsengine.JsEngine) error {
		e := engine.(*QuickjsEngine)
		e.Option.CanBlock = canBlock
		e.Runtime.SetCanBlock(canBlock)
		return nil
	}
}

// WithEnableModuleImport enables or disables ES6 module import support.
// Enabling this may have security implications.
func WithEnableModuleImport(enable bool) jsengine.JsEngineOption {
	return func(engine jsengine.JsEngine) error {
		e := engine.(*QuickjsEngine)
		e.Option.EnableModuleImport = enable
		e.Runtime.SetModuleImport(enable)
		return nil
	}
}

// WithStrip sets the strip level for bytecode compilation.
// 0 = no stripping, higher values strip more debug information.
func WithStrip(strip int) jsengine.JsEngineOption {
	return func(engine jsengine.JsEngine) error {
		e := engine.(*QuickjsEngine)
		if strip < 0 || strip > 2 {
			return fmt.Errorf("invalid strip level: %d", strip)
		}
		e.Option.Strip = strip
		e.Runtime.SetStripInfo(strip)
		return nil
	}
}
