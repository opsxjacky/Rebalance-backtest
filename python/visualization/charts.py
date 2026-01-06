"""图表生成模块"""

import pandas as pd
import numpy as np
from typing import Optional

try:
    import matplotlib.pyplot as plt
    import matplotlib.dates as mdates
    HAS_MATPLOTLIB = True
except ImportError:
    HAS_MATPLOTLIB = False


class ChartGenerator:
    """图表生成器"""

    def __init__(self, style: str = 'seaborn-v0_8-whitegrid'):
        """初始化图表生成器"""
        if HAS_MATPLOTLIB:
            try:
                plt.style.use(style)
            except:
                plt.style.use('seaborn-whitegrid')

    def plot_equity_curve(self, portfolio_values: pd.Series,
                          benchmark: Optional[pd.Series] = None,
                          title: str = "权益曲线",
                          save_path: Optional[str] = None):
        """
        绘制权益曲线
        Args:
            portfolio_values: 投资组合价值序列
            benchmark: 基准收益序列 (可选)
            title: 图表标题
            save_path: 保存路径 (可选)
        """
        if not HAS_MATPLOTLIB:
            print("需要安装 matplotlib 才能绘制图表")
            return

        fig, ax = plt.subplots(figsize=(12, 6))

        # 归一化到初始值
        normalized = portfolio_values / portfolio_values.iloc[0] * 100
        ax.plot(normalized.index, normalized.values, label='策略', linewidth=2)

        if benchmark is not None:
            benchmark_normalized = benchmark / benchmark.iloc[0] * 100
            ax.plot(benchmark_normalized.index, benchmark_normalized.values,
                   label='基准', linewidth=2, alpha=0.7)

        ax.set_title(title, fontsize=14, fontweight='bold')
        ax.set_xlabel('日期')
        ax.set_ylabel('净值 (初始=100)')
        ax.legend()
        ax.grid(True, alpha=0.3)

        # 格式化日期
        ax.xaxis.set_major_formatter(mdates.DateFormatter('%Y-%m'))
        ax.xaxis.set_major_locator(mdates.MonthLocator(interval=3))
        plt.xticks(rotation=45)

        plt.tight_layout()

        if save_path:
            plt.savefig(save_path, dpi=150, bbox_inches='tight')
        plt.show()

    def plot_drawdown(self, portfolio_values: pd.Series,
                      title: str = "回撤图",
                      save_path: Optional[str] = None):
        """
        绘制回撤图
        Args:
            portfolio_values: 投资组合价值序列
            title: 图表标题
            save_path: 保存路径 (可选)
        """
        if not HAS_MATPLOTLIB:
            print("需要安装 matplotlib 才能绘制图表")
            return

        cummax = portfolio_values.cummax()
        drawdown = (portfolio_values - cummax) / cummax * 100

        fig, ax = plt.subplots(figsize=(12, 4))

        ax.fill_between(drawdown.index, drawdown.values, 0,
                        color='red', alpha=0.3)
        ax.plot(drawdown.index, drawdown.values, color='red', linewidth=1)

        ax.set_title(title, fontsize=14, fontweight='bold')
        ax.set_xlabel('日期')
        ax.set_ylabel('回撤 (%)')
        ax.grid(True, alpha=0.3)

        # 标注最大回撤
        max_dd_idx = drawdown.idxmin()
        max_dd = drawdown.min()
        ax.annotate(f'最大回撤: {max_dd:.2f}%',
                   xy=(max_dd_idx, max_dd),
                   xytext=(max_dd_idx, max_dd - 5),
                   fontsize=10,
                   arrowprops=dict(arrowstyle='->', color='black'))

        ax.xaxis.set_major_formatter(mdates.DateFormatter('%Y-%m'))
        ax.xaxis.set_major_locator(mdates.MonthLocator(interval=3))
        plt.xticks(rotation=45)

        plt.tight_layout()

        if save_path:
            plt.savefig(save_path, dpi=150, bbox_inches='tight')
        plt.show()

    def plot_weights(self, weights_df: pd.DataFrame,
                     title: str = "资产权重变化",
                     save_path: Optional[str] = None):
        """
        绘制资产权重堆叠面积图
        Args:
            weights_df: 权重DataFrame (索引为日期，列为资产)
            title: 图表标题
            save_path: 保存路径 (可选)
        """
        if not HAS_MATPLOTLIB:
            print("需要安装 matplotlib 才能绘制图表")
            return

        if weights_df.empty:
            print("没有权重数据")
            return

        fig, ax = plt.subplots(figsize=(12, 6))

        # 移除CASH列如果存在
        plot_df = weights_df.drop(columns=['CASH'], errors='ignore')

        ax.stackplot(plot_df.index, plot_df.T.values * 100,
                    labels=plot_df.columns, alpha=0.8)

        ax.set_title(title, fontsize=14, fontweight='bold')
        ax.set_xlabel('日期')
        ax.set_ylabel('权重 (%)')
        ax.legend(loc='upper left', bbox_to_anchor=(1, 1))
        ax.set_ylim(0, 100)
        ax.grid(True, alpha=0.3)

        ax.xaxis.set_major_formatter(mdates.DateFormatter('%Y-%m'))
        ax.xaxis.set_major_locator(mdates.MonthLocator(interval=3))
        plt.xticks(rotation=45)

        plt.tight_layout()

        if save_path:
            plt.savefig(save_path, dpi=150, bbox_inches='tight')
        plt.show()

    def plot_monthly_returns(self, portfolio_values: pd.Series,
                             title: str = "月度收益热力图",
                             save_path: Optional[str] = None):
        """
        绘制月度收益热力图
        Args:
            portfolio_values: 投资组合价值序列
            title: 图表标题
            save_path: 保存路径 (可选)
        """
        if not HAS_MATPLOTLIB:
            print("需要安装 matplotlib 才能绘制图表")
            return

        # 计算月度收益
        monthly = portfolio_values.resample('M').last()
        monthly_returns = monthly.pct_change().dropna() * 100

        # 创建年月矩阵
        years = monthly_returns.index.year.unique()
        months = range(1, 13)

        data = np.full((len(years), 12), np.nan)
        for i, year in enumerate(years):
            for j, month in enumerate(months):
                mask = (monthly_returns.index.year == year) & (monthly_returns.index.month == month)
                if mask.any():
                    data[i, j] = monthly_returns[mask].values[0]

        fig, ax = plt.subplots(figsize=(14, len(years) * 0.5 + 2))

        im = ax.imshow(data, cmap='RdYlGn', aspect='auto', vmin=-10, vmax=10)

        ax.set_xticks(range(12))
        ax.set_xticklabels(['1月', '2月', '3月', '4月', '5月', '6月',
                           '7月', '8月', '9月', '10月', '11月', '12月'])
        ax.set_yticks(range(len(years)))
        ax.set_yticklabels(years)

        # 添加数值标注
        for i in range(len(years)):
            for j in range(12):
                if not np.isnan(data[i, j]):
                    text = ax.text(j, i, f'{data[i, j]:.1f}%',
                                  ha='center', va='center', fontsize=8)

        ax.set_title(title, fontsize=14, fontweight='bold')
        plt.colorbar(im, ax=ax, label='收益率 (%)')

        plt.tight_layout()

        if save_path:
            plt.savefig(save_path, dpi=150, bbox_inches='tight')
        plt.show()
