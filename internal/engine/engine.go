package engine

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/opsxjacky/Rebalance-backtest/internal/cost"
	"github.com/opsxjacky/Rebalance-backtest/internal/data"
	"github.com/opsxjacky/Rebalance-backtest/internal/portfolio"
	"github.com/opsxjacky/Rebalance-backtest/internal/strategy"
	"github.com/opsxjacky/Rebalance-backtest/pkg/types"
)

// BacktestEngine 回测引擎
type BacktestEngine struct {
	config           types.BacktestConfig
	dataLoader       *data.CSVLoader
	strategy         strategy.RebalanceStrategy
	costModel        *cost.DefaultCostModel
	portfolioManager *portfolio.Manager
	snapshots        []types.PortfolioSnapshot
	result           *types.BacktestResult
}

// New 创建回测引擎
func New(config types.BacktestConfig) *BacktestEngine {
	return &BacktestEngine{
		config:    config,
		snapshots: make([]types.PortfolioSnapshot, 0),
	}
}

// SetDataLoader 设置数据加载器
func (e *BacktestEngine) SetDataLoader(loader *data.CSVLoader) {
	e.dataLoader = loader
}

// SetStrategy 设置策略
func (e *BacktestEngine) SetStrategy(s strategy.RebalanceStrategy) {
	e.strategy = s
}

// SetCostModel 设置成本模型
func (e *BacktestEngine) SetCostModel(model *cost.DefaultCostModel) {
	e.costModel = model
}

// Run 运行回测
func (e *BacktestEngine) Run() (*types.BacktestResult, error) {
	// 验证配置
	if err := e.validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 加载数据
	fmt.Printf("Loading data for symbols: %v\n", e.config.Symbols)
	_, err := e.dataLoader.LoadPrices(e.config.Symbols, e.config.StartDate, e.config.EndDate)
	if err != nil {
		return nil, fmt.Errorf("failed to load prices: %w", err)
	}

	// 初始化投资组合管理器
	e.portfolioManager = portfolio.NewManager(e.config.InitialCapital, e.costModel)

	// 获取所有交易日期
	dates := e.dataLoader.GetAllDates()
	if len(dates) == 0 {
		return nil, fmt.Errorf("no trading dates found")
	}

	fmt.Printf("Running backtest from %s to %s (%d trading days)\n",
		dates[0].Format("2006-01-02"),
		dates[len(dates)-1].Format("2006-01-02"),
		len(dates))

	// 按日期遍历
	for i, date := range dates {
		// 获取当日价格
		prices := e.dataLoader.GetPricesOnDate(date)
		if len(prices) == 0 {
			continue
		}

		// 获取当日基本面数据
		fundamentals := e.dataLoader.GetFundamentalsOnDate(date)

		// 更新投资组合价值和基本面数据
		e.portfolioManager.UpdatePrices(prices, date)
		e.portfolioManager.UpdateFundamentals(fundamentals)

		// 判断是否需要再平衡
		pf := e.portfolioManager.GetPortfolio()
		if e.strategy.ShouldRebalance(pf, prices) {
			// 计算目标权重
			targetWeights := e.strategy.TargetWeights(pf, prices)

			// 生成交易订单
			orders := e.strategy.GenerateOrders(pf, targetWeights, prices)

			// 执行订单
			for _, order := range orders {
				_, err := e.portfolioManager.ExecuteOrder(order, date)
				if err != nil {
					// 记录错误但继续执行
					fmt.Printf("Warning: failed to execute order %v: %v\n", order, err)
				}
			}

			// 更新持仓价值
			e.portfolioManager.UpdatePrices(prices, date)

			// 回调策略
			e.strategy.OnRebalance()
		}

		// 记录快照
		snapshot := e.portfolioManager.TakeSnapshot()
		e.snapshots = append(e.snapshots, snapshot)

		// 打印进度
		if (i+1)%100 == 0 || i == len(dates)-1 {
			fmt.Printf("Progress: %d/%d days, Portfolio Value: %.2f\n",
				i+1, len(dates), snapshot.TotalValue)
		}
	}

	// 生成结果
	e.result = e.generateResult()
	return e.result, nil
}

