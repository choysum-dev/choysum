// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package quickjsengine

import (
	"fmt"
	"unicode/utf8"

	"github.com/buke/quickjs-go"
	"github.com/choysum-dev/choysum/pkg/jsengine"
)

const textEncodingPolyfillJSScript = `(function () {
	if (typeof globalThis.TextEncoder !== "function") {
		globalThis.TextEncoder = class TextEncoder {
			encode(input) {
				var str = String(input == null ? "" : input);
				var utf8 = unescape(encodeURIComponent(str));
				var out = new Uint8Array(utf8.length);
				for (var i = 0; i < utf8.length; i++) {
					out[i] = utf8.charCodeAt(i);
				}
				return out;
			}
		};
	}

	if (typeof globalThis.TextDecoder !== "function") {
		globalThis.TextDecoder = class TextDecoder {
			constructor(label, options) {
				this.fatal = !!(options && options.fatal);
			}

			decode(input) {
				if (input == null) {
					return "";
				}

				var bytes;
				if (input instanceof Uint8Array) {
					bytes = input;
				} else if (ArrayBuffer.isView(input)) {
					bytes = new Uint8Array(input.buffer, input.byteOffset, input.byteLength);
				} else if (input instanceof ArrayBuffer) {
					bytes = new Uint8Array(input);
				} else if (Array.isArray(input)) {
					bytes = new Uint8Array(input);
				} else {
					bytes = new Uint8Array(0);
				}

				var latin1 = "";
				for (var i = 0; i < bytes.length; i++) {
					latin1 += String.fromCharCode(bytes[i]);
				}

				try {
					return decodeURIComponent(escape(latin1));
				} catch (err) {
					if (this.fatal) {
						throw err;
					}
					return "\uFFFD";
				}
			}
		};
	}
})();`

const textEncodingPolyfillGoBridgeScript = `(function () {
	var bridge = globalThis.$choysum && globalThis.$choysum.__text_encoding;
	if (!bridge || typeof bridge.encode !== "function" || typeof bridge.decode !== "function") {
		throw new Error("text-encoding bridge is not available");
	}

	globalThis.TextEncoder = class TextEncoder {
		encode(input) {
			return bridge.encode(String(input == null ? "" : input));
		}
	};

	globalThis.TextDecoder = class TextDecoder {
		constructor(label, options) {
			this.fatal = !!(options && options.fatal);
		}

		decode(input) {
			if (input == null) {
				return "";
			}

			var bytes;
			if (input instanceof Uint8Array) {
				bytes = input;
			} else if (ArrayBuffer.isView(input)) {
				bytes = new Uint8Array(input.buffer, input.byteOffset, input.byteLength);
			} else if (input instanceof ArrayBuffer) {
				bytes = new Uint8Array(input);
			} else if (Array.isArray(input)) {
				bytes = new Uint8Array(input);
			} else {
				bytes = new Uint8Array(0);
			}

			return bridge.decode(bytes, this.fatal);
		}
	};
})();`

func installTextEncodingPolyfill(ctx *quickjs.Context, script string, fileName string) error {
	ret := ctx.Eval(script, quickjs.EvalFileName(fileName))
	if ret == nil {
		return fmt.Errorf("failed to install text-encoding polyfill: eval returned nil")
	}
	defer ret.Free()
	if ret.IsException() {
		return fmt.Errorf("failed to install text-encoding polyfill: %w", ctx.Exception())
	}
	return nil
}

// WithTextEncodingPolyfillJS installs the JavaScript TextEncoder/TextDecoder polyfill.
func WithTextEncodingPolyfillJS() jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*QuickjsEngine)
		return installTextEncodingPolyfill(jse.Ctx, textEncodingPolyfillJSScript, "polyfills/text-encoding.js")
	}
}

func textEncodingGoEncode(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	input := ""
	if len(args) > 0 {
		input = args[0].String()
	}
	return ctx.NewUint8Array([]byte(input))
}

func textEncodingGoDecode(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
	if len(args) == 0 {
		return ctx.NewString("")
	}
	bytes, err := toDecodeBytes(args[0])
	if err != nil {
		return ctx.ThrowError(err)
	}
	fatal := len(args) > 1 && args[1].Bool()
	if !utf8.Valid(bytes) {
		if fatal {
			return ctx.ThrowError(fmt.Errorf("invalid UTF-8 input"))
		}
		return ctx.NewString("\uFFFD")
	}
	return ctx.NewString(string(bytes))
}

func toDecodeBytes(v *quickjs.Value) ([]byte, error) {
	if v == nil || v.IsUndefined() || v.IsNull() {
		return nil, nil
	}
	if v.IsUint8Array() || v.IsUint8ClampedArray() {
		return v.ToUint8Array()
	}
	if v.IsByteArray() {
		return v.ToByteArray(uint(v.ByteLen()))
	}
	if v.IsTypedArray() {
		buffer := v.Get("buffer")
		defer buffer.Free()
		if buffer == nil || !buffer.IsByteArray() {
			return nil, fmt.Errorf("typed array buffer is not an ArrayBuffer")
		}
		raw, err := buffer.ToByteArray(uint(buffer.ByteLen()))
		if err != nil {
			return nil, err
		}
		byteOffset := v.Get("byteOffset")
		defer byteOffset.Free()
		byteLength := v.Get("byteLength")
		defer byteLength.Free()
		start := int(byteOffset.Int64())
		length := int(byteLength.Int64())
		end := start + length
		if start < 0 || length < 0 || end < 0 || end > len(raw) {
			return nil, fmt.Errorf("typed array bounds are invalid")
		}
		return raw[start:end], nil
	}
	return nil, fmt.Errorf("unsupported decode input type")
}

// WithTextEncodingPolyfillGoBridge installs TextEncoder/TextDecoder backed by Go callbacks.
func WithTextEncodingPolyfillGoBridge() jsengine.JsEngineOption {
	return func(jsEngine jsengine.JsEngine) error {
		jse := jsEngine.(*QuickjsEngine)
		globalsObj := jse.Ctx.Globals()

		choysumObj := globalsObj.Get("$choysum")
		if choysumObj.IsUndefined() {
			choysumObj = jse.Ctx.Object()
		}

		bridgeObj := jse.Ctx.Object()
		bridgeObj.Set("encode", jse.Ctx.Function(textEncodingGoEncode))
		bridgeObj.Set("decode", jse.Ctx.Function(textEncodingGoDecode))
		choysumObj.Set("__text_encoding", bridgeObj)
		globalsObj.Set("$choysum", choysumObj)

		return installTextEncodingPolyfill(jse.Ctx, textEncodingPolyfillGoBridgeScript, "polyfills/text-encoding-go-bridge.js")
	}
}
