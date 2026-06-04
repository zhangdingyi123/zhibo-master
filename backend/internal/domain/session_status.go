package domain

// SessionStatus 竞拍场次状态
type SessionStatus string

const (
	SessionStatusPending   SessionStatus = "pending"   // 未开始
	SessionStatusRunning   SessionStatus = "running"   // 进行中
	SessionStatusSettled   SessionStatus = "settled"   // 已成交
	SessionStatusCancelled SessionStatus = "cancelled" // 已取消（主播主动）
	SessionStatusFailed    SessionStatus = "failed"    // 异常
)

// sessionTransitions 合法状态迁移：from -> 允许的 to 集合
var sessionTransitions = map[SessionStatus]map[SessionStatus]struct{}{
	SessionStatusPending: {
		SessionStatusRunning:   {},
		SessionStatusCancelled: {},
		SessionStatusFailed:    {},
	},
	SessionStatusRunning: {
		SessionStatusSettled:   {},
		SessionStatusCancelled: {},
		SessionStatusFailed:    {},
	},
}

// CanTransition 是否允许从 from 迁移到 to
func CanTransition(from, to SessionStatus) bool {
	targets, ok := sessionTransitions[from]
	if !ok {
		return false
	}
	_, allowed := targets[to]
	return allowed
}

// IsTerminal 是否为终态
func (s SessionStatus) IsTerminal() bool {
	switch s {
	case SessionStatusSettled, SessionStatusCancelled, SessionStatusFailed:
		return true
	default:
		return false
	}
}

// CanModifyRules 未开始场次才允许修改规则
func (s SessionStatus) CanModifyRules() bool {
	return s == SessionStatusPending
}

// CanBid 是否允许出价
func (s SessionStatus) CanBid() bool {
	return s == SessionStatusRunning
}

// CanCancelByAnchor 主播是否可取消
func (s SessionStatus) CanCancelByAnchor() bool {
	return s == SessionStatusPending || s == SessionStatusRunning
}