// validate 验证配置
func (e *BacktestEngine) validate() error {
	if e.dataLoader == nil {
		return fmt.Errorf("data loader not set")
	}
	if e.strategy == nil {
		return fmt.Errorf("strategy not set")
	}
	if e.costModel == nil {
		return fmt.Errorf("cost model not set")
	}
	if len(e.config.Symbols) == 0 {
		return fmt.Errorf("no symbols specified")
	}
	if e.config.InitialCapital <= 0 {
		return fmt.Errorf("initial capital must be positive")
	}
	return nil
}

// generateResult 生成回测结果
func (e *BacktestEngine) generateResult() *types.BacktestResult {
	trades := e.portfolioManager.GetTrades()
	pf := e.portfolioManager.GetPortfolio()

	// 计算总费用
	var totalFees float64
	for _, trade := range trades {
		totalFees += trade.Fee
	}

	// 计算收益率
	totalReturn := (pf.TotalValue - e.config.InitialCapital) / e.config.InitialCapital

	result := &types.BacktestResult{
		Config:      e.config,
		Trades:      trades,
		Snapshots:   e.snapshots,
		FinalValue:  pf.TotalValue,
		TotalReturn: totalReturn,
		TotalTrades: len(trades),
		TotalFees:   totalFees,
	}

	if len(e.snapshots) > 0 {
		result.StartDate = e.snapshots[0].Timestamp
		result.EndDate = e.snapshots[len(e.snapshots)-1].Timestamp
	}

	return result
}

// GetResult 获取回测结果
func (e *BacktestEngine) GetResult() *types.BacktestResult {
	return e.result
}

// ExportResults 导出结果到JSON文件
func (e *BacktestEngine) ExportResults(filepath string) error {
	if e.result == nil {
		return fmt.Errorf("no results to export, run backtest first")
	}

	// 创建输出结构
	output := struct {
		Summary   ResultSummary                `json:"summary"`
		Trades    []types.Trade                `json:"trades"`
		Snapshots []types.PortfolioSnapshot    `json:"snapshots"`
		Config    types.BacktestConfig         `json:"config"`
	}{
		Summary:   e.getSummary(),
		Trades:    e.result.Trades,
		Snapshots: e.result.Snapshots,
		Config:    e.result.Config,
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	err = ioutil.WriteFile(filepath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("Results exported to: %s\n", filepath)
	return nil
}

// ResultSummary 结果摘要
type ResultSummary struct {
	StrategyName   string    `json:"strategy_name"`
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	InitialCapital float64   `json:"initial_capital"`
	FinalValue     float64   `json:"final_value"`
	TotalReturn    float64   `json:"total_return"`
	TotalTrades    int       `json:"total_trades"`
	TotalFees      float64   `json:"total_fees"`
}

// getSummary 获取结果摘要
func (e *BacktestEngine) getSummary() ResultSummary {
	return ResultSummary{
		StrategyName:   e.strategy.Name(),
		StartDate:      e.result.StartDate,
		EndDate:        e.result.EndDate,
		InitialCapital: e.config.InitialCapital,
		FinalValue:     e.result.FinalValue,
		TotalReturn:    e.result.TotalReturn,
		TotalTrades:    e.result.TotalTrades,
		TotalFees:      e.result.TotalFees,
	}
}

// PrintSummary 打印回测摘要
func (e *BacktestEngine) PrintSummary() {
	if e.result == nil {
		fmt.Println("No results available")
		return
	}

	fmt.Println("\n========== Backtest Summary ==========")
	fmt.Printf("Strategy: %s\n", e.strategy.Name())
	fmt.Printf("Period: %s to %s\n",
		e.result.StartDate.Format("2006-01-02"),
		e.result.EndDate.Format("2006-01-02"))
	fmt.Printf("Initial Capital: $%.2f\n", e.config.InitialCapital)
	fmt.Printf("Final Value: $%.2f\n", e.result.FinalValue)
	fmt.Printf("Total Return: %.2f%%\n", e.result.TotalReturn*100)
	fmt.Printf("Total Trades: %d\n", e.result.TotalTrades)
	fmt.Printf("Total Fees: $%.2f\n", e.result.TotalFees)
	fmt.Println("========================================")
}
