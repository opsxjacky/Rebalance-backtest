#!/usr/bin/env python3
"""
Yahoo Finance 历史数据下载与转换脚本
用于将雪盈账户股票的历史数据转换为回测系统可用格式
"""

import yfinance as yf
import pandas as pd
import numpy as np
from datetime import datetime, timedelta
import os
import sys

# 雪盈账户持仓数据（来自 Notion）
XUEYING_HOLDINGS = {
    "QQQ": {
        "name": "Invesco QQQ Trust",
        "asset_type": "ETF",
        "is_core": True,
        "is_tech": True,
        "current_pe": 34.04,
        "current_pe_rank": 95,
    },
    "SPY": {
        "name": "SPDR S&P 500",
        "asset_type": "ETF",
        "is_core": True,
        "is_tech": False,
        "current_pe": 27.71,
        "current_pe_rank": 98.33,
    },
    "SMH": {
        "name": "VanEck Semiconductor ETF",
        "asset_type": "ETF",
        "is_core": False,
        "is_tech": True,
        "current_pe": 42.9,
        "current_pe_rank": 98.33,
    },
    "QTUM": {
        "name": "Defiance Quantum ETF",
        "asset_type": "ETF",
        "is_core": False,
        "is_tech": True,
        "current_pe": 31.9,
        "current_pe_rank": 96.67,
    },
    "NLR": {
        "name": "VanEck Uranium Nuclear ETF",
        "asset_type": "ETF",
        "is_core": False,
        "is_tech": False,
        "current_pe": 28.65,
        "current_pe_rank": 96.67,
    },
    "PPA": {
        "name": "Invesco Aerospace Defense ETF",
        "asset_type": "ETF",
        "is_core": False,
        "is_tech": False,
        "current_pe": 34.74,
        "current_pe_rank": 98.33,
    },
    "DXJ": {
        "name": "WisdomTree Japan Hedged Equity",
        "asset_type": "ETF",
        "is_core": True,  # 核心资产（日经）
        "is_tech": False,
        "current_pe": 15.93,
        "current_pe_rank": 98.33,
    },
    "BRK-B": {  # Yahoo Finance 用 BRK-B 而不是 BRK.B
        "name": "Berkshire Hathaway",
        "asset_type": "个股",
        "is_core": False,
        "is_tech": False,
        "current_pe": 16.08,
        "current_pe_rank": 88.33,
        "roe": 10.17,
        "peg": None,
    },
    "ARKW": {
        "name": "ARK Next Gen Internet ETF",
        "asset_type": "ETF",
        "is_core": False,
        "is_tech": True,
        "current_pe": 22.96,
        "current_pe_rank": 90,
    },
}

# PE百分位历史估算参数（基于历史平均）
# 这些是模拟值，真实回测需要历史PE百分位数据
PE_HISTORY_PARAMS = {
    "SPY": {"mean_pe": 22, "std_pe": 4, "mean_rank": 60},
    "QQQ": {"mean_pe": 28, "std_pe": 6, "mean_rank": 55},
    "SMH": {"mean_pe": 25, "std_pe": 8, "mean_rank": 50},
    "DXJ": {"mean_pe": 14, "std_pe": 3, "mean_rank": 50},
    "default": {"mean_pe": 20, "std_pe": 5, "mean_rank": 50},
}


def download_price_data(symbol: str, start_date: str, end_date: str) -> pd.DataFrame:
    """从 Yahoo Finance 下载价格数据"""
    print(f"正在下载 {symbol} 数据...")
    try:
        ticker = yf.Ticker(symbol)
        df = ticker.history(start=start_date, end=end_date)
        
        if df.empty:
            print(f"警告: {symbol} 没有数据")
            return pd.DataFrame()
        
        # 重命名列
        df = df.reset_index()
        df = df.rename(columns={
            "Date": "Date",
            "Open": "Open",
            "High": "High",
            "Low": "Low",
            "Close": "Close",
            "Volume": "Volume",
        })
        
        # 只保留需要的列
        df = df[["Date", "Open", "High", "Low", "Close", "Volume"]]
        df["Adj Close"] = df["Close"]  # 简化处理
        
        print(f"  下载了 {len(df)} 条记录")
        return df
    except Exception as e:
        print(f"错误: 下载 {symbol} 失败: {e}")
        return pd.DataFrame()


