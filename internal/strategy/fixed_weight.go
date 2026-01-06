package strategy

import (
	"math"
	"time"

	"github.com/opsxjacky/Rebalance-backtest/pkg/types"
)

// FixedWeightStrategy 固定权重再平衡策略
type FixedWeightStrategy struct {
	name                 string
	targetWeights        map[string]float64
	threshold            float64 // 偏离阈值，触发再平衡
	minTradeValue        float64 // 最小交易金额
	minRebalanceInterval int     // 最小再平衡间隔天数
	lastRebalanceTime    time.Time
	daysSinceRebalance   int
}

// NewFixedWeightStrategy 创建固定权重策略
func NewFixedWeightStrategy(config types.StrategyConfig) *FixedWeightStrategy {
	return &FixedWeightStrategy{
		name:                 config.Name,
		targetWeights:        config.TargetWeights,
		threshold:            config.Threshold,
		minTradeValue:        config.MinTradeValue,
		minRebalanceInterval: config.MinRebalanceInterval,
		daysSinceRebalance:   0,
	}
}

// Name 返回策略名称
func (s *FixedWeightStrategy) Name() string {
	if s.name != "" {
		return s.name
	}
	return "FixedWeight"
}

// TargetWeights 返回目标权重
func (s *FixedWeightStrategy) TargetWeights(portfolio *types.Portfolio, prices map[string]float64) map[string]float64 {
	return s.targetWeights
}

// ShouldRebalance 判断是否需要再平衡
func (s *FixedWeightStrategy) ShouldRebalance(portfolio *types.Portfolio, prices map[string]float64) bool {
	// 检查最小再平衡间隔
	s.daysSinceRebalance++
	if s.minRebalanceInterval > 0 && s.daysSinceRebalance < s.minRebalanceInterval {
		return false
	}

	// 如果阈值为0，则每次都再平衡
	if s.threshold <= 0 {
		return true
	}

	// 计算当前权重与目标权重的偏离
	currentWeights := portfolio.GetWeights()

	for symbol, targetWeight := range s.targetWeights {
		currentWeight, ok := currentWeights[symbol]
		if !ok {
			currentWeight = 0
		}

		deviation := math.Abs(currentWeight - targetWeight)
		if deviation > s.threshold {
			return true
		}
	}

	return false
}

// GenerateOrders 生成交易订单
func (s *FixedWeightStrategy) GenerateOrders(portfolio *types.Portfolio, targetWeights map[string]float64, prices map[string]float64) []types.Order {
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
			// 需要卖出
			sellOrders = append(sellOrders, types.Order{
				Symbol:   symbol,
				Side:     "SELL",
				Quantity: quantity,
				Price:    price,
			})
		} else {
			// 需要买入
			buyOrders = append(buyOrders, types.Order{
				Symbol:   symbol,
				Side:     "BUY",
				Quantity: quantity,
				Price:    price,
			})
		}
	}

	// 先添加卖出订单，再添加买入订单
	orders = append(orders, sellOrders...)
	orders = append(orders, buyOrders...)

	return orders
}

// OnRebalance 再平衡后回调
func (s *FixedWeightStrategy) OnRebalance() {
	s.lastRebalanceTime = time.Now()
	s.daysSinceRebalance = 0
}

// SetThreshold 设置阈值
func (s *FixedWeightStrategy) SetThreshold(threshold float64) {
	s.threshold = threshold
}

// SetMinTradeValue 设置最小交易金额
func (s *FixedWeightStrategy) SetMinTradeValue(minValue float64) {
	s.minTradeValue = minValue
}
