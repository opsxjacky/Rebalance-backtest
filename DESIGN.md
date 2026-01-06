# Rebalance-Backtest 设计文档

## 1. 项目概述

### 1.1 项目背景
Rebalance-Backtest 是一个投资组合再平衡策略回测系统，用于评估和验证不同再平衡策略在历史数据上的表现。

### 1.2 核心目标
- 支持多种再平衡策略的定义和回测
- 提供高性能的回测引擎，支持大规模历史数据处理
- 生成详细的策略表现分析报告和可视化图表

### 1.3 技术栈选择
| 组件 | 技术 | 用途 |
|------|------|------|
| 回测引擎 | Go | 高性能核心计算、并发处理 |
| 数据分析 | Python | 数据处理、统计分析、可视化 |
| 数据存储 | SQLite/PostgreSQL | 持久化存储历史数据和回测结果 |
| 配置管理 | YAML/JSON | 策略配置和系统参数 |

---

## 2. 系统架构

### 2.1 整体架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户接口层                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │   CLI 工具    │  │  配置文件     │  │  Python API 接口     │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Go 核心引擎层                                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │  数据加载器   │  │  策略执行器   │  │     回测引擎         │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │  交易模拟器   │  │  成本计算器   │  │     结果收集器       │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Python 分析层                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │  性能指标计算  │  │  风险分析     │  │     可视化报告       │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       数据层                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │  市场数据     │  │  回测结果     │  │     策略配置         │   │
│  └──────────────┘  └──────────────┘  └──────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Go 与 Python 交互方式

```
┌────────────────┐         JSON/CSV           ┌────────────────┐
│                │  ──────────────────────▶   │                │
│   Go 回测引擎   │                            │  Python 分析   │
│                │  ◀──────────────────────   │                │
└────────────────┘       配置参数              └────────────────┘
         │                                            │
         │              gRPC (可选)                    │
         └────────────────────────────────────────────┘
```

---

## 3. 核心模块设计

### 3.1 数据模块 (Go)

#### 3.1.1 数据结构定义

```go
// 资产价格数据
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

// 投资组合持仓
type Position struct {
    Symbol   string
    Quantity float64
    AvgCost  float64
    Value    float64
}

// 投资组合快照
type Portfolio struct {
    Timestamp   time.Time
    Cash        float64
    Positions   map[string]Position
    TotalValue  float64
}

// 交易记录
type Trade struct {
    Timestamp time.Time
    Symbol    string
    Side      string  // "BUY" or "SELL"
    Quantity  float64
    Price     float64
    Fee       float64
}
```

#### 3.1.2 数据加载器接口

```go
type DataLoader interface {
    // 加载历史价格数据
    LoadPrices(symbols []string, start, end time.Time) (map[string][]PriceData, error)

    // 获取可用数据范围
    GetDataRange(symbol string) (start, end time.Time, error)

    // 支持的数据源类型
    SourceType() string
}
```

### 3.2 策略模块 (Go)

#### 3.2.1 再平衡策略接口

```go
type RebalanceStrategy interface {
    // 策略名称
    Name() string

    // 计算目标权重
    TargetWeights(portfolio Portfolio, prices map[string]PriceData) map[string]float64

    // 判断是否需要再平衡
    ShouldRebalance(portfolio Portfolio, prices map[string]PriceData) bool

    // 生成交易订单
    GenerateOrders(current, target map[string]float64, portfolio Portfolio, prices map[string]PriceData) []Order
}
```

#### 3.2.2 内置策略类型

| 策略类型 | 描述 |
|---------|------|
| `FixedWeight` | 固定权重再平衡，维持预设的资产配比 |
| `ThresholdBased` | 阈值触发再平衡，偏离超过阈值时触发 |
| `TimeBased` | 定期再平衡，按固定时间间隔执行 |
| `Valuation` | 估值驱动再平衡，基于PE百分位/PEG/ROE等指标 |
| `WeightedValuation` | 权重偏离+估值信号驱动，结合偏离阈值和估值判断 |

#### 3.2.3 策略配置示例

