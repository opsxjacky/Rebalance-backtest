"""回测结果分析器"""

import json
import pandas as pd
from typing import Dict, List, Optional
from datetime import datetime
from .metrics import PerformanceMetrics


class BacktestAnalyzer:
    """回测结果分析器"""

    def __init__(self, result_path: str):
        """
        初始化分析器
        Args:
            result_path: 回测结果JSON文件路径
        """
        self.result_path = result_path
        self.data = self._load_result()
        self.metrics = PerformanceMetrics()

    def _load_result(self) -> Dict:
        """加载回测结果"""
        with open(self.result_path, 'r') as f:
            return json.load(f)

    def get_portfolio_values(self) -> pd.Series:
        """获取投资组合价值时间序列"""
        snapshots = self.data.get('snapshots', [])
        if not snapshots:
            return pd.Series()

        dates = []
        values = []
        for snap in snapshots:
            ts = snap.get('Timestamp', '')
            if ts:
                # 解析时间戳
                try:
                    dt = datetime.fromisoformat(ts.replace('Z', '+00:00'))
                    dates.append(dt)
                    values.append(snap.get('TotalValue', 0))
                except:
                    pass

        return pd.Series(values, index=pd.DatetimeIndex(dates))

    def get_trades(self) -> pd.DataFrame:
        """获取交易记录DataFrame"""
        trades = self.data.get('trades', [])
        if not trades:
            return pd.DataFrame()

        records = []
        for trade in trades:
            records.append({
                'timestamp': trade.get('Timestamp', ''),
                'symbol': trade.get('Symbol', ''),
                'side': trade.get('Side', ''),
                'quantity': trade.get('Quantity', 0),
                'price': trade.get('Price', 0),
                'fee': trade.get('Fee', 0),
                'value': trade.get('Value', 0),
            })

        df = pd.DataFrame(records)
        if not df.empty and 'timestamp' in df.columns:
            df['timestamp'] = pd.to_datetime(df['timestamp'])
        return df

    def get_summary(self) -> Dict:
        """获取回测摘要"""
        return self.data.get('summary', {})

    def calculate_all_metrics(self) -> Dict:
        """计算所有性能指标"""
        portfolio_values = self.get_portfolio_values()
        if portfolio_values.empty:
            return {}

        metrics = self.metrics.calculate_all(portfolio_values)

        # 添加交易统计
        trades_df = self.get_trades()
        if not trades_df.empty:
            metrics['total_trades'] = len(trades_df)
            metrics['total_fees'] = trades_df['fee'].sum()
            metrics['buy_trades'] = len(trades_df[trades_df['side'] == 'BUY'])
            metrics['sell_trades'] = len(trades_df[trades_df['side'] == 'SELL'])
            metrics['avg_trade_value'] = trades_df['value'].mean()

        # 添加基本信息
        summary = self.get_summary()
        metrics['strategy_name'] = summary.get('strategy_name', 'Unknown')
        metrics['initial_capital'] = summary.get('initial_capital', 0)
        metrics['final_value'] = summary.get('final_value', 0)

        return metrics

    def get_weights_history(self) -> pd.DataFrame:
        """获取权重历史"""
        snapshots = self.data.get('snapshots', [])
        if not snapshots:
            return pd.DataFrame()

        records = []
        for snap in snapshots:
            ts = snap.get('Timestamp', '')
            weights = snap.get('Weights', {})
            if ts and weights:
                record = {'timestamp': ts}
                record.update(weights)
                records.append(record)

        df = pd.DataFrame(records)
        if not df.empty and 'timestamp' in df.columns:
            df['timestamp'] = pd.to_datetime(df['timestamp'])
            df = df.set_index('timestamp')
        return df

    def print_report(self):
        """打印分析报告"""
        metrics = self.calculate_all_metrics()

        print("\n" + "=" * 60)
        print("           回测结果分析报告")
        print("=" * 60)

        print(f"\n策略名称: {metrics.get('strategy_name', 'Unknown')}")

        print("\n--- 收益指标 ---")
        print(f"  初始资金: ${metrics.get('initial_capital', 0):,.2f}")
        print(f"  最终价值: ${metrics.get('final_value', 0):,.2f}")
        print(f"  总收益率: {metrics.get('total_return', 0) * 100:.2f}%")
        print(f"  年化收益率: {metrics.get('annualized_return', 0) * 100:.2f}%")
        print(f"  CAGR: {metrics.get('cagr', 0) * 100:.2f}%")

        print("\n--- 风险指标 ---")
        print(f"  年化波动率: {metrics.get('volatility', 0) * 100:.2f}%")
        print(f"  最大回撤: {metrics.get('max_drawdown', 0) * 100:.2f}%")
        print(f"  最大回撤持续: {metrics.get('max_drawdown_duration', 0)} 天")
        print(f"  VaR (95%): {metrics.get('var_95', 0) * 100:.2f}%")
        print(f"  CVaR (95%): {metrics.get('cvar_95', 0) * 100:.2f}%")

        print("\n--- 风险调整收益 ---")
        print(f"  Sharpe Ratio: {metrics.get('sharpe_ratio', 0):.3f}")
        print(f"  Sortino Ratio: {metrics.get('sortino_ratio', 0):.3f}")
        print(f"  Calmar Ratio: {metrics.get('calmar_ratio', 0):.3f}")

        print("\n--- 交易统计 ---")
        print(f"  总交易次数: {metrics.get('total_trades', 0)}")
        print(f"  买入交易: {metrics.get('buy_trades', 0)}")
        print(f"  卖出交易: {metrics.get('sell_trades', 0)}")
        print(f"  总交易费用: ${metrics.get('total_fees', 0):,.2f}")
        print(f"  平均交易金额: ${metrics.get('avg_trade_value', 0):,.2f}")

        print("\n" + "=" * 60)


if __name__ == '__main__':
    import sys
    if len(sys.argv) > 1:
        analyzer = BacktestAnalyzer(sys.argv[1])
        analyzer.print_report()
    else:
        print("Usage: python analyzer.py <result.json>")
