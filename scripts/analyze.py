#!/usr/bin/env python3
"""回测结果分析脚本"""

import sys
import os

# 添加python目录到路径
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'python'))

from analysis import BacktestAnalyzer
from visualization import ReportGenerator


def main():
    if len(sys.argv) < 2:
        print("用法: python analyze.py <result.json> [output_report.html]")
        print("示例: python analyze.py output/result.json output/report.html")
        sys.exit(1)

    result_path = sys.argv[1]
    report_path = sys.argv[2] if len(sys.argv) > 2 else None

    # 检查文件是否存在
    if not os.path.exists(result_path):
        print(f"错误: 文件不存在 - {result_path}")
        sys.exit(1)

    # 分析结果
    print(f"正在分析: {result_path}")
    analyzer = BacktestAnalyzer(result_path)

    # 打印报告
    analyzer.print_report()

    # 生成HTML报告
    if report_path:
        metrics = analyzer.calculate_all_metrics()
        generator = ReportGenerator()
        generator.generate_html_report(metrics, report_path)


if __name__ == '__main__':
    main()