```yaml
strategy:
  name: "threshold_rebalance"
  type: "ThresholdBased"
  params:
    target_weights:
      SPY: 0.60    # 美股60%
      TLT: 0.30    # 债券30%
      GLD: 0.10    # 黄金10%
    threshold: 0.05  # 偏离5%触发再平衡
    min_trade_value: 100  # 最小交易金额
```

### 3.3 回测引擎 (Go)

#### 3.3.1 引擎核心结构

```go
type BacktestEngine struct {
    config      BacktestConfig
    dataLoader  DataLoader
    strategy    RebalanceStrategy
    costModel   CostModel
    portfolio   Portfolio
    trades      []Trade
    snapshots   []PortfolioSnapshot
}

type BacktestConfig struct {
    StartDate       time.Time
    EndDate         time.Time
    InitialCapital  float64
    Symbols         []string
    RebalanceFreq   string  // "daily", "weekly", "monthly"
    Benchmark       string
}
```

#### 3.3.2 回测流程

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  初始化组合   │────▶│  加载数据    │────▶│  按日期遍历  │
└─────────────┘     └─────────────┘     └─────────────┘
                                               │
       ┌───────────────────────────────────────┘
       ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ 更新市场价格 │────▶│ 检查再平衡   │────▶│ 执行交易    │
└─────────────┘     └─────────────┘     └─────────────┘
       │                                       │
       │         ┌─────────────┐               │
       └────────▶│ 记录快照    │◀──────────────┘
                 └─────────────┘
                        │
                        ▼
                 ┌─────────────┐
                 │ 输出结果    │
                 └─────────────┘
```

### 3.4 成本模型 (Go)

```go
type CostModel interface {
    // 计算交易成本
    CalculateCost(trade Trade) float64
}

type DefaultCostModel struct {
    CommissionRate float64  // 佣金率
    MinCommission  float64  // 最低佣金
    SlippageRate   float64  // 滑点率
    TaxRate        float64  // 税率
}
```

### 3.5 分析模块 (Python)

#### 3.5.1 性能指标

```python
class PerformanceMetrics:
    """回测结果性能指标计算"""

    def calculate_returns(self, portfolio_values: pd.Series) -> dict:
        """计算收益指标"""
        return {
            'total_return': self._total_return(portfolio_values),
            'annualized_return': self._annualized_return(portfolio_values),
            'cagr': self._cagr(portfolio_values),
        }

    def calculate_risk(self, portfolio_values: pd.Series) -> dict:
        """计算风险指标"""
        returns = portfolio_values.pct_change().dropna()
        return {
            'volatility': returns.std() * np.sqrt(252),
            'max_drawdown': self._max_drawdown(portfolio_values),
            'var_95': np.percentile(returns, 5),
            'cvar_95': returns[returns <= np.percentile(returns, 5)].mean(),
        }

    def calculate_ratios(self, portfolio_values: pd.Series,
                         risk_free_rate: float = 0.02) -> dict:
        """计算风险调整收益指标"""
        returns = portfolio_values.pct_change().dropna()
        excess_returns = returns - risk_free_rate / 252
        return {
            'sharpe_ratio': excess_returns.mean() / returns.std() * np.sqrt(252),
            'sortino_ratio': self._sortino_ratio(returns, risk_free_rate),
            'calmar_ratio': self._calmar_ratio(portfolio_values),
        }
```

#### 3.5.2 可视化报告

```python
class ReportGenerator:
    """生成可视化报告"""

    def generate_report(self, backtest_result: dict, output_path: str):
        """生成完整的HTML报告"""

    def plot_equity_curve(self, portfolio_values: pd.Series):
        """绘制权益曲线"""

    def plot_drawdown(self, portfolio_values: pd.Series):
        """绘制回撤图"""

    def plot_weights_over_time(self, weights_history: pd.DataFrame):
        """绘制权重变化图"""

    def plot_trade_analysis(self, trades: pd.DataFrame):
        """绘制交易分析图"""
