// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

(() => {
  'use strict';

  // Ensure the global namespace exists.
  globalThis.$choysum = globalThis.$choysum || {};

  // Mount the named function consistently as $choysum.__rpc__.
  $choysum.__rpc__ = async function (req) {
    // Check for $choysum.utils, which only exists at runtime.
    // Compiler environments do not provide $choysum.utils, so skip Decimal/BigInt conversions there.
    const isChoysumRuntime = typeof $choysum !== 'undefined' && typeof $choysum.utils === 'object';
    const { serialize, deserialize } = isChoysumRuntime ? $choysum.utils : { serialize: v => v, deserialize: v => v };

    /**
     * Parse a JSON-RPC request.
     */
    const parseRpcRequest = rawReq => {
      try {
        let rpcRequest = rawReq;
        if (rpcRequest.args == undefined) {
          rpcRequest.args = [];
        }

        globalThis.$choysum = globalThis.$choysum || {};
        globalThis.$choysum.request = rpcRequest;
        globalThis.$choysum.request.__choysumServiceState = globalThis.$choysum.request.__choysumServiceState || { depth: 0 };

        // Deserialize only in runtime environments.
        if (isChoysumRuntime) {
          try {
            if (Array.isArray(rpcRequest.args)) {
              rpcRequest.args = rpcRequest.args.map(deserialize);
            } else {
              rpcRequest.args = deserialize(rpcRequest.args);
            }
          } catch (err) {
            console.warn('Failed to deserialize RPC args:', err);
          }
        }

        return rpcRequest;
      } catch (err) {
        throw new Error('Invalid JSON-RPC request: ' + err.message);
      }
    };

    /**
     * Execute a JSON-RPC request.
     */
    const executeRpcRequest = async rpcRequest => {
      // 1. Normalize the service name.
      if (rpcRequest.service.split('.').length == 1) {
        rpcRequest.service = 'globalThis.' + rpcRequest.service;
      }

      // 2. Resolve the service path.
      const parts = rpcRequest.service.split('.');
      const methodName = parts.pop();
      const servicePath = parts.join('.');

      // 3. Resolve the service class.
      let cls = globalThis.pool?.get(servicePath);
      if (!cls) {
        try {
          cls = (0, eval)(servicePath);
        } catch (err) {
          throw new Error(`Service not found: ${rpcRequest.service}`);
        }
      }

      // 4. Check whether the method exists.
      if (cls?.[methodName] === undefined) {
        throw new ReferenceError(`${rpcRequest.service} is not defined`);
      }

      // 5. Execute the method.
      let result;
      if (typeof cls[methodName] === 'function') {
        result = cls[methodName](...rpcRequest.args);
      } else {
        const obj = new cls(...rpcRequest.args[0]);
        result = obj[methodName](...rpcRequest.args.slice(1));
      }

      // 6. Await async results.
      if (result instanceof Promise) {
        result = await result;
      }

      // 7. Serialize only at runtime unless the result is already plain.
      if (isChoysumRuntime) {
        try {
          const isPlain = result && typeof result === 'object' && result.__choysum_plain === true;
          if (!isPlain) {
            result = serialize(result);
          }
        } catch (err) {
          console.warn('Failed to serialize result:', err);
        }
      }

      return result;
    };

    // Execute the JSON-RPC request.
    const rpcRequest = parseRpcRequest(req);
    let rpcResponse = { id: rpcRequest.id, context: rpcRequest.context, routing: rpcRequest.routing };
    try {
      rpcResponse.result = await executeRpcRequest(rpcRequest);
      return rpcResponse;
    } catch (err) {
      throw err;
    } finally {
      if (globalThis.$choysum) {
        try {
          delete globalThis.$choysum.request?.__choysumServiceState;
        } catch {}
        try {
          delete globalThis.$choysum.request;
        } catch {}
      }
    }
  };
})();
