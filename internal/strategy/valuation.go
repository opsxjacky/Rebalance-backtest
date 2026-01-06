package strategy

import (
	"math"
	"time"

	"github.com/opsxjacky/Rebalance-backtest/pkg/types"
)

// ValuationStrategy 估值驱动再平衡策略
// 基于PE百分位、PEG、ROE等基本面指标动态调整持仓
type ValuationStrategy struct {
	name               string
	baseWeights        map[string]float64 // 基础目标权重
	params             *types.ValuationParams
	minTradeValue      float64
	daysSinceRebalance int
	minRebalanceInterval int
	lastRebalanceTime  time.Time
	isFirstDay         bool
}

// NewValuationStrategy 创建估值驱动策略
func NewValuationStrategy(config types.StrategyConfig) *ValuationStrategy {
	params := config.ValuationParams
	if params == nil {
		params = types.DefaultValuationParams()
	}

	return &ValuationStrategy{
		name:               config.Name,
		baseWeights:        config.TargetWeights,
		params:             params,
		minTradeValue:      config.MinTradeValue,
		minRebalanceInterval: config.MinRebalanceInterval,
		daysSinceRebalance: 0,
		isFirstDay:         true,
	}
}

// Name 返回策略名称
func (s *ValuationStrategy) Name() string {
	if s.name != "" {
		return s.name
	}
	return "ValuationDriven"
}

// TargetWeights 根据估值计算动态目标权重
func (s *ValuationStrategy) TargetWeights(portfolio *types.Portfolio, prices map[string]float64) map[string]float64 {
	// 首先复制基础权重
	dynamicWeights := make(map[string]float64)
	for symbol, weight := range s.baseWeights {
		dynamicWeights[symbol] = weight
	}

	// 根据每个持仓的估值信号调整权重
	for symbol, pos := range portfolio.Positions {
		if pos.Fundamental == nil {
			continue
		}

		signal := s.evaluateAsset(pos)
		baseWeight := s.baseWeights[symbol]

		switch signal {
		case types.SignalStrongSell, types.SignalSell:
			// 极高风险/卖出：大幅减少权重
			dynamicWeights[symbol] = baseWeight * (1 - s.params.SellRatio)
		case types.SignalTrim:
			// 动态再平衡：适度减少权重
			dynamicWeights[symbol] = baseWeight * (1 - s.params.TrimRatio)
		case types.SignalReduce:
			// 减仓：减少权重
			dynamicWeights[symbol] = baseWeight * (1 - s.params.ReduceRatio)
		case types.SignalBuy:
			// 买入：增加权重
			dynamicWeights[symbol] = baseWeight * (1 + s.params.BuyRatio)
		case types.SignalStrongHold:
			// 优质持有：略微增加权重
			dynamicWeights[symbol] = baseWeight * 1.1
		default:
			// 其他情况保持基础权重
		}
	}

	// 归一化权重
	return s.normalizeWeights(dynamicWeights)
}

// normalizeWeights 归一化权重使总和为1
func (s *ValuationStrategy) normalizeWeights(weights map[string]float64) map[string]float64 {
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

// evaluateAsset 评估单个资产并返回交易信号
func (s *ValuationStrategy) evaluateAsset(pos types.Position) types.SignalType {
	fund := pos.Fundamental
	if fund == nil {
		return types.SignalUnknown
	}

	// 计算盈亏
	plVal := pos.ProfitLoss

	// 垃圾股检测：亏损且基本面差
	isTrash := plVal < 0 && (fund.PE == 0 || fund.ROE < s.params.PoorROE)
	if isTrash {
		return types.SignalStrongSell
	}

	// 安全资产
	isSafe := fund.AssetType == types.AssetTypeBond ||
		fund.AssetType == types.AssetTypeGold ||
		fund.AssetType == types.AssetTypeCash
	if isSafe {
		return types.SignalAllocate
	}

	// 估值区间判断
	peRank := fund.PERank
	isExtremeHigh := peRank > 0 && peRank >= s.params.ExtremeHighPERank
	isHigh := peRank > 0 && peRank >= s.params.HighPERank
	isLow := peRank > 0 && peRank <= s.params.LowPERank
	isCoreLow := (fund.IsCoreETF || fund.IsTechETF) && peRank > 0 && peRank <= s.params.CoreLowPERank

	// ETF 评估
	if fund.AssetType == types.AssetTypeETF {
		if isExtremeHigh && fund.IsCoreETF {
			return types.SignalTrim // 核心ETF极高估：动态再平衡
		}
		if isExtremeHigh && fund.IsTechETF {
			return types.SignalHold // 科技ETF极高估：趋势持有
		}
		if isExtremeHigh {
			return types.SignalSell // 其他ETF极高估：卖出
		}
		if isLow || isCoreLow {
			return types.SignalBuy // 低估：买入
		}
		if isHigh {
			return types.SignalWatch // 偏高：观察
		}
		return types.SignalHold
	}

	// 个股评估
	if fund.AssetType == types.AssetTypeStock {
		peg := fund.PEG
		roe := fund.ROE

		// 泡沫破裂：PE>=80且PEG>2.5
		if peRank >= 80 && peg > s.params.BubblePEG {
			return types.SignalStrongSell
		}
		// 估值透支：PEG>2.0
		if peg > s.params.HighPEG {
			return types.SignalReduce
		}
		// 优质持有：PEG<1.5或ROE>=20
		if (peg > 0 && peg < s.params.LowPEG) || roe >= s.params.GoodROE {
			return types.SignalStrongHold
		}
		// 估值过高：PE>=80
		if peRank >= 80 {
			return types.SignalReduce
		}
		return types.SignalHold
	}

	return types.SignalUnknown
}

// ShouldRebalance 判断是否需要再平衡
func (s *ValuationStrategy) ShouldRebalance(portfolio *types.Portfolio, prices map[string]float64) bool {
	// 第一天需要建仓
	if s.isFirstDay {
		return true
	}

	s.daysSinceRebalance++

	// 检查最小再平衡间隔
	if s.minRebalanceInterval > 0 && s.daysSinceRebalance < s.minRebalanceInterval {
		return false
	}

	// 检查是否有任何资产需要操作
	for _, pos := range portfolio.Positions {
		signal := s.evaluateAsset(pos)
		switch signal {
		case types.SignalStrongSell, types.SignalSell, types.SignalReduce, types.SignalTrim, types.SignalBuy:
			return true
		}
	}

	return false
}

// GenerateOrders 生成交易订单
func (s *ValuationStrategy) GenerateOrders(portfolio *types.Portfolio, targetWeights map[string]float64, prices map[string]float64) []types.Order {
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
func (s *ValuationStrategy) OnRebalance() {
	s.lastRebalanceTime = time.Now()
	s.daysSinceRebalance = 0
	s.isFirstDay = false
}

// GetSignals 获取所有持仓的信号 (用于报告)
func (s *ValuationStrategy) GetSignals(portfolio *types.Portfolio) map[string]types.SignalType {
	signals := make(map[string]types.SignalType)
	for symbol, pos := range portfolio.Positions {
		signals[symbol] = s.evaluateAsset(pos)
	}
	return signals
}
