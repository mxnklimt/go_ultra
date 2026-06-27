import { describe, it, expect } from "vitest";
import { ApiError, parseApiError } from "@/api/client";

describe("parseApiError", () => {
  it("parses backend { error: { code, message } } body", () => {
    const err = parseApiError({
      response: {
        status: 404,
        data: { error: { code: "PLAYER_NOT_FOUND", message: "玩家不存在" } },
      },
    });
    expect(err).toBeInstanceOf(ApiError);
    expect(err.code).toBe("PLAYER_NOT_FOUND");
    expect(err.message).toBe("玩家不存在");
    expect(err.status).toBe(404);
  });

  it("falls back to UNKNOWN when body has no error envelope", () => {
    const err = parseApiError({
      response: { status: 500, data: { something: "else" } },
    });
    expect(err.code).toBe("UNKNOWN");
    expect(err.status).toBe(500);
  });

  it("handles network error with no response", () => {
    const err = parseApiError({ message: "Network Error" });
    expect(err.code).toBe("NETWORK_ERROR");
    expect(err.status).toBe(0);
  });
});