def estimate_pe_rank(current_price: float, base_price: float, 
                     current_pe: float, current_pe_rank: float,
                     params: dict) -> tuple:
    """
    根据价格变化估算历史 PE 和 PE百分位
    这是一个简化模型，真实情况需要历史盈利数据
    """
    # 价格变化比例
    price_ratio = current_price / base_price if base_price > 0 else 1
    
    # 估算PE (简化：假设PE与价格成正比变化)
    estimated_pe = current_pe / price_ratio if price_ratio > 0 else current_pe
    
    # 估算PE百分位 (使用正态分布CDF近似)
    mean_pe = params.get("mean_pe", 20)
    std_pe = params.get("std_pe", 5)
    
    z_score = (estimated_pe - mean_pe) / std_pe if std_pe > 0 else 0
    # 简化CDF计算
    estimated_rank = 50 + z_score * 20  # 近似正态分布
    estimated_rank = max(0, min(100, estimated_rank))
    
    return estimated_pe, estimated_rank


def add_fundamental_data(df: pd.DataFrame, symbol: str, holding_info: dict) -> pd.DataFrame:
    """添加基本面数据列"""
    if df.empty:
        return df
    
    # 获取参数
    params = PE_HISTORY_PARAMS.get(symbol, PE_HISTORY_PARAMS["default"])
    current_pe = holding_info.get("current_pe", 20)
    current_pe_rank = holding_info.get("current_pe_rank", 50)
    
    # 最后一天的价格作为基准
    base_price = df["Close"].iloc[-1]
    
    # 计算每日的估算PE和PE百分位
    pe_values = []
    pe_rank_values = []
    
    for idx, row in df.iterrows():
        pe, rank = estimate_pe_rank(
            row["Close"], base_price, current_pe, current_pe_rank, params
        )
        pe_values.append(round(pe, 2))
        pe_rank_values.append(round(rank, 2))
    
    df["PE"] = pe_values
    df["PE_Rank"] = pe_rank_values
    
    # 添加其他基本面数据
    df["PEG"] = holding_info.get("peg", 1.8)  # 默认PEG
    df["ROE"] = holding_info.get("roe", 15)   # 默认ROE
    df["Asset_Type"] = holding_info.get("asset_type", "ETF")
    df["Name"] = holding_info.get("name", symbol)
    df["Is_Core"] = str(holding_info.get("is_core", False)).lower()
    df["Is_Tech"] = str(holding_info.get("is_tech", False)).lower()
    
    return df


def save_to_csv(df: pd.DataFrame, symbol: str, output_dir: str):
    """保存为CSV文件"""
    if df.empty:
        return
    
    # 格式化日期
    df["Date"] = pd.to_datetime(df["Date"]).dt.strftime("%Y-%m-%d")
    
    # 调整列顺序
    columns = ["Date", "Open", "High", "Low", "Close", "Volume", "Adj Close",
               "PE", "PE_Rank", "PEG", "ROE", "Asset_Type", "Name", "Is_Core", "Is_Tech"]
    df = df[columns]
    
    # 输出文件名（处理特殊字符）
    output_symbol = symbol.replace("-", ".")  # BRK-B -> BRK.B
    output_path = os.path.join(output_dir, f"{output_symbol}.csv")
    
    df.to_csv(output_path, index=False)
    print(f"已保存: {output_path} ({len(df)} 条记录)")


def main():
    # 配置
    start_date = "2022-01-01"
    end_date = "2026-01-06"
    output_dir = "data/xueying"
    
    # 确保输出目录存在
    os.makedirs(output_dir, exist_ok=True)
    
    print(f"=" * 50)
    print(f"雪盈账户历史数据下载")
    print(f"日期范围: {start_date} ~ {end_date}")
    print(f"输出目录: {output_dir}")
    print(f"=" * 50)
    
    # 下载并转换每个标的
    for symbol, info in XUEYING_HOLDINGS.items():
        print(f"\n处理 {symbol}...")
        
        # 下载价格数据
        df = download_price_data(symbol, start_date, end_date)
        
        if df.empty:
            continue
        
        # 添加基本面数据
        df = add_fundamental_data(df, symbol, info)
        
        # 保存CSV
        save_to_csv(df, symbol, output_dir)
    
    print(f"\n{'=' * 50}")
    print("数据下载完成!")
    print(f"文件保存在: {output_dir}/")
    print("请运行回测: ./backtest run --config configs/xueying_config.yaml")
    print(f"{'=' * 50}")


if __name__ == "__main__":
    main()
