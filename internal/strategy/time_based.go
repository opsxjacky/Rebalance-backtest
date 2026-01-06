package strategy

import (
	"math"
	"time"

	"github.com/opsxjacky/Rebalance-backtest/pkg/types"
)

// TimeBasedStrategy 定期再平衡策略
type TimeBasedStrategy struct {
	name              string
	targetWeights     map[string]float64
	rebalanceInterval int // 再平衡间隔天数
	minTradeValue     float64
	daysSinceRebalance int
	lastRebalanceTime  time.Time
	isFirstDay        bool
}

// NewTimeBasedStrategy 创建定期再平衡策略
func NewTimeBasedStrategy(config types.StrategyConfig) *TimeBasedStrategy {
	interval := config.RebalanceInterval
	if interval <= 0 {
		interval = 30 // 默认30天
	}

	return &TimeBasedStrategy{
		name:              config.Name,
		targetWeights:     config.TargetWeights,
		rebalanceInterval: interval,
		minTradeValue:     config.MinTradeValue,
		daysSinceRebalance: 0,
		isFirstDay:        true,
	}
}

// Name 返回策略名称
func (s *TimeBasedStrategy) Name() string {
	if s.name != "" {
		return s.name
	}
	return "TimeBased"
}

// TargetWeights 返回目标权重
func (s *TimeBasedStrategy) TargetWeights(portfolio *types.Portfolio, prices map[string]float64) map[string]float64 {
	return s.targetWeights
}

// ShouldRebalance 判断是否需要再平衡
func (s *TimeBasedStrategy) ShouldRebalance(portfolio *types.Portfolio, prices map[string]float64) bool {
	// 第一天需要建仓
	if s.isFirstDay {
		return true
	}

	s.daysSinceRebalance++
	return s.daysSinceRebalance >= s.rebalanceInterval
}

// GenerateOrders 生成交易订单
func (s *TimeBasedStrategy) GenerateOrders(portfolio *types.Portfolio, targetWeights map[string]float64, prices map[string]float64) []types.Order {
	orders := make([]types.Order, 0)
	totalValue := portfolio.TotalValue

	if totalValue <= 0 {
		return orders
	}

	// 计算目标持仓金额
	targetValues := make(map[string]float64)
	for symbol, weight := range targetWeights {
		targetValues[symbol] = totalValue * weight
	}

	// 先处理卖出订单，释放现金
	sellOrders := make([]types.Order, 0)
	buyOrders := make([]types.Order, 0)

	for symbol, targetValue := range targetValues {
		price, ok := prices[symbol]
		if !ok || price <= 0 {
			continue
		}

		currentValue := 0.0
		if pos, exists := portfolio.Positions[symbol]; exists {
			currentValue = pos.Value
		}

		diff := targetValue - currentValue

		// 忽略小额交易
		if math.Abs(diff) < s.minTradeValue {
			continue
		}

		quantity := math.Abs(diff) / price

		if diff < 0 {
			sellOrders = append(sellOrders, types.Order{
				Symbol:   symbol,
				Side:     "SELL",
				Quantity: quantity,
				Price:    price,
			})
		} else {
			buyOrders = append(buyOrders, types.Order{
				Symbol:   symbol,
				Side:     "BUY",
				Quantity: quantity,
				Price:    price,
			})
		}
	}

	orders = append(orders, sellOrders...)
	orders = append(orders, buyOrders...)

	return orders
}

// OnRebalance 再平衡后回调
func (s *TimeBasedStrategy) OnRebalance() {
	s.lastRebalanceTime = time.Now()
	s.daysSinceRebalance = 0
	s.isFirstDay = false
}
