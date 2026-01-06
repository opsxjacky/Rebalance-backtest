package types

import (
	"time"
)

// PriceData 资产价格数据
type PriceData struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	AdjClose  float64
}

// AssetType 资产类型
type AssetType string

const (
	AssetTypeETF    AssetType = "ETF"
	AssetTypeStock  AssetType = "个股"
	AssetTypeBond   AssetType = "债券"
	AssetTypeGold   AssetType = "黄金"
	AssetTypeCash   AssetType = "现金"
	AssetTypeOther  AssetType = "其他"
)

// FundamentalData 基本面数据
type FundamentalData struct {
	Symbol     string
	Timestamp  time.Time
	PE         float64 // 市盈率
	PERank     float64 // PE百分位 (0-100)
	PEG        float64 // PEG值
	ROE        float64 // 净资产收益率 (%)
	AssetType  AssetType
	Name       string
	IsCoreETF  bool // 是否核心指数ETF (SPY/QQQ/DXJ等)
	IsTechETF  bool // 是否科技类ETF
}

// AssetData 综合资产数据 (价格+基本面)
type AssetData struct {
	Price       PriceData
	Fundamental FundamentalData
}

// SignalType 交易信号类型
type SignalType string

const (
	SignalStrongSell  SignalType = "🔴 极高风险"
	SignalSell        SignalType = "🔴 卖出"
	SignalTrim        SignalType = "🟠 动态再平衡"
	SignalReduce      SignalType = "🟠 减仓"
	SignalWatch       SignalType = "🟡 观察"
	SignalHold        SignalType = "⚪️ 正常持有"
	SignalAllocate    SignalType = "⚪️ 按权重配置"
	SignalBuy         SignalType = "🟢 买入"
	SignalStrongHold  SignalType = "🟢 优质持有"
	SignalUnknown     SignalType = "❓ 未知"
)

// Position 投资组合持仓
type Position struct {
	Symbol      string
	Quantity    float64
	AvgCost     float64
	Value       float64
	ProfitLoss  float64   // 浮动盈亏
	Fundamental *FundamentalData
}

// Portfolio 投资组合快照
type Portfolio struct {
	Timestamp  time.Time
	Cash       float64
	Positions  map[string]Position
	TotalValue float64
}

// NewPortfolio 创建新的投资组合
func NewPortfolio(initialCash float64) *Portfolio {
	return &Portfolio{
		Timestamp:  time.Now(),
		Cash:       initialCash,
		Positions:  make(map[string]Position),
		TotalValue: initialCash,
	}
}

// UpdateValue 更新投资组合价值
func (p *Portfolio) UpdateValue(prices map[string]float64) {
	totalPositionValue := 0.0
	for symbol, pos := range p.Positions {
		if price, ok := prices[symbol]; ok {
			pos.Value = pos.Quantity * price
			p.Positions[symbol] = pos
			totalPositionValue += pos.Value
		}
	}
	p.TotalValue = p.Cash + totalPositionValue
}

// GetWeights 获取当前权重
func (p *Portfolio) GetWeights() map[string]float64 {
	weights := make(map[string]float64)
	if p.TotalValue == 0 {
		return weights
	}
	for symbol, pos := range p.Positions {
		weights[symbol] = pos.Value / p.TotalValue
	}
	weights["CASH"] = p.Cash / p.TotalValue
	return weights
}

// Trade 交易记录
type Trade struct {
	Timestamp time.Time
	Symbol    string
	Side      string // "BUY" or "SELL"
	Quantity  float64
	Price     float64
	Fee       float64
	Value     float64 // 交易金额 (不含手续费)
}

// Order 交易订单
type Order struct {
	Symbol   string
	Side     string // "BUY" or "SELL"
	Quantity float64
	Price    float64
}

// PortfolioSnapshot 投资组合快照 (用于记录历史)
type PortfolioSnapshot struct {
	Timestamp  time.Time
	Cash       float64
	Positions  map[string]Position
	TotalValue float64
	Weights    map[string]float64
}

// BacktestConfig 回测配置
type BacktestConfig struct {
	StartDate      time.Time
	EndDate        time.Time
	InitialCapital float64
	Symbols        []string
	Benchmark      string
}

// BacktestResult 回测结果
type BacktestResult struct {
	Config        BacktestConfig
	Trades        []Trade
	Snapshots     []PortfolioSnapshot
	FinalValue    float64
	TotalReturn   float64
	TotalTrades   int
	TotalFees     float64
	StartDate     time.Time
	EndDate       time.Time
}

// CostConfig 成本配置
type CostConfig struct {
	CommissionRate float64 // 佣金率
	MinCommission  float64 // 最低佣金
	SlippageRate   float64 // 滑点率
	TaxRate        float64 // 税率
}

// StrategyConfig 策略配置
type StrategyConfig struct {
	Name                 string
	Type                 string
	TargetWeights        map[string]float64
	Threshold            float64 // 阈值触发再平衡的偏离阈值
	RebalanceInterval    int     // 定期再平衡的间隔天数
	MinTradeValue        float64 // 最小交易金额
	MinRebalanceInterval int     // 最小再平衡间隔天数

	// 估值策略参数
	ValuationParams *ValuationParams
}

// ValuationParams 估值策略参数
type ValuationParams struct {
	// PE百分位阈值
	ExtremeHighPERank float64 // 极度高估阈值 (默认90)
	HighPERank        float64 // 高估阈值 (默认75)
	LowPERank         float64 // 低估阈值 (默认20)
	CoreLowPERank     float64 // 核心资产低估阈值 (默认50)

	// PEG阈值
	HighPEG           float64 // PEG高估阈值 (默认2.0)
	BubblePEG         float64 // PEG泡沫阈值 (默认2.5)
	LowPEG            float64 // PEG低估阈值 (默认1.5)

	// ROE阈值
	GoodROE           float64 // 优质ROE阈值 (默认20)
	PoorROE           float64 // 差ROE阈值 (默认5)

	// 操作比例
	TrimRatio         float64 // 动态再平衡减仓比例 (默认0.2)
	ReduceRatio       float64 // 减仓比例 (默认0.3)
	SellRatio         float64 // 卖出比例 (默认0.5)
	BuyRatio          float64 // 买入增仓比例 (默认0.2)
}

// DefaultValuationParams 默认估值参数
func DefaultValuationParams() *ValuationParams {
	return &ValuationParams{
		ExtremeHighPERank: 90,
		HighPERank:        75,
		LowPERank:         20,
		CoreLowPERank:     50,
		HighPEG:           2.0,
		BubblePEG:         2.5,
		LowPEG:            1.5,
		GoodROE:           20,
		PoorROE:           5,
		TrimRatio:         0.2,
		ReduceRatio:       0.3,
		SellRatio:         0.5,
		BuyRatio:          0.2,
	}
}
