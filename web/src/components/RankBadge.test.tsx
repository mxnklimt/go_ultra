import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import RankBadge from "@/components/RankBadge";

describe("RankBadge", () => {
  it("shows 未定级 with gray color below floor", () => {
    render(<RankBadge rating={1000} />);
    const badge = screen.getByTestId("rank-badge");
    expect(badge).toHaveTextContent("未定级");
    expect(badge).toHaveAttribute("data-dan", "0");
    expect(badge.style.color).toBe("rgb(156, 163, 175)"); // #9ca3af
  });

  it("shows 段 3 blue for rating 1500", () => {
    render(<RankBadge rating={1500} />);
    const badge = screen.getByTestId("rank-badge");
    expect(badge).toHaveTextContent("段 3");
    expect(badge).toHaveAttribute("data-dan", "3");
    expect(badge.style.color).toBe("rgb(74, 158, 255)"); // #4a9eff
  });

  it("shows 段 9 red for very high rating", () => {
    render(<RankBadge rating={2700} />);
    const badge = screen.getByTestId("rank-badge");
    expect(badge).toHaveTextContent("段 9");
    expect(badge).toHaveAttribute("data-dan", "9");
    expect(badge.style.color).toBe("rgb(240, 128, 128)"); // #f08080
  });
});
