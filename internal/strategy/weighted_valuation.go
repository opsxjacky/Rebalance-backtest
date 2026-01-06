package strategy

import (
	"math"
	"time"

	"github.com/opsxjacky/Rebalance-backtest/pkg/types"
)

// WeightedValuationStrategy 权重偏离+估值驱动再平衡策略
// 用于平安证券账户，结合权重偏离和估值信号
type WeightedValuationStrategy struct {
	name                 string
	targetWeights        map[string]float64 // 目标权重
	params               *WeightedValuationParams
	minTradeValue        float64
	daysSinceRebalance   int
	minRebalanceInterval int
	lastRebalanceTime    time.Time
	isFirstDay           bool
}

// WeightedValuationParams 权重估值策略参数
type WeightedValuationParams struct {
	// 偏离阈值
	DeviationThreshold float64 // 默认0.10 (10%)

	// PE百分位阈值
	PEHighRank float64 // 高估阈值 (默认0.70)
	PELowRank  float64 // 低估阈值 (默认0.30)

	// 恒生ETF PB阈值
	PBLow  float64 // PB低估阈值 (默认1.0)
	PBHigh float64 // PB高估阈值 (默认1.3)

	// 债券Yield阈值 (按标的)
	BondYieldThresholds map[string]YieldThreshold

	// 操作比例
	TrimRatio   float64 // 减仓比例 (默认0.3)
	AddRatio    float64 // 补仓比例 (默认0.2)
	StrongRatio float64 // 强力操作比例 (默认0.5)
}

// YieldThreshold 收益率阈值
type YieldThreshold struct {
	High float64 // 高息阈值 (便宜)
	Low  float64 // 低息阈值 (贵)
}

// DefaultWeightedValuationParams 默认参数
func DefaultWeightedValuationParams() *WeightedValuationParams {
	return &WeightedValuationParams{
		DeviationThreshold: 0.10,
		PEHighRank:         0.70,
		PELowRank:          0.30,
		PBLow:              1.0,
		PBHigh:             1.3,
		BondYieldThresholds: map[string]YieldThreshold{
			"511010": {High: 1.8, Low: 1.4}, // 5年期国债
			"511260": {High: 2.0, Low: 1.6}, // 10年期国债
			"511520": {High: 2.3, Low: 1.9}, // 7-10年政策性金融债
			"511090": {High: 2.4, Low: 2.0}, // 30年期国债
		},
		TrimRatio:   0.3,
		AddRatio:    0.2,
		StrongRatio: 0.5,
	}
}

// NewWeightedValuationStrategy 创建权重估值策略
func NewWeightedValuationStrategy(config types.StrategyConfig) *WeightedValuationStrategy {
	params := DefaultWeightedValuationParams()

	// 从配置中覆盖参数
	if config.Threshold > 0 {
		params.DeviationThreshold = config.Threshold
	}

	return &WeightedValuationStrategy{
		name:                 config.Name,
		targetWeights:        config.TargetWeights,
		params:               params,
		minTradeValue:        config.MinTradeValue,
		minRebalanceInterval: config.MinRebalanceInterval,
		daysSinceRebalance:   0,
		isFirstDay:           true,
	}
}

// Name 返回策略名称
func (s *WeightedValuationStrategy) Name() string {
	if s.name != "" {
		return s.name
	}
	return "WeightedValuation"
}

// SignalType 信号类型
type PingAnSignal string

const (
	SignalStrongSell   PingAnSignal = "🔴 坚决止盈"
	SignalSell         PingAnSignal = "🟠 减仓"
	SignalHoldNoSell   PingAnSignal = "🟡 暂不卖"
	SignalStrongBuy    PingAnSignal = "🟢 积极补仓"
	SignalBuy          PingAnSignal = "🔵 补仓"
	SignalHoldNoBuy    PingAnSignal = "🟡 暂不买"
	SignalNormal       PingAnSignal = "⚪️ 正常"
	SignalSkip         PingAnSignal = ""
)

