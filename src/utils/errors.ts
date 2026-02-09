import logger from '../services/logger';

/**
 * Extracts a readable error message from an unknown error type.
 * Handles Error instances, strings, and unknown types gracefully.
 */
export function extractErrorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === 'string') {
    return error;
  }
  return 'Unknown error';
}

/**
 * Logs an error with structured context and metadata.
 * Provides consistent error logging across the application.
 *
 * @param context - Descriptive context of what operation failed
 * @param error - The error that occurred
 * @param additionalData - Optional metadata to include in the log
 */
export function logError(
  context: string,
  error: unknown,
  additionalData?: Record<string, unknown>
): void {
  const message = extractErrorMessage(error);
  const stack = error instanceof Error ? error.stack : undefined;

  logger.error(context, {
    error: message,
    stack,
    timestamp: new Date().toISOString(),
    ...additionalData,
  });
}

/**
 * Custom error class for API-related errors.
 * Includes HTTP status code and response data for better debugging.
 */
export class ApiError extends Error {
  constructor(
    message: string,
    public statusCode?: number,
    public response?: unknown
  ) {
    super(message);
    this.name = 'ApiError';
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, ApiError);
    }
  }
}

/**
 * Custom error class for book matching failures.
 * Includes book data for debugging matching issues.
 */
export class BookMatchError extends Error {
  constructor(
    message: string,
    public bookData?: unknown
  ) {
    super(message);
    this.name = 'BookMatchError';
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, BookMatchError);
    }
  }
}

/**
 * Custom error class for synchronization failures.
 * Includes sync-specific context for debugging.
 */
export class SyncError extends Error {
  constructor(
    message: string,
    public syncData?: unknown
  ) {
    super(message);
    this.name = 'SyncError';
    if (Error.captureStackTrace) {
      Error.captureStackTrace(this, SyncError);
    }
  }
}
