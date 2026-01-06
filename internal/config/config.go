package config

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/opsxjacky/Rebalance-backtest/pkg/types"
	"gopkg.in/yaml.v3"
)

// Config 配置文件结构
type Config struct {
	Backtest BacktestSection `yaml:"backtest"`
	Assets   []AssetConfig   `yaml:"assets"`
	Strategy StrategySection `yaml:"strategy"`
	Costs    CostsSection    `yaml:"costs"`
	Output   OutputSection   `yaml:"output"`
}

// BacktestSection 回测配置
type BacktestSection struct {
	StartDate      string  `yaml:"start_date"`
	EndDate        string  `yaml:"end_date"`
	InitialCapital float64 `yaml:"initial_capital"`
	Benchmark      string  `yaml:"benchmark"`
	DataDir        string  `yaml:"data_dir"`
}

// AssetConfig 资产配置
type AssetConfig struct {
	Symbol string `yaml:"symbol"`
	Name   string `yaml:"name"`
}

// StrategySection 策略配置
type StrategySection struct {
	Type   string             `yaml:"type"`
	Name   string             `yaml:"name"`
	Params StrategyParams     `yaml:"params"`
}

// StrategyParams 策略参数
type StrategyParams struct {
	TargetWeights        map[string]float64  `yaml:"target_weights"`
	Threshold            float64             `yaml:"threshold"`
	RebalanceInterval    int                 `yaml:"rebalance_interval"`
	MinTradeValue        float64             `yaml:"min_trade_value"`
	MinRebalanceInterval int                 `yaml:"min_rebalance_interval"`
	Valuation            *ValuationParamsYAML `yaml:"valuation"`
}

// ValuationParamsYAML 估值参数YAML配置
type ValuationParamsYAML struct {
	ExtremeHighPERank float64 `yaml:"extreme_high_pe_rank"`
	HighPERank        float64 `yaml:"high_pe_rank"`
	LowPERank         float64 `yaml:"low_pe_rank"`
	CoreLowPERank     float64 `yaml:"core_low_pe_rank"`
	HighPEG           float64 `yaml:"high_peg"`
	BubblePEG         float64 `yaml:"bubble_peg"`
	LowPEG            float64 `yaml:"low_peg"`
	GoodROE           float64 `yaml:"good_roe"`
	PoorROE           float64 `yaml:"poor_roe"`
	TrimRatio         float64 `yaml:"trim_ratio"`
	ReduceRatio       float64 `yaml:"reduce_ratio"`
	SellRatio         float64 `yaml:"sell_ratio"`
	BuyRatio          float64 `yaml:"buy_ratio"`
}

// CostsSection 成本配置
type CostsSection struct {
	CommissionRate float64 `yaml:"commission_rate"`
	MinCommission  float64 `yaml:"min_commission"`
	SlippageRate   float64 `yaml:"slippage_rate"`
	TaxRate        float64 `yaml:"tax_rate"`
}

// OutputSection 输出配置
type OutputSection struct {
	Format         string `yaml:"format"`
	Path           string `yaml:"path"`
	GenerateReport bool   `yaml:"generate_report"`
}

// LoadConfig 从文件加载配置
func LoadConfig(filepath string) (*Config, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// ToBacktestConfig 转换为回测配置
func (c *Config) ToBacktestConfig() (types.BacktestConfig, error) {
	startDate, err := time.Parse("2006-01-02", c.Backtest.StartDate)
	if err != nil {
		return types.BacktestConfig{}, fmt.Errorf("invalid start_date: %w", err)
	}

	endDate, err := time.Parse("2006-01-02", c.Backtest.EndDate)
	if err != nil {
		return types.BacktestConfig{}, fmt.Errorf("invalid end_date: %w", err)
	}

	symbols := make([]string, len(c.Assets))
	for i, asset := range c.Assets {
		symbols[i] = asset.Symbol
	}

	return types.BacktestConfig{
		StartDate:      startDate,
		EndDate:        endDate,
		InitialCapital: c.Backtest.InitialCapital,
		Symbols:        symbols,
		Benchmark:      c.Backtest.Benchmark,
	}, nil
}

// ToCostConfig 转换为成本配置
func (c *Config) ToCostConfig() types.CostConfig {
	return types.CostConfig{
		CommissionRate: c.Costs.CommissionRate,
		MinCommission:  c.Costs.MinCommission,
		SlippageRate:   c.Costs.SlippageRate,
		TaxRate:        c.Costs.TaxRate,
	}
}

// ToStrategyConfig 转换为策略配置
func (c *Config) ToStrategyConfig() types.StrategyConfig {
	config := types.StrategyConfig{
		Name:                 c.Strategy.Name,
		Type:                 c.Strategy.Type,
		TargetWeights:        c.Strategy.Params.TargetWeights,
		Threshold:            c.Strategy.Params.Threshold,
		RebalanceInterval:    c.Strategy.Params.RebalanceInterval,
		MinTradeValue:        c.Strategy.Params.MinTradeValue,
		MinRebalanceInterval: c.Strategy.Params.MinRebalanceInterval,
	}

	// 转换估值参数
	if c.Strategy.Params.Valuation != nil {
		v := c.Strategy.Params.Valuation
		config.ValuationParams = &types.ValuationParams{
			ExtremeHighPERank: v.ExtremeHighPERank,
			HighPERank:        v.HighPERank,
			LowPERank:         v.LowPERank,
			CoreLowPERank:     v.CoreLowPERank,
			HighPEG:           v.HighPEG,
			BubblePEG:         v.BubblePEG,
			LowPEG:            v.LowPEG,
			GoodROE:           v.GoodROE,
			PoorROE:           v.PoorROE,
			TrimRatio:         v.TrimRatio,
			ReduceRatio:       v.ReduceRatio,
			SellRatio:         v.SellRatio,
			BuyRatio:          v.BuyRatio,
		}
	}

	return config
}

// GetDataDir 获取数据目录
func (c *Config) GetDataDir() string {
	if c.Backtest.DataDir != "" {
		return c.Backtest.DataDir
	}
	return "data/sample"
}

// GetOutputPath 获取输出路径
func (c *Config) GetOutputPath() string {
	if c.Output.Path != "" {
		return c.Output.Path
	}
	return "output"
}
