package data

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/opsxjacky/Rebalance-backtest/pkg/types"
)

// CSVLoader CSV数据加载器
type CSVLoader struct {
	dataDir         string
	priceData       map[string][]types.PriceData
	fundamentalData map[string][]types.FundamentalData
	allDates        []time.Time
}

// NewCSVLoader 创建CSV加载器
func NewCSVLoader(dataDir string) *CSVLoader {
	return &CSVLoader{
		dataDir:         dataDir,
		priceData:       make(map[string][]types.PriceData),
		fundamentalData: make(map[string][]types.FundamentalData),
	}
}

// SourceType 返回数据源类型
func (l *CSVLoader) SourceType() string {
	return "csv"
}

// LoadPrices 加载价格数据
func (l *CSVLoader) LoadPrices(symbols []string, start, end time.Time) (map[string][]types.PriceData, error) {
	result := make(map[string][]types.PriceData)
	dateSet := make(map[time.Time]bool)

	for _, symbol := range symbols {
		priceData, fundData, err := l.loadSymbolData(symbol, start, end)
		if err != nil {
			return nil, fmt.Errorf("failed to load data for %s: %w", symbol, err)
		}
		result[symbol] = priceData
		l.priceData[symbol] = priceData
		l.fundamentalData[symbol] = fundData

		// 收集所有日期
		for _, d := range priceData {
			dateSet[d.Timestamp] = true
		}
	}

	// 整理所有日期
	l.allDates = make([]time.Time, 0, len(dateSet))
	for d := range dateSet {
		l.allDates = append(l.allDates, d)
	}
	sort.Slice(l.allDates, func(i, j int) bool {
		return l.allDates[i].Before(l.allDates[j])
	})

	return result, nil
}

// loadSymbolData 加载单个标的数据
func (l *CSVLoader) loadSymbolData(symbol string, start, end time.Time) ([]types.PriceData, []types.FundamentalData, error) {
	filePath := filepath.Join(l.dataDir, symbol+".csv")
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, nil, fmt.Errorf("CSV file has no data rows")
	}

	// 解析表头，找到各列的索引
	header := records[0]
	colIndex := parseHeader(header)

	var priceResult []types.PriceData
	var fundResult []types.FundamentalData
	for i := 1; i < len(records); i++ {
		row := records[i]
		priceData, fundData, err := parseRow(row, colIndex, symbol)
		if err != nil {
			continue // 跳过解析错误的行
		}

		// 过滤日期范围
		if !priceData.Timestamp.Before(start) && !priceData.Timestamp.After(end) {
			priceResult = append(priceResult, priceData)
			fundResult = append(fundResult, fundData)
		}
	}

	// 按日期排序
	sort.Slice(priceResult, func(i, j int) bool {
		return priceResult[i].Timestamp.Before(priceResult[j].Timestamp)
	})
	sort.Slice(fundResult, func(i, j int) bool {
		return fundResult[i].Timestamp.Before(fundResult[j].Timestamp)
	})

	return priceResult, fundResult, nil
}

// parseHeader 解析CSV表头
func parseHeader(header []string) map[string]int {
	colIndex := make(map[string]int)
	for i, col := range header {
		switch col {
		case "Date", "date", "DATE", "Timestamp", "timestamp":
			colIndex["date"] = i
		case "Open", "open", "OPEN":
			colIndex["open"] = i
		case "High", "high", "HIGH":
			colIndex["high"] = i
		case "Low", "low", "LOW":
			colIndex["low"] = i
		case "Close", "close", "CLOSE":
			colIndex["close"] = i
		case "Volume", "volume", "VOLUME":
			colIndex["volume"] = i
		case "Adj Close", "adj_close", "AdjClose", "Adj_Close":
			colIndex["adj_close"] = i
		// 基本面数据
		case "PE", "pe":
			colIndex["pe"] = i
		case "PE_Rank", "pe_rank", "PERank":
			colIndex["pe_rank"] = i
		case "PEG", "peg":
			colIndex["peg"] = i
		case "ROE", "roe":
			colIndex["roe"] = i
		case "Asset_Type", "asset_type", "AssetType":
			colIndex["asset_type"] = i
		case "Name", "name":
			colIndex["name"] = i
		case "Is_Core", "is_core", "IsCore":
			colIndex["is_core"] = i
		case "Is_Tech", "is_tech", "IsTech":
			colIndex["is_tech"] = i
		}
	}
	return colIndex
}

