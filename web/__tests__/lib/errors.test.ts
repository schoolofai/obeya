import { describe, it, expect } from "vitest";
import { AppError, ErrorCode } from "@/lib/errors";

describe("AppError", () => {
  it("sets name to AppError", () => {
    const err = new AppError(ErrorCode.UNAUTHORIZED, "not allowed");
    expect(err.name).toBe("AppError");
  });

  it("is an instance of Error", () => {
    const err = new AppError(ErrorCode.INTERNAL_ERROR, "boom");
    expect(err).toBeInstanceOf(Error);
  });

  it("stores the error code and message", () => {
    const err = new AppError(ErrorCode.BOARD_NOT_FOUND, "board 123 missing");
    expect(err.code).toBe(ErrorCode.BOARD_NOT_FOUND);
    expect(err.message).toBe("board 123 missing");
  });

  it("maps UNAUTHORIZED to 401", () => {
    const err = new AppError(ErrorCode.UNAUTHORIZED, "msg");
    expect(err.statusCode).toBe(401);
  });

  it("maps FORBIDDEN to 403", () => {
    const err = new AppError(ErrorCode.FORBIDDEN, "msg");
    expect(err.statusCode).toBe(403);
  });

  it("maps VALIDATION_ERROR to 400", () => {
    const err = new AppError(ErrorCode.VALIDATION_ERROR, "msg");
    expect(err.statusCode).toBe(400);
  });

  it("maps BOARD_NOT_FOUND to 404", () => {
    const err = new AppError(ErrorCode.BOARD_NOT_FOUND, "msg");
    expect(err.statusCode).toBe(404);
  });

  it("maps PLAN_LIMIT_REACHED to 403", () => {
    const err = new AppError(ErrorCode.PLAN_LIMIT_REACHED, "msg");
    expect(err.statusCode).toBe(403);
  });

  it("maps COUNTER_CONFLICT to 409", () => {
    const err = new AppError(ErrorCode.COUNTER_CONFLICT, "msg");
    expect(err.statusCode).toBe(409);
  });

  it("maps INTERNAL_ERROR to 500", () => {
    const err = new AppError(ErrorCode.INTERNAL_ERROR, "msg");
    expect(err.statusCode).toBe(500);
  });
});
