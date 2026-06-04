package domain

// AuctionRules 竞拍规则（嵌入场次，未开始可修改）
// 金额单位均为「分」。
type AuctionRules struct {
	StartingPrice       int64  `json:"startingPrice"`       // 起拍价，0 表示 0 元起拍
	BidIncrement        int64  `json:"bidIncrement"`        // 加价幅度
	CapPrice            *int64 `json:"capPrice,omitempty"`  // 封顶价，nil 表示无封顶
	DurationSec         uint32 `json:"durationSec"`         // 基础竞拍时长（秒）
	ExtendThresholdSec  uint32 `json:"extendThresholdSec"`  // 结束前 N 秒内有出价触发延时
	ExtendSec           uint32 `json:"extendSec"`           // 单次延时秒数，建议 10–30
}

// Validate 校验规则合法性（发布/修改时调用）
func (r AuctionRules) Validate() error {
	if r.StartingPrice < 0 {
		return ErrInvalidStartingPrice
	}
	if r.BidIncrement <= 0 {
		return ErrInvalidBidIncrement
	}
	if r.CapPrice != nil && *r.CapPrice < r.StartingPrice {
		return ErrCapBelowStarting
	}
	if r.DurationSec == 0 {
		return ErrInvalidDuration
	}
	if r.ExtendThresholdSec == 0 {
		return ErrInvalidExtendThreshold
	}
	if r.ExtendSec < 10 || r.ExtendSec > 30 {
		return ErrInvalidExtendSec
	}
	return nil
}

// MinNextBid 计算下一笔最低合法出价（分）
func (r AuctionRules) MinNextBid(currentPrice int64, hasBids bool) int64 {
	if !hasBids {
		return r.StartingPrice
	}
	return currentPrice + r.BidIncrement
}

// IsCapReached 是否达到封顶价
func (r AuctionRules) IsCapReached(price int64) bool {
	return r.CapPrice != nil && price >= *r.CapPrice
}
