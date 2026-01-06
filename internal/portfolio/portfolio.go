package portfolio

import (
	"fmt"
	"time"

	"github.com/opsxjacky/Rebalance-backtest/internal/cost"
	"github.com/opsxjacky/Rebalance-backtest/pkg/types"
)

// Manager 投资组合管理器
type Manager struct {
	portfolio *types.Portfolio
	costModel cost.CostModel
	trades    []types.Trade
}

// NewManager 创建投资组合管理器
func NewManager(initialCash float64, costModel cost.CostModel) *Manager {
	return &Manager{
		portfolio: types.NewPortfolio(initialCash),
		costModel: costModel,
		trades:    make([]types.Trade, 0),
	}
}

// GetPortfolio 获取当前投资组合
func (m *Manager) GetPortfolio() *types.Portfolio {
	return m.portfolio
}

// GetTrades 获取所有交易记录
func (m *Manager) GetTrades() []types.Trade {
	return m.trades
}

// UpdatePrices 更新持仓价值
func (m *Manager) UpdatePrices(prices map[string]float64, timestamp time.Time) {
	m.portfolio.Timestamp = timestamp
	m.portfolio.UpdateValue(prices)

	// 更新盈亏
	for symbol, pos := range m.portfolio.Positions {
		if price, ok := prices[symbol]; ok {
			pos.ProfitLoss = (price - pos.AvgCost) * pos.Quantity
			m.portfolio.Positions[symbol] = pos
		}
	}
}

// UpdateFundamentals 更新基本面数据
func (m *Manager) UpdateFundamentals(fundamentals map[string]*types.FundamentalData) {
	for symbol, pos := range m.portfolio.Positions {
		if fund, ok := fundamentals[symbol]; ok {
			pos.Fundamental = fund
			m.portfolio.Positions[symbol] = pos
		}
	}
}

// ExecuteOrder 执行订单
func (m *Manager) ExecuteOrder(order types.Order, timestamp time.Time) (types.Trade, error) {
	// 计算滑点调整后的价格
	executionPrice := m.costModel.CalculateSlippage(order.Price, order.Side)

	trade := types.Trade{
		Timestamp: timestamp,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Quantity:  order.Quantity,
		Price:     executionPrice,
		Value:     order.Quantity * executionPrice,
	}

	// 计算交易费用
	trade.Fee = m.costModel.CalculateCost(trade)

	// 执行交易
	if order.Side == "BUY" {
		err := m.executeBuy(trade)
		if err != nil {
			return types.Trade{}, err
		}
	} else {
		err := m.executeSell(trade)
		if err != nil {
			return types.Trade{}, err
		}
	}

	m.trades = append(m.trades, trade)
	return trade, nil
}

// executeBuy 执行买入
func (m *Manager) executeBuy(trade types.Trade) error {
	totalCost := trade.Value + trade.Fee

	if m.portfolio.Cash < totalCost {
		return fmt.Errorf("insufficient cash: need %.2f, have %.2f", totalCost, m.portfolio.Cash)
	}

	// 扣减现金
	m.portfolio.Cash -= totalCost

	// 更新持仓
	pos, exists := m.portfolio.Positions[trade.Symbol]
	if exists {
		// 计算新的平均成本
		totalQuantity := pos.Quantity + trade.Quantity
		totalCostBasis := pos.AvgCost*pos.Quantity + trade.Price*trade.Quantity
		pos.AvgCost = totalCostBasis / totalQuantity
		pos.Quantity = totalQuantity
	} else {
		pos = types.Position{
			Symbol:   trade.Symbol,
			Quantity: trade.Quantity,
			AvgCost:  trade.Price,
		}
	}
	pos.Value = pos.Quantity * trade.Price
	m.portfolio.Positions[trade.Symbol] = pos

	return nil
}

// executeSell 执行卖出
func (m *Manager) executeSell(trade types.Trade) error {
	pos, exists := m.portfolio.Positions[trade.Symbol]
	if !exists {
		return fmt.Errorf("no position in %s", trade.Symbol)
	}

	if pos.Quantity < trade.Quantity {
		return fmt.Errorf("insufficient shares: need %.4f, have %.4f", trade.Quantity, pos.Quantity)
	}

	// 增加现金 (扣除费用)
	m.portfolio.Cash += trade.Value - trade.Fee

	// 更新持仓
	pos.Quantity -= trade.Quantity
	if pos.Quantity < 0.0001 {
		// 清仓
		delete(m.portfolio.Positions, trade.Symbol)
	} else {
		pos.Value = pos.Quantity * trade.Price
		m.portfolio.Positions[trade.Symbol] = pos
	}

	return nil
}

// TakeSnapshot 创建快照
func (m *Manager) TakeSnapshot() types.PortfolioSnapshot {
	positions := make(map[string]types.Position)
	for k, v := range m.portfolio.Positions {
		positions[k] = v
	}

	return types.PortfolioSnapshot{
		Timestamp:  m.portfolio.Timestamp,
		Cash:       m.portfolio.Cash,
		Positions:  positions,
		TotalValue: m.portfolio.TotalValue,
		Weights:    m.portfolio.GetWeights(),
	}
}

// CanBuy 检查是否可以买入指定金额
func (m *Manager) CanBuy(symbol string, amount float64, price float64) bool {
	quantity := amount / price
	trade := types.Trade{
		Symbol:   symbol,
		Side:     "BUY",
		Quantity: quantity,
		Price:    price,
	}
	fee := m.costModel.CalculateCost(trade)
	return m.portfolio.Cash >= (amount + fee)
}
