export enum ErrorCode {
  UNAUTHORIZED = "UNAUTHORIZED",
  FORBIDDEN = "FORBIDDEN",
  INVALID_CREDENTIALS = "INVALID_CREDENTIALS",
  TOKEN_EXPIRED = "TOKEN_EXPIRED",
  TOKEN_NOT_FOUND = "TOKEN_NOT_FOUND",
  VALIDATION_ERROR = "VALIDATION_ERROR",
  BOARD_NOT_FOUND = "BOARD_NOT_FOUND",
  ITEM_NOT_FOUND = "ITEM_NOT_FOUND",
  ORG_NOT_FOUND = "ORG_NOT_FOUND",
  PLAN_NOT_FOUND = "PLAN_NOT_FOUND",
  USER_NOT_FOUND = "USER_NOT_FOUND",
  PLAN_LIMIT_REACHED = "PLAN_LIMIT_REACHED",
  COUNTER_CONFLICT = "COUNTER_CONFLICT",
  EMAIL_ALREADY_EXISTS = "EMAIL_ALREADY_EXISTS",
  SLUG_ALREADY_EXISTS = "SLUG_ALREADY_EXISTS",
  INTERNAL_ERROR = "INTERNAL_ERROR",
}

const STATUS_MAP: Record<ErrorCode, number> = {
  [ErrorCode.UNAUTHORIZED]: 401,
  [ErrorCode.FORBIDDEN]: 403,
  [ErrorCode.INVALID_CREDENTIALS]: 401,
  [ErrorCode.TOKEN_EXPIRED]: 401,
  [ErrorCode.TOKEN_NOT_FOUND]: 404,
  [ErrorCode.VALIDATION_ERROR]: 400,
  [ErrorCode.BOARD_NOT_FOUND]: 404,
  [ErrorCode.ITEM_NOT_FOUND]: 404,
  [ErrorCode.ORG_NOT_FOUND]: 404,
  [ErrorCode.PLAN_NOT_FOUND]: 404,
  [ErrorCode.USER_NOT_FOUND]: 404,
  [ErrorCode.PLAN_LIMIT_REACHED]: 403,
  [ErrorCode.COUNTER_CONFLICT]: 409,
  [ErrorCode.EMAIL_ALREADY_EXISTS]: 409,
  [ErrorCode.SLUG_ALREADY_EXISTS]: 409,
  [ErrorCode.INTERNAL_ERROR]: 500,
};

export class AppError extends Error {
  public readonly code: ErrorCode;
  public readonly statusCode: number;

  constructor(code: ErrorCode, message: string) {
    super(message);
    this.name = "AppError";
    this.code = code;
    this.statusCode = STATUS_MAP[code];
  }
}
