"""性能指标计算模块"""

import numpy as np
import pandas as pd
from typing import Dict, Optional


class PerformanceMetrics:
    """回测结果性能指标计算"""

    def __init__(self, risk_free_rate: float = 0.02):
        """
        初始化
        Args:
            risk_free_rate: 无风险利率 (年化)
        """
        self.risk_free_rate = risk_free_rate

    def calculate_all(self, portfolio_values: pd.Series) -> Dict:
        """计算所有指标"""
        return {
            **self.calculate_returns(portfolio_values),
            **self.calculate_risk(portfolio_values),
            **self.calculate_ratios(portfolio_values),
        }

    def calculate_returns(self, portfolio_values: pd.Series) -> Dict:
        """计算收益指标"""
        if len(portfolio_values) < 2:
            return {'total_return': 0, 'annualized_return': 0, 'cagr': 0}

        total_return = (portfolio_values.iloc[-1] / portfolio_values.iloc[0]) - 1

        # 计算交易天数
        trading_days = len(portfolio_values)
        years = trading_days / 252

        # 年化收益率
        if years > 0:
            annualized_return = (1 + total_return) ** (1 / years) - 1
        else:
            annualized_return = 0

        # CAGR
        cagr = annualized_return

        return {
            'total_return': total_return,
            'annualized_return': annualized_return,
            'cagr': cagr,
        }

    def calculate_risk(self, portfolio_values: pd.Series) -> Dict:
        """计算风险指标"""
        if len(portfolio_values) < 2:
            return {
                'volatility': 0,
                'max_drawdown': 0,
                'max_drawdown_duration': 0,
                'var_95': 0,
                'cvar_95': 0,
            }

        returns = portfolio_values.pct_change().dropna()

        # 年化波动率
        volatility = returns.std() * np.sqrt(252)

        # 最大回撤
        cummax = portfolio_values.cummax()
        drawdown = (portfolio_values - cummax) / cummax
        max_drawdown = drawdown.min()

        # 最大回撤持续时间
        max_drawdown_duration = self._calculate_max_drawdown_duration(portfolio_values)

        # VaR (95%)
        var_95 = np.percentile(returns, 5)

        # CVaR (95%)
        cvar_95 = returns[returns <= var_95].mean() if len(returns[returns <= var_95]) > 0 else var_95

        return {
            'volatility': volatility,
            'max_drawdown': max_drawdown,
            'max_drawdown_duration': max_drawdown_duration,
            'var_95': var_95,
            'cvar_95': cvar_95,
        }

    def calculate_ratios(self, portfolio_values: pd.Series) -> Dict:
        """计算风险调整收益指标"""
        if len(portfolio_values) < 2:
            return {
                'sharpe_ratio': 0,
                'sortino_ratio': 0,
                'calmar_ratio': 0,
            }

        returns = portfolio_values.pct_change().dropna()
        daily_rf = self.risk_free_rate / 252
        excess_returns = returns - daily_rf

        # Sharpe Ratio
        if returns.std() > 0:
            sharpe_ratio = (excess_returns.mean() / returns.std()) * np.sqrt(252)
        else:
            sharpe_ratio = 0

        # Sortino Ratio
        downside_returns = returns[returns < 0]
        if len(downside_returns) > 0 and downside_returns.std() > 0:
            sortino_ratio = (excess_returns.mean() / downside_returns.std()) * np.sqrt(252)
        else:
            sortino_ratio = 0

        # Calmar Ratio
        cummax = portfolio_values.cummax()
        drawdown = (portfolio_values - cummax) / cummax
        max_drawdown = abs(drawdown.min())

        trading_days = len(portfolio_values)
        years = trading_days / 252
        total_return = (portfolio_values.iloc[-1] / portfolio_values.iloc[0]) - 1
        annualized_return = (1 + total_return) ** (1 / years) - 1 if years > 0 else 0

        if max_drawdown > 0:
            calmar_ratio = annualized_return / max_drawdown
        else:
            calmar_ratio = 0

        return {
            'sharpe_ratio': sharpe_ratio,
            'sortino_ratio': sortino_ratio,
            'calmar_ratio': calmar_ratio,
        }

    def _calculate_max_drawdown_duration(self, portfolio_values: pd.Series) -> int:
        """计算最大回撤持续时间（天数）"""
        cummax = portfolio_values.cummax()
        drawdown = (portfolio_values - cummax) / cummax

        # 找到回撤开始和结束的位置
        max_duration = 0
        current_duration = 0

        for dd in drawdown:
            if dd < 0:
                current_duration += 1
                max_duration = max(max_duration, current_duration)
            else:
                current_duration = 0

        return max_duration
