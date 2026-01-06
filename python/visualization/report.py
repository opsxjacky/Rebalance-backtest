"""HTML报告生成模块"""

import json
from typing import Dict, Optional
from datetime import datetime


class ReportGenerator:
    """HTML报告生成器"""

    def __init__(self):
        self.template = self._get_template()

    def generate_html_report(self, metrics: Dict, output_path: str,
                             title: str = "回测报告"):
        """
        生成HTML报告
        Args:
            metrics: 性能指标字典
            output_path: 输出文件路径
            title: 报告标题
        """
        html = self.template.format(
            title=title,
            generated_time=datetime.now().strftime('%Y-%m-%d %H:%M:%S'),
            strategy_name=metrics.get('strategy_name', 'Unknown'),
            initial_capital=f"${metrics.get('initial_capital', 0):,.2f}",
            final_value=f"${metrics.get('final_value', 0):,.2f}",
            total_return=f"{metrics.get('total_return', 0) * 100:.2f}%",
            annualized_return=f"{metrics.get('annualized_return', 0) * 100:.2f}%",
            cagr=f"{metrics.get('cagr', 0) * 100:.2f}%",
            volatility=f"{metrics.get('volatility', 0) * 100:.2f}%",
            max_drawdown=f"{metrics.get('max_drawdown', 0) * 100:.2f}%",
            max_drawdown_duration=metrics.get('max_drawdown_duration', 0),
            var_95=f"{metrics.get('var_95', 0) * 100:.2f}%",
            cvar_95=f"{metrics.get('cvar_95', 0) * 100:.2f}%",
            sharpe_ratio=f"{metrics.get('sharpe_ratio', 0):.3f}",
            sortino_ratio=f"{metrics.get('sortino_ratio', 0):.3f}",
            calmar_ratio=f"{metrics.get('calmar_ratio', 0):.3f}",
            total_trades=metrics.get('total_trades', 0),
            buy_trades=metrics.get('buy_trades', 0),
            sell_trades=metrics.get('sell_trades', 0),
            total_fees=f"${metrics.get('total_fees', 0):,.2f}",
            avg_trade_value=f"${metrics.get('avg_trade_value', 0):,.2f}",
        )

        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(html)

        print(f"报告已生成: {output_path}")

    def _get_template(self) -> str:
        return '''<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{title}</title>
    <style>
        * {{
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }}
        body {{
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            background-color: #f5f5f5;
            color: #333;
            line-height: 1.6;
        }}
        .container {{
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }}
        .header {{
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px 20px;
            text-align: center;
            margin-bottom: 30px;
            border-radius: 10px;
        }}
        .header h1 {{
            font-size: 2.5em;
            margin-bottom: 10px;
        }}
        .header p {{
            opacity: 0.9;
        }}
        .section {{
            background: white;
            border-radius: 10px;
            padding: 25px;
            margin-bottom: 20px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }}
        .section h2 {{
            color: #667eea;
            margin-bottom: 20px;
            padding-bottom: 10px;
            border-bottom: 2px solid #667eea;
        }}
        .metrics-grid {{
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
        }}
        .metric-card {{
            background: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
        }}
        .metric-card .value {{
            font-size: 1.8em;
            font-weight: bold;
            color: #667eea;
        }}
        .metric-card .label {{
            color: #666;
            font-size: 0.9em;
            margin-top: 5px;
        }}
        .metric-card.positive .value {{
            color: #28a745;
        }}
        .metric-card.negative .value {{
            color: #dc3545;
        }}
        .footer {{
            text-align: center;
            padding: 20px;
            color: #666;
            font-size: 0.9em;
        }}
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>{title}</h1>
            <p>策略: {strategy_name} | 生成时间: {generated_time}</p>
        </div>

        <div class="section">
            <h2>收益指标</h2>
            <div class="metrics-grid">
                <div class="metric-card">
                    <div class="value">{initial_capital}</div>
                    <div class="label">初始资金</div>
                </div>
                <div class="metric-card">
                    <div class="value">{final_value}</div>
                    <div class="label">最终价值</div>
                </div>
                <div class="metric-card positive">
                    <div class="value">{total_return}</div>
                    <div class="label">总收益率</div>
                </div>
                <div class="metric-card positive">
                    <div class="value">{annualized_return}</div>
                    <div class="label">年化收益率</div>
                </div>
                <div class="metric-card positive">
                    <div class="value">{cagr}</div>
                    <div class="label">CAGR</div>
                </div>
            </div>
        </div>

        <div class="section">
            <h2>风险指标</h2>
            <div class="metrics-grid">
                <div class="metric-card">
                    <div class="value">{volatility}</div>
                    <div class="label">年化波动率</div>
                </div>
                <div class="metric-card negative">
                    <div class="value">{max_drawdown}</div>
                    <div class="label">最大回撤</div>
                </div>
                <div class="metric-card">
                    <div class="value">{max_drawdown_duration} 天</div>
                    <div class="label">最大回撤持续时间</div>
                </div>
                <div class="metric-card">
                    <div class="value">{var_95}</div>
                    <div class="label">VaR (95%)</div>
                </div>
                <div class="metric-card">
                    <div class="value">{cvar_95}</div>
                    <div class="label">CVaR (95%)</div>
                </div>
            </div>
        </div>

        <div class="section">
            <h2>风险调整收益</h2>
            <div class="metrics-grid">
                <div class="metric-card">
                    <div class="value">{sharpe_ratio}</div>
                    <div class="label">Sharpe Ratio</div>
                </div>
                <div class="metric-card">
                    <div class="value">{sortino_ratio}</div>
                    <div class="label">Sortino Ratio</div>
                </div>
                <div class="metric-card">
                    <div class="value">{calmar_ratio}</div>
                    <div class="label">Calmar Ratio</div>
                </div>
            </div>
        </div>

        <div class="section">
            <h2>交易统计</h2>
            <div class="metrics-grid">
                <div class="metric-card">
                    <div class="value">{total_trades}</div>
                    <div class="label">总交易次数</div>
                </div>
                <div class="metric-card">
                    <div class="value">{buy_trades}</div>
                    <div class="label">买入交易</div>
                </div>
                <div class="metric-card">
                    <div class="value">{sell_trades}</div>
                    <div class="label">卖出交易</div>
                </div>
                <div class="metric-card">
                    <div class="value">{total_fees}</div>
                    <div class="label">总交易费用</div>
                </div>
                <div class="metric-card">
                    <div class="value">{avg_trade_value}</div>
                    <div class="label">平均交易金额</div>
                </div>
            </div>
        </div>

        <div class="footer">
            <p>Rebalance-Backtest 回测系统 | Powered by Go + Python</p>
        </div>
    </div>
</body>
</html>'''