```

---

## 4. 目录结构

```
Rebalance-backtest/
├── cmd/                          # Go 命令行入口
│   └── backtest/
│       └── main.go
├── internal/                     # Go 内部包
│   ├── config/                   # 配置加载
│   │   └── config.go
│   ├── engine/                   # 回测引擎
│   │   └── engine.go
│   ├── strategy/                 # 策略实现
│   │   ├── interface.go          # 策略接口
│   │   ├── fixed_weight.go       # 固定权重策略
│   │   ├── time_based.go         # 定期再平衡策略
│   │   ├── valuation.go          # 估值驱动策略 ✨
│   │   └── weighted_valuation.go # 权重+估值策略 ✨
│   ├── data/                     # 数据加载
│   │   ├── loader.go             # 加载器接口
│   │   └── csv_loader.go         # CSV加载器
│   ├── cost/                     # 成本模型
│   │   └── cost_model.go
│   └── portfolio/                # 投资组合管理
│       └── portfolio.go
├── pkg/                          # Go 公共包
│   └── types/                    # 公共类型定义
│       └── types.go
├── python/                       # Python 分析模块
│   ├── analysis/
│   │   ├── __init__.py
│   │   ├── analyzer.py
│   │   └── metrics.py
│   └── visualization/
│       ├── __init__.py
│       ├── charts.py
│       └── report.py
├── configs/                      # 配置文件
│   ├── default.yaml              # 默认配置
│   ├── xueying_config.yaml       # 雪盈账户配置 ✨
│   └── pingan_config.yaml        # 平安证券配置 ✨
├── data/                         # 数据目录
│   ├── sample/                   # 示例数据 (SPY/QQQ/TLT/GLD)
│   ├── xueying/                  # 雪盈账户数据 ✨ (9个美股ETF)
│   └── pingan/                   # 平安证券数据 ✨ (12个A股ETF)
├── output/                       # 输出目录
│   ├── xueying/                  # 雪盈回测结果
│   └── pingan/                   # 平安回测结果
├── scripts/                      # 辅助脚本
│   ├── analyze.py                # 分析脚本
│   ├── download_xueying_data.py  # 雪盈数据下载 ✨
│   └── download_pingan_data.py   # 平安数据下载 ✨
├── go.mod
├── go.sum
├── requirements.txt              # Python 依赖
├── .gitignore                    # Git忽略规则
└── DESIGN.md                     # 本文档
```

---

## 5. 接口设计

### 5.1 CLI 接口

```bash
# 运行回测
./backtest run --config configs/default.yaml --strategy threshold

# 查看帮助
./backtest --help

# 输出参数
./backtest run --config configs/default.yaml \
  --start 2020-01-01 \
  --end 2023-12-31 \
  --capital 100000 \
  --output output/result.json
```

### 5.2 Go 编程接口

```go
// 创建回测引擎
engine := engine.New(config)

// 设置策略
engine.SetStrategy(strategy.NewThresholdBased(params))

// 运行回测
result, err := engine.Run()

// 导出结果
engine.ExportResults("output/result.json")
```

### 5.3 Python 分析接口

```python
from analysis import BacktestAnalyzer
from visualization import ReportGenerator

# 加载回测结果
analyzer = BacktestAnalyzer("output/result.json")

# 计算指标
metrics = analyzer.calculate_all_metrics()

