package data

import (
	"time"

	"github.com/opsxjacky/Rebalance-backtest/pkg/types"
)

// DataLoader 数据加载器接口
type DataLoader interface {
	// LoadPrices 加载历史价格数据
	LoadPrices(symbols []string, start, end time.Time) (map[string][]types.PriceData, error)

	// GetDataRange 获取可用数据范围
	GetDataRange(symbol string) (start, end time.Time, err error)

	// SourceType 支持的数据源类型
	SourceType() string

	// GetAllDates 获取所有交易日期
	GetAllDates() []time.Time
}
