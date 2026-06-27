import axios, { type AxiosError } from "axios";
import type { ApiErrorBody } from "@/api/types";

export class ApiError extends Error {
  code: string;
  status: number;
  constructor(code: string, message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

/** 把 axios 抛出的错误解析为统一的 ApiError。纯函数，便于单测。 */
export function parseApiError(error: unknown): ApiError {
  const ax = error as Partial<AxiosError<ApiErrorBody>>;
  const response = ax.response;
  if (response) {
    const body = response.data as ApiErrorBody | undefined;
    if (body && body.error && body.error.code) {
      return new ApiError(
        body.error.code,
        body.error.message,
        response.status,
      );
    }
    return new ApiError("UNKNOWN", "未知错误", response.status);
  }
  return new ApiError("NETWORK_ERROR", "网络错误", 0);
}

export const client = axios.create({
  baseURL: "/api",
  withCredentials: true,
  headers: { "Content-Type": "application/json" },
});

client.interceptors.response.use(
  (res) => res,
  (error) => {
    const apiError = parseApiError(error);
    if (apiError.status === 401) {
      if (window.location.pathname !== "/login") {
        window.location.assign("/login");
      }
    }
    return Promise.reject(apiError);
  },
);
