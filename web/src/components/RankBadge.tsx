import { danOf, danLabel, danColor } from "@/lib/rank";
import { cn } from "@/lib/utils";

interface RankBadgeProps {
  rating: number;
  className?: string;
}

export default function RankBadge({ rating, className }: RankBadgeProps) {
  const dan = danOf(rating);
  const color = danColor(dan);
  const label = danLabel(dan);
  return (
    <span
      data-testid="rank-badge"
      data-dan={dan}
      className={cn(
        "inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-semibold",
        className,
      )}
      style={{ color, borderColor: color }}
    >
      {label}
    </span>
  );
}