# 生成报告
generator = ReportGenerator()
generator.generate_html_report(metrics, "output/report.html")
```

---

## 6. 性能指标清单

### 6.1 收益指标
- 总收益率 (Total Return)
- 年化收益率 (Annualized Return)
- 复合年增长率 (CAGR)

### 6.2 风险指标
- 波动率 (Volatility)
- 最大回撤 (Maximum Drawdown)
- VaR (Value at Risk)
- CVaR (Conditional VaR)

### 6.3 风险调整收益
- 夏普比率 (Sharpe Ratio)
- 索提诺比率 (Sortino Ratio)
- 卡玛比率 (Calmar Ratio)
- 信息比率 (Information Ratio)

### 6.4 交易统计
- 交易次数
- 换手率
- 交易成本总额
- 平均持仓时间

---

## 7. 开发路线

### Phase 1: 基础框架 ✅
- [x] 项目初始化和目录结构
- [x] 核心数据结构定义
- [x] CSV 数据加载器
- [x] 基础投资组合管理

### Phase 2: 回测引擎 ✅
- [x] 回测引擎核心实现
- [x] 固定权重再平衡策略
- [x] 阈值触发再平衡策略
- [x] 交易成本模型

### Phase 3: 估值策略 ✅
- [x] 估值驱动策略 (Valuation)
- [x] 权重偏离+估值策略 (WeightedValuation)
- [x] Yahoo Finance 数据下载脚本
- [x] A股 ETF 数据支持

### Phase 4: 扩展功能
- [x] 多策略配置支持
- [ ] Python 可视化图表
- [ ] HTML 报告生成
- [ ] 参数优化功能

---

## 8. 附录

### 8.1 配置文件完整示例

```yaml
# configs/default.yaml
backtest:
  start_date: "2020-01-01"
  end_date: "2023-12-31"
  initial_capital: 100000
  benchmark: "SPY"

assets:
  - symbol: "SPY"
    name: "S&P 500 ETF"
  - symbol: "TLT"
    name: "20+ Year Treasury Bond ETF"
  - symbol: "GLD"
    name: "Gold ETF"

strategy:
  type: "threshold"
  params:
    target_weights:
      SPY: 0.60
      TLT: 0.30
      GLD: 0.10
    threshold: 0.05
    min_rebalance_interval: 7  # days

costs:
  commission_rate: 0.001
  min_commission: 1.0
  slippage_rate: 0.0005

output:
  format: "json"
  path: "output/"
  generate_report: true
```

### 8.2 依赖版本

**Go (go.mod):**
```
go 1.21

require (
    gopkg.in/yaml.v3 v3.0.1
    github.com/spf13/cobra v1.8.0
)
```

**Python (requirements.txt):**
```
pandas>=2.0.0
numpy>=1.24.0
matplotlib>=3.7.0
plotly>=5.18.0
jinja2>=3.1.0
```

### 8.3 已实现策略

#### 估值驱动策略 (Valuation)
基于 PE 百分位、PEG、ROE 等基本面指标动态调整持仓权重。

**信号逻辑：**
- PE百分位 ≥ 90：极度高估，触发卖出
- PE百分位 ≥ 75：高估，观察
- PE百分位 ≤ 20：低估，买入机会
- 核心ETF (SPY/QQQ/DXJ)：极高估时动态再平衡，不完全卖出
- 科技ETF (SMH/QTUM)：趋势跟随，高估时持有

#### 权重偏离+估值策略 (WeightedValuation)
结合权重偏离阈值和估值信号的复合策略。

**核心参数：**
- 偏离阈值：10%（超过触发再平衡）
- PE百分位：高估 >70%，低估 <30%
- 恒生ETF：PE+PB双因子判断
- 债券ETF：Yield阈值判断

---

## 9. 回测结果

### 9.1 雪盈账户 (估值策略)

| 指标 | 数值 |
|------|------|
| 策略类型 | Valuation |
| 回测期间 | 2022-01-03 至 2025-12-31 |
| 初始资金 | $100,000 |
| 最终价值 | $206,263 |
| **总收益率** | **106.26%** |
| 总交易次数 | 767 笔 |
| 手续费 | $791.24 |

### 9.2 平安证券账户 (权重+估值策略)

| 指标 | 数值 |
|------|------|
| 策略类型 | WeightedValuation |
| 回测期间 | 2022-01-03 至 2025-12-31 |
| 初始资金 | ¥1,000,000 |
| 最终价值 | ¥1,362,818 |
| **总收益率** | **36.28%** |
| 总交易次数 | 1615 笔 |
| 手续费 | ¥15,102.46 |

---

## 10. 运行说明

```bash
# 编译项目
go build -o backtest ./cmd/backtest

# 运行雪盈账户回测
./backtest run --config configs/xueying_config.yaml

# 运行平安证券账户回测
./backtest run --config configs/pingan_config.yaml

# 下载历史数据
python scripts/download_xueying_data.py   # 雪盈
python scripts/download_pingan_data.py    # 平安
```