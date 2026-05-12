import "@testing-library/jest-dom/vitest";
import { afterEach, beforeEach, vi } from "vitest";

let consoleErrorSpy: ReturnType<typeof vi.spyOn>;
let stderrWriteSpy: ReturnType<typeof vi.spyOn>;
type StderrWrite = (
  chunk: string | Uint8Array,
  encodingOrCallback?: BufferEncoding | ((err?: Error | null) => void),
  callback?: (err?: Error | null) => void
) => boolean;

beforeEach(() => {
  const originalError = console.error;
  consoleErrorSpy = vi.spyOn(console, "error").mockImplementation((message?: unknown, ...args: unknown[]) => {
    if (typeof message === "string" && message.includes("Not implemented: navigation to another Document")) {
      return;
    }
    originalError(message, ...args);
  });
  const originalStderrWrite = process.stderr.write.bind(process.stderr) as StderrWrite;
  stderrWriteSpy = vi.spyOn(process.stderr, "write").mockImplementation(((chunk, encodingOrCallback, callback) => {
    const message = typeof chunk === "string" ? chunk : Buffer.from(chunk).toString("utf8");
    if (message.includes("Not implemented: navigation to another Document")) {
      return true;
    }
    return originalStderrWrite(chunk, encodingOrCallback, callback);
  }) as typeof process.stderr.write);
});

afterEach(() => {
  consoleErrorSpy.mockRestore();
  stderrWriteSpy.mockRestore();
});