// TargetWeights 计算动态目标权重
func (s *WeightedValuationStrategy) TargetWeights(portfolio *types.Portfolio, prices map[string]float64) map[string]float64 {
	dynamicWeights := make(map[string]float64)
	for symbol, weight := range s.targetWeights {
		dynamicWeights[symbol] = weight
	}

	// 根据信号调整权重
	currentWeights := portfolio.GetWeights()
	for symbol, pos := range portfolio.Positions {
		targetWeight := s.targetWeights[symbol]
		if targetWeight == 0 {
			continue
		}

		currentWeight := currentWeights[symbol]
		signal := s.evaluatePosition(symbol, pos, currentWeight, targetWeight)

		switch signal {
		case SignalStrongSell:
			dynamicWeights[symbol] = targetWeight * (1 - s.params.StrongRatio)
		case SignalSell:
			dynamicWeights[symbol] = targetWeight * (1 - s.params.TrimRatio)
		case SignalHoldNoSell:
			// 保持当前权重，不减仓
			dynamicWeights[symbol] = currentWeight
		case SignalStrongBuy:
			dynamicWeights[symbol] = targetWeight * (1 + s.params.StrongRatio)
		case SignalBuy:
			dynamicWeights[symbol] = targetWeight * (1 + s.params.AddRatio)
		case SignalHoldNoBuy:
			// 保持当前权重，不加仓
			dynamicWeights[symbol] = currentWeight
		default:
			// 正常情况回归目标权重
			dynamicWeights[symbol] = targetWeight
		}
	}

	return s.normalizeWeights(dynamicWeights)
}

// normalizeWeights 归一化权重
func (s *WeightedValuationStrategy) normalizeWeights(weights map[string]float64) map[string]float64 {
	total := 0.0
	for _, w := range weights {
		total += w
	}

	if total == 0 {
		return weights
	}

	normalized := make(map[string]float64)
	for symbol, w := range weights {
		normalized[symbol] = w / total
	}
	return normalized
}

// evaluatePosition 评估持仓信号
func (s *WeightedValuationStrategy) evaluatePosition(symbol string, pos types.Position, currentWeight, targetWeight float64) PingAnSignal {
	if targetWeight == 0 {
		return SignalSkip
	}

	// 计算偏离度
	deviation := (currentWeight - targetWeight) / targetWeight
	over := deviation > s.params.DeviationThreshold
	under := deviation < -s.params.DeviationThreshold

	fund := pos.Fundamental
	if fund == nil {
		// 无基本面数据，仅按偏离度操作
		if over {
			return SignalSell
		}
		if under {
			return SignalBuy
		}
		return SignalNormal
	}

	// PE百分位 (归一化到0-1)
	peRank := fund.PERank
	if peRank > 1 {
		peRank = peRank / 100
	}
	peLow := peRank > 0 && peRank <= s.params.PELowRank
	peHigh := peRank > 0 && peRank >= s.params.PEHighRank

	// 恒生ETF (159920) 特殊处理
	if symbol == "159920" {
		return s.evaluateHangSeng(pos, over, under, peRank)
	}

	// 债券ETF特殊处理
	if yieldThreshold, ok := s.params.BondYieldThresholds[symbol]; ok {
		return s.evaluateBondETF(pos, over, under, yieldThreshold)
	}

	// 黄金/其他债券 - 简单再平衡
	if fund.AssetType == types.AssetTypeBond || fund.AssetType == types.AssetTypeGold {
		if over {
			return SignalSell
		}
		if under {
			return SignalBuy
		}
		return SignalNormal
	}

	// 通用股票/ETF
	return s.evaluateGenericETF(over, under, peLow, peHigh)
}

