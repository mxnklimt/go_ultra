export const K_FACTOR = 16;
export const DEFAULT_RATING = 1500;

/** E_A = 1 / (1 + 10^((B - A) / 400))，与后端 ExpectedScore 一致 */
export function expectedScore(ratingA: number, ratingB: number): number {
  return 1 / (1 + Math.pow(10, (ratingB - ratingA) / 400));
}

/** half-away-from-zero 取整，与 Go math.Round 一致 */
function roundHalfAway(x: number): number {
  return Math.sign(x) * Math.round(Math.abs(x));
}

/** round(K * (1 - E_winner))，与后端 ComputeDelta 一致 */
export function computeDelta(winnerRating: number, loserRating: number): number {
  const eWinner = expectedScore(winnerRating, loserRating);
  return roundHalfAway(K_FACTOR * (1 - eWinner));
}

export type SelfResult = "win" | "loss";

export interface MatchPreview {
  self_before: number;
  opponent_before: number;
  self_after: number;
  opponent_after: number;
  self_delta: number;
  opponent_delta: number;
}

/**
 * 以 self 视角预览一局结果。
 * result=win => self 是赢家；result=loss => opponent 是赢家。
 * 与后端 MatchService.Record 的快照语义一致（零和）。
 */
export function previewMatch(
  selfRating: number,
  opponentRating: number,
  result: SelfResult,
): MatchPreview {
  if (result === "win") {
    const delta = computeDelta(selfRating, opponentRating);
    return {
      self_before: selfRating,
      opponent_before: opponentRating,
      self_after: selfRating + delta,
      opponent_after: opponentRating - delta,
      self_delta: delta,
      opponent_delta: -delta,
    };
  }
  // self loss => opponent wins
  const delta = computeDelta(opponentRating, selfRating);
  return {
    self_before: selfRating,
    opponent_before: opponentRating,
    self_after: selfRating - delta,
    opponent_after: opponentRating + delta,
    self_delta: -delta,
    opponent_delta: delta,
  };
}
