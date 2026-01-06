package cost

import (
	"math"

	"github.com/opsxjacky/Rebalance-backtest/pkg/types"
)

// CostModel 成本模型接口
type CostModel interface {
	// CalculateCost 计算交易成本
	CalculateCost(trade types.Trade) float64

	// CalculateSlippage 计算滑点
	CalculateSlippage(price float64, side string) float64
}

// DefaultCostModel 默认成本模型
type DefaultCostModel struct {
	CommissionRate float64 // 佣金率
	MinCommission  float64 // 最低佣金
	SlippageRate   float64 // 滑点率
	TaxRate        float64 // 税率 (卖出时收取)
}

// NewDefaultCostModel 创建默认成本模型
func NewDefaultCostModel(config types.CostConfig) *DefaultCostModel {
	return &DefaultCostModel{
		CommissionRate: config.CommissionRate,
		MinCommission:  config.MinCommission,
		SlippageRate:   config.SlippageRate,
		TaxRate:        config.TaxRate,
	}
}

// NewZeroCostModel 创建零成本模型 (用于测试)
func NewZeroCostModel() *DefaultCostModel {
	return &DefaultCostModel{
		CommissionRate: 0,
		MinCommission:  0,
		SlippageRate:   0,
		TaxRate:        0,
	}
}

// CalculateCost 计算交易成本
func (m *DefaultCostModel) CalculateCost(trade types.Trade) float64 {
	tradeValue := math.Abs(trade.Quantity * trade.Price)

	// 佣金
	commission := tradeValue * m.CommissionRate
	if commission < m.MinCommission && tradeValue > 0 {
		commission = m.MinCommission
	}

	// 税费 (仅卖出时收取)
	var tax float64
	if trade.Side == "SELL" {
		tax = tradeValue * m.TaxRate
	}

	return commission + tax
}

// CalculateSlippage 计算滑点调整后的价格
func (m *DefaultCostModel) CalculateSlippage(price float64, side string) float64 {
	if side == "BUY" {
		// 买入时价格上浮
		return price * (1 + m.SlippageRate)
	}
	// 卖出时价格下浮
	return price * (1 - m.SlippageRate)
}

// CalculateTotalCost 计算总成本 (包括滑点损失)
func (m *DefaultCostModel) CalculateTotalCost(trade types.Trade) float64 {
	baseCost := m.CalculateCost(trade)
	slippageCost := math.Abs(trade.Quantity * trade.Price * m.SlippageRate)
	return baseCost + slippageCost
}