// evaluateHangSeng 评估恒生ETF
func (s *WeightedValuationStrategy) evaluateHangSeng(pos types.Position, over, under bool, peRank float64) PingAnSignal {
	fund := pos.Fundamental
	if fund == nil {
		if over {
			return SignalSell
		}
		if under {
			return SignalBuy
		}
		return SignalNormal
	}

	// PE和PB状态
	peLow := peRank > 0 && peRank <= s.params.PELowRank
	peHigh := peRank > 0 && peRank >= s.params.PEHighRank

	// 需要从FundamentalData获取PB (这里用PE代替模拟，实际需要扩展)
	// 假设PB通过其他方式传入，这里简化处理
	pbValue := 1.0 // 默认值，实际应从数据中获取
	pbLow := pbValue < s.params.PBLow
	pbHigh := pbValue > s.params.PBHigh

	doubleLow := peLow && pbLow
	doubleHigh := peHigh && pbHigh
	anyLow := peLow || pbLow
	anyHigh := peHigh || pbHigh

	if over {
		if doubleHigh {
			return SignalStrongSell
		}
		if doubleLow {
			return SignalHoldNoSell
		}
		if anyHigh {
			return SignalSell
		}
		return SignalSell
	}

	if under {
		if doubleLow {
			return SignalStrongBuy
		}
		if doubleHigh {
			return SignalHoldNoBuy
		}
		if anyLow {
			return SignalBuy
		}
		return SignalBuy
	}

	return SignalNormal
}

// evaluateBondETF 评估债券ETF
func (s *WeightedValuationStrategy) evaluateBondETF(pos types.Position, over, under bool, threshold YieldThreshold) PingAnSignal {
	fund := pos.Fundamental
	if fund == nil {
		if over {
			return SignalSell
		}
		if under {
			return SignalBuy
		}
		return SignalNormal
	}

	// 从ROE字段借用存储Yield数据 (临时方案)
	yieldValue := fund.ROE // 需要扩展FundamentalData添加Yield字段

	yieldCheap := yieldValue > threshold.High
	yieldExpensive := yieldValue < threshold.Low && yieldValue > 0

	if over {
		if yieldExpensive {
			return SignalStrongSell
		}
		if yieldCheap {
			return SignalHoldNoSell
		}
		return SignalSell
	}

	if under {
		if yieldCheap {
			return SignalStrongBuy
		}
		if yieldExpensive {
			return SignalHoldNoBuy
		}
		return SignalBuy
	}

	return SignalNormal
}

// evaluateGenericETF 评估通用ETF
func (s *WeightedValuationStrategy) evaluateGenericETF(over, under, peLow, peHigh bool) PingAnSignal {
	if over {
		if peHigh {
			return SignalStrongSell
		}
		if peLow {
			return SignalHoldNoSell
		}
		return SignalSell
	}

	if under {
		if peLow {
			return SignalStrongBuy
		}
		if peHigh {
			return SignalHoldNoBuy
		}
		return SignalBuy
	}

	return SignalNormal
}

// ShouldRebalance 判断是否需要再平衡
func (s *WeightedValuationStrategy) ShouldRebalance(portfolio *types.Portfolio, prices map[string]float64) bool {
	if s.isFirstDay {
		return true
	}

	s.daysSinceRebalance++

	if s.minRebalanceInterval > 0 && s.daysSinceRebalance < s.minRebalanceInterval {
		return false
	}

	// 检查是否有偏离超过阈值的持仓
	currentWeights := portfolio.GetWeights()
	for symbol, targetWeight := range s.targetWeights {
		if targetWeight == 0 {
			continue
		}
		currentWeight := currentWeights[symbol]
		deviation := math.Abs((currentWeight - targetWeight) / targetWeight)
		if deviation > s.params.DeviationThreshold {
			return true
		}
	}

	return false
}

// GenerateOrders 生成交易订单
func (s *WeightedValuationStrategy) GenerateOrders(portfolio *types.Portfolio, targetWeights map[string]float64, prices map[string]float64) []types.Order {
	orders := make([]types.Order, 0)
	totalValue := portfolio.TotalValue

	if totalValue <= 0 {
		return orders
	}

	targetValues := make(map[string]float64)
	for symbol, weight := range targetWeights {
		targetValues[symbol] = totalValue * weight
	}

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
func (s *WeightedValuationStrategy) OnRebalance() {
	s.lastRebalanceTime = time.Now()
	s.daysSinceRebalance = 0
	s.isFirstDay = false
}