// parseRow 解析CSV行
func parseRow(row []string, colIndex map[string]int, symbol string) (types.PriceData, types.FundamentalData, error) {
	var priceData types.PriceData
	var fundData types.FundamentalData
	priceData.Symbol = symbol
	fundData.Symbol = symbol

	// 解析日期
	if idx, ok := colIndex["date"]; ok && idx < len(row) {
		t, err := parseDate(row[idx])
		if err != nil {
			return priceData, fundData, err
		}
		priceData.Timestamp = t
		fundData.Timestamp = t
	}

	// 解析价格数据
	if idx, ok := colIndex["open"]; ok && idx < len(row) {
		priceData.Open, _ = strconv.ParseFloat(row[idx], 64)
	}
	if idx, ok := colIndex["high"]; ok && idx < len(row) {
		priceData.High, _ = strconv.ParseFloat(row[idx], 64)
	}
	if idx, ok := colIndex["low"]; ok && idx < len(row) {
		priceData.Low, _ = strconv.ParseFloat(row[idx], 64)
	}
	if idx, ok := colIndex["close"]; ok && idx < len(row) {
		priceData.Close, _ = strconv.ParseFloat(row[idx], 64)
	}
	if idx, ok := colIndex["volume"]; ok && idx < len(row) {
		priceData.Volume, _ = strconv.ParseFloat(row[idx], 64)
	}
	if idx, ok := colIndex["adj_close"]; ok && idx < len(row) {
		priceData.AdjClose, _ = strconv.ParseFloat(row[idx], 64)
	} else {
		priceData.AdjClose = priceData.Close // 默认使用收盘价
	}

	// 解析基本面数据
	if idx, ok := colIndex["pe"]; ok && idx < len(row) {
		fundData.PE, _ = strconv.ParseFloat(row[idx], 64)
	}
	if idx, ok := colIndex["pe_rank"]; ok && idx < len(row) {
		fundData.PERank, _ = strconv.ParseFloat(row[idx], 64)
	}
	if idx, ok := colIndex["peg"]; ok && idx < len(row) {
		fundData.PEG, _ = strconv.ParseFloat(row[idx], 64)
	}
	if idx, ok := colIndex["roe"]; ok && idx < len(row) {
		fundData.ROE, _ = strconv.ParseFloat(row[idx], 64)
	}
	if idx, ok := colIndex["asset_type"]; ok && idx < len(row) {
		fundData.AssetType = types.AssetType(row[idx])
	}
	if idx, ok := colIndex["name"]; ok && idx < len(row) {
		fundData.Name = row[idx]
	}
	if idx, ok := colIndex["is_core"]; ok && idx < len(row) {
		fundData.IsCoreETF = row[idx] == "true" || row[idx] == "1" || row[idx] == "TRUE"
	}
	if idx, ok := colIndex["is_tech"]; ok && idx < len(row) {
		fundData.IsTechETF = row[idx] == "true" || row[idx] == "1" || row[idx] == "TRUE"
	}

	return priceData, fundData, nil
}

// parseDate 解析日期字符串
func parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"02-01-2006",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// GetDataRange 获取数据范围
func (l *CSVLoader) GetDataRange(symbol string) (start, end time.Time, err error) {
	data, ok := l.priceData[symbol]
	if !ok || len(data) == 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("no data for symbol %s", symbol)
	}
	return data[0].Timestamp, data[len(data)-1].Timestamp, nil
}

// GetAllDates 获取所有交易日期
func (l *CSVLoader) GetAllDates() []time.Time {
	return l.allDates
}

// GetPriceOnDate 获取指定日期的价格
func (l *CSVLoader) GetPriceOnDate(symbol string, date time.Time) (types.PriceData, bool) {
	data, ok := l.priceData[symbol]
	if !ok {
		return types.PriceData{}, false
	}

	// 二分查找
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	idx := sort.Search(len(data), func(i int) bool {
		d := data[i].Timestamp
		dOnly := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		return !dOnly.Before(dateOnly)
	})

	if idx < len(data) {
		d := data[idx].Timestamp
		dOnly := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		if dOnly.Equal(dateOnly) {
			return data[idx], true
		}
	}

	return types.PriceData{}, false
}

// GetPricesOnDate 获取指定日期所有标的的价格
func (l *CSVLoader) GetPricesOnDate(date time.Time) map[string]float64 {
	prices := make(map[string]float64)
	for symbol := range l.priceData {
		if data, ok := l.GetPriceOnDate(symbol, date); ok {
			prices[symbol] = data.AdjClose
		}
	}
	return prices
}

// GetFundamentalOnDate 获取指定日期的基本面数据
func (l *CSVLoader) GetFundamentalOnDate(symbol string, date time.Time) (types.FundamentalData, bool) {
	data, ok := l.fundamentalData[symbol]
	if !ok {
		return types.FundamentalData{}, false
	}

	// 二分查找
	dateOnly := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	idx := sort.Search(len(data), func(i int) bool {
		d := data[i].Timestamp
		dOnly := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		return !dOnly.Before(dateOnly)
	})

	if idx < len(data) {
		d := data[idx].Timestamp
		dOnly := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		if dOnly.Equal(dateOnly) {
			return data[idx], true
		}
	}

	return types.FundamentalData{}, false
}

// GetFundamentalsOnDate 获取指定日期所有标的的基本面数据
func (l *CSVLoader) GetFundamentalsOnDate(date time.Time) map[string]*types.FundamentalData {
	fundMap := make(map[string]*types.FundamentalData)
	for symbol := range l.fundamentalData {
		if data, ok := l.GetFundamentalOnDate(symbol, date); ok {
			fundMap[symbol] = &data
		}
	}
	return fundMap
}
