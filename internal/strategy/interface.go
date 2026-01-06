package strategy

import (
	"github.com/opsxjacky/Rebalance-backtest/pkg/types"
)

// RebalanceStrategy 再平衡策略接口
type RebalanceStrategy interface {
	// Name 策略名称
	Name() string

	// TargetWeights 计算目标权重
	TargetWeights(portfolio *types.Portfolio, prices map[string]float64) map[string]float64

	// ShouldRebalance 判断是否需要再平衡
	ShouldRebalance(portfolio *types.Portfolio, prices map[string]float64) bool

	// GenerateOrders 生成交易订单
	GenerateOrders(portfolio *types.Portfolio, targetWeights map[string]float64, prices map[string]float64) []types.Order

	// OnRebalance 再平衡后回调 (用于更新内部状态)
	OnRebalance()
}
