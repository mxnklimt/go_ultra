import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import SubmitMatchDialog from "@/components/SubmitMatchDialog";
import * as playersApi from "@/api/players";
import * as matchesApi from "@/api/matches";

function setup() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <SubmitMatchDialog />
    </QueryClientProvider>,
  );
}

describe("SubmitMatchDialog", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.spyOn(playersApi, "getMe").mockResolvedValue({
      id: 1,
      username: "alice",
      rating: 1500.00,
      dan: 3,
      created_at: "2026-06-25T00:00:00Z",
    });
    vi.spyOn(playersApi, "listPlayers").mockResolvedValue([
      { id: 1, username: "alice", rating: 1500.00, dan: 3, games_played: 0, win_rate: 0 },
      { id: 2, username: "bob", rating: 1500.00, dan: 3, games_played: 0, win_rate: 0 },
    ]);
  });

  it("shows elo preview and submits via recordMatch", async () => {
    const recordSpy = vi
      .spyOn(matchesApi, "recordMatch")
      .mockResolvedValue({
        id: 10,
        winner_delta: 8.00,
        loser_delta: -8.00,
        new_self_rating: 1508.00,
        new_opponent_rating: 1492.00,
      });
    const user = userEvent.setup();
    setup();

    await user.click(await screen.findByTestId("open-submit"));
    await user.click(await screen.findByTestId("player-combobox-trigger"));
    await user.click(await screen.findByText("bob"));

    const preview = await screen.findByTestId("elo-preview");
    expect(preview).toHaveTextContent("1508.00");
    expect(preview).toHaveTextContent("+8.00");
    expect(preview).toHaveTextContent("1492.00");

    await user.click(screen.getByTestId("submit-match"));

    await waitFor(() => {
      expect(recordSpy).toHaveBeenCalledWith({
        opponent_username: "bob",
        result: "win",
      });
    });
  });
});
