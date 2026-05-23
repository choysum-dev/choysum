// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ErrorInfo } from './error_pb';
import { Code } from '@connectrpc/connect';
import { generateErrorId } from './utils';
import type { ErrorCodePattern, ErrorOptions } from './types';

export { Code as GrpcCode } from '@connectrpc/connect';
export type { ErrorCodePattern, ErrorOptions } from './types';

type ErrorConstructor<T> = abstract new (...args: never[]) => T;

/**
 * ChoysumError is the base error type and implements ErrorInfo directly.
 */
export class ChoysumError extends Error implements Omit<ErrorInfo, '$typeName'> {
  // Properties required for Protobuf serialization.
  readonly $typeName: 'oerrors.ErrorInfo' = 'oerrors.ErrorInfo';

  // ErrorInfo interface properties.
  errorId: string;
  domain: string;
  code: ErrorCodePattern;
  grpcCode: number = Code.Internal;
  metadata: { [key: string]: string } = {};

  // Error chain support.
  cause?: Error;

  /**
   * Creates a new error instance.
   */
  constructor(options: ErrorOptions) {
    super(options.message);
    this.name = 'ChoysumError';

    // Initialize the ErrorInfo interface properties.
    this.errorId = generateErrorId();
    this.domain = options.domain;
    this.code = options.code;
    this.message = options.message;
  }

  /**
   * Creates a ChoysumError from ErrorInfo.
   */
  static fromErrorInfo(info: ErrorInfo): ChoysumError {
    const error = new ChoysumError({
      domain: info.domain,
      code: info.code as ErrorCodePattern,
      message: info.message,
    });

    // Copy runtime ErrorInfo fields.
    error.errorId = info.errorId || generateErrorId();
    error.grpcCode = info.grpcCode;
    error.metadata = { ...info.metadata };

    return error;
  }

  /**
   * Wraps an error in a new ChoysumError that keeps the original cause.
   * Returns the original error when it already matches and re-wrapping is not forced.
   *
   * @param err The original error to wrap.
   * @param options Error options.
   * @param force Whether to force creation of a new error instance. Defaults to false.
   */
  static wrap(err: unknown, options: ErrorOptions, force: boolean = false): ChoysumError {
    // Return early when the existing error already matches and re-wrapping is not forced.
    if (!force && err instanceof ChoysumError) {
      // Return the original error when the domain and code already match.
      if (err.domain === options.domain && err.code === options.code) {
        return err;
      }
    }

    // Create a new ChoysumError.
    const choysumError = new ChoysumError(options);

    // Attach the original cause.
    if (err instanceof Error) {
      choysumError.cause = err;
    } else if (err !== null && err !== undefined) {
      choysumError.cause = new Error(String(err));
    }

    // Preserve useful details from the original error.
    if (err instanceof ChoysumError) {
      // Preserve the original ChoysumError status code.
      choysumError.withGrpcCode(err.grpcCode);

      // Merge metadata so the original error context is kept.
      choysumError.withMetadata({ ...err.metadata });
    }

    return choysumError;
  }

  /**
   * Sets the gRPC status code.
   */
  withGrpcCode(code: Code): this {
    this.grpcCode = code;
    return this;
  }

  /**
   * Adds metadata.
   */
  withMetadata(metadata: { [key: string]: string }): this {
    this.metadata = { ...this.metadata, ...metadata };
    return this;
  }

  /**
   * Sets the cause error to build the error chain.
   */
  withCause(cause: Error): this {
    this.cause = cause;
    return this;
  }

  /**
   * Returns the first error in the chain that matches the requested type.
   * Similar to Go's errors.As.
   */
  as<T>(errorType: ErrorConstructor<T>): T | null {
    // Check the current error.
    if (this instanceof errorType) {
      return this as unknown as T;
    }

    // Check the error chain.
    if (this.cause) {
      // Recurse when the nested error is also a ChoysumError.
      if (this.cause instanceof ChoysumError) {
        return this.cause.as(errorType);
      }

      // Otherwise check the nested error type directly.
      if (this.cause instanceof errorType) {
        return this.cause as unknown as T;
      }
    }

    return null;
  }

  /**
   * Reports whether the error matches the requested domain and optional code.
   * @param domain Error domain.
   * @param code Error code, when one is required.
   */
  is(domain: string, code?: string): boolean {
    // Check the current error.
    if (this.domain === domain) {
      // If a code is provided, it must match too. Otherwise only the domain matters.
      if (code === undefined || this.code === code) {
        return true;
      }
    }

    // Check the error chain.
    if (this.cause instanceof ChoysumError) {
      return this.cause.is(domain, code);
    }

    return false;
  }

  /**
   * Returns the full string representation of the error.
   */
  toString(): string {
    const metaStr = Object.entries(this.metadata)
      .map(([k, v]) => `${k}=${v}`)
      .join(', ');

    let result = `${this.name}: [${this.domain}] ${this.code}: ${this.message}`;
    if (metaStr) {
      result += ` (${metaStr})`;
    }

    // Append the error chain details.
    if (this.cause) {
      result += `\n  Caused by: ${this.cause.toString()}`;
    }

    return result;
  }

  /**
   * Returns the ErrorInfo representation.
   * Use this when the error must be serialized to protobuf.
   */
  toErrorInfo(): ErrorInfo {
    return {
      $typeName: 'oerrors.ErrorInfo',
      errorId: this.errorId,
      domain: this.domain,
      code: this.code,
      message: this.message,
      grpcCode: this.grpcCode,
      metadata: { ...this.metadata },
    };
  }
}

/**
 * Global error type matcher.
 * Similar to Go's errors.As.
 *
 * @param err The error to inspect.
 * @param errorType Error type constructor.
 * @returns The matching error instance, or null when none is found.
 */
export function errorAs<T>(err: unknown, errorType: ErrorConstructor<T>): T | null {
  if (err instanceof ChoysumError) {
    return err.as(errorType);
  }

  if (err instanceof errorType) {
    return err as T;
  }

  const cause = err instanceof Error ? getErrorCause(err) : undefined;
  if (cause instanceof Error) {
    return errorAs(cause, errorType);
  }

  return null;
}

/**
 * Global error domain and code matcher.
 * Similar to Go's errors.Is.
 *
 * @param err The error to inspect.
 * @param domain Error domain.
 * @param code Error code, when one is required.
 * @returns True when the error matches the requested domain and code.
 */
export function isErrorOf(err: unknown, domain: string, code?: string): err is ChoysumError {
  if (err instanceof ChoysumError) {
    return err.is(domain, code);
  }

  const cause = err instanceof Error ? getErrorCause(err) : undefined;
  if (cause instanceof Error) {
    return isErrorOf(cause, domain, code);
  }

  return false;
}

function getErrorCause(err: Error): unknown {
  return (err as Error & { cause?: unknown }).cause;
}
