import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import AuthGuard from "@/components/AuthGuard";
import * as playersApi from "@/api/players";

function renderWithAuth(initialPath: string) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route
            path="/me"
            element={
              <AuthGuard>
                <div data-testid="protected">secret</div>
              </AuthGuard>
            }
          />
          <Route path="/login" element={<div data-testid="login">login</div>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("AuthGuard", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("renders children when authenticated", async () => {
    vi.spyOn(playersApi, "getMe").mockResolvedValue({
      id: 1,
      username: "alice",
      rating: 1500,
      dan: 3,
      created_at: "2026-06-25T00:00:00Z",
    });
    renderWithAuth("/me");
    expect(await screen.findByTestId("protected")).toBeInTheDocument();
  });

  it("redirects to /login when unauthenticated", async () => {
    vi.spyOn(playersApi, "getMe").mockRejectedValue(new Error("401"));
    renderWithAuth("/me");
    expect(await screen.findByTestId("login")).toBeInTheDocument();
  });
});
