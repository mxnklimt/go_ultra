package domain

// RankFloor 是显示段位的最低分；低于此值返回段 0（UI 不显示徽章）。
// 换场景时连同 Dan 的边界逻辑一起调整。
const RankFloor = 1050

// Dan 把等级分映射为段位：
//   rating < RankFloor        -> 0（未定级）
//   tier = (rating-800)/200，封顶 9
func Dan(rating int) int {
	if rating < RankFloor {
		return 0
	}
	tier := (rating - 800) / 200
	if tier > 9 {
		return 9
	}
	return tier
}
