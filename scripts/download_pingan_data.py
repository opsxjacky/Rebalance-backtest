#!/usr/bin/env python3
"""
A股/港股 ETF 历史数据下载脚本
用于平安证券账户回测
使用 akshare 或 yfinance 获取数据
"""

import yfinance as yf
import pandas as pd
import numpy as np
from datetime import datetime
import os

# 平安证券持仓数据 (来自 Notion)
# A股ETF需要用不同的数据源，这里使用模拟数据或尝试通过yfinance获取
PINGAN_HOLDINGS = {
    # A股ETF (无法直接从yfinance获取，需要模拟或使用其他数据源)
    "159920": {
        "name": "华夏恒生ETF",
        "asset_type": "ETF",
        "is_core": False,
        "is_tech": False,
        "target_weight": 0.025,
        "current_pe_rank": 21.97,
        "pb": 0.91,
    },
    "159941": {
        "name": "广发纳斯达克100ETF",
        "asset_type": "ETF",
        "is_core": True,
        "is_tech": True,
        "target_weight": 0.05,
        "current_pe_rank": 95,
        "yahoo_proxy": "QQQ",  # 使用QQQ作为代理
    },
    "510300": {
        "name": "华泰柏瑞沪深300ETF",
        "asset_type": "ETF",
        "is_core": True,
        "is_tech": False,
        "target_weight": 0.10,
        "current_pe_rank": 79.86,
        "pb": 1.48,
        "yahoo_proxy": "ASHR",  # 使用ASHR (沪深300 ETF) 作为代理
    },
    "510500": {
        "name": "南方中证500ETF",
        "asset_type": "ETF",
        "is_core": False,
        "is_tech": False,
        "target_weight": 0.05,
        "current_pe_rank": 78.21,
        "pb": 2.28,
    },
    "511010": {
        "name": "国泰上证5年期国债ETF",
        "asset_type": "债券",
        "is_core": False,
        "is_tech": False,
        "target_weight": 0.10,
        "yield": 1.64,
    },
    "511090": {
        "name": "鹏扬中债-30年期国债ETF",
        "asset_type": "债券",
        "is_core": False,
        "is_tech": False,
        "target_weight": 0.095,
        "yield": 2.28,
    },
    "511260": {
        "name": "国泰上证10年期国债ETF",
        "asset_type": "债券",
        "is_core": False,
        "is_tech": False,
        "target_weight": 0.10,
        "yield": 1.86,
    },
    "511380": {
        "name": "博时可转债ETF",
        "asset_type": "债券",
        "is_core": False,
        "is_tech": False,
        "target_weight": 0.05,
    },
    "511520": {
        "name": "富国中债7-10年政策性金融债ETF",
        "asset_type": "债券",
        "is_core": False,
        "is_tech": False,
        "target_weight": 0.10,
        "yield": 1.86,
    },
    "513050": {
        "name": "易方达中证海外中国互联网50",
        "asset_type": "ETF",
        "is_core": False,
        "is_tech": True,
        "target_weight": 0.03,
        "current_pe_rank": 98.33,
        "yahoo_proxy": "KWEB",  # 使用KWEB作为代理
    },
    "513500": {
        "name": "博时标普500ETF",
        "asset_type": "ETF",
        "is_core": True,
        "is_tech": False,
        "target_weight": 0.10,
        "current_pe_rank": 98.33,
        "yahoo_proxy": "SPY",  # 使用SPY作为代理
    },
    "518880": {
        "name": "华安黄金易ETF",
        "asset_type": "黄金",
        "is_core": False,
        "is_tech": False,
        "target_weight": 0.10,
        "yahoo_proxy": "GLD",  # 使用GLD作为代理
    },
}

# 可以从Yahoo Finance获取的代理标的
YAHOO_PROXIES = {
    "159941": "QQQ",
    "510300": "ASHR",
    "513050": "KWEB",
    "513500": "SPY",
    "518880": "GLD",
}


def download_proxy_data(symbol: str, proxy: str, start_date: str, end_date: str) -> pd.DataFrame:
    """从 Yahoo Finance 下载代理标的数据"""
    print(f"正在下载 {symbol} (代理: {proxy}) 数据...")
    try:
        ticker = yf.Ticker(proxy)
        df = ticker.history(start=start_date, end=end_date)
        
        if df.empty:
            print(f"警告: {proxy} 没有数据")
            return pd.DataFrame()
        
        df = df.reset_index()
        df = df.rename(columns={
            "Date": "Date",
            "Open": "Open",
            "High": "High",
            "Low": "Low",
            "Close": "Close",
            "Volume": "Volume",
        })
        
        df = df[["Date", "Open", "High", "Low", "Close", "Volume"]]
        df["Adj Close"] = df["Close"]
        
        # 调整价格比例（A股ETF价格通常较低）
        # 这里只是示例，实际需要根据汇率和净值比例调整
        price_ratio = 0.01 if symbol.startswith("5") else 1
        for col in ["Open", "High", "Low", "Close", "Adj Close"]:
            df[col] = df[col] * price_ratio
        
        print(f"  下载了 {len(df)} 条记录")
        return df
    except Exception as e:
        print(f"错误: 下载 {proxy} 失败: {e}")
        return pd.DataFrame()


def generate_simulated_data(symbol: str, info: dict, start_date: str, end_date: str) -> pd.DataFrame:
    """生成模拟数据（用于无法获取真实数据的标的）"""
    print(f"生成 {symbol} 模拟数据...")
    
    dates = pd.date_range(start=start_date, end=end_date, freq='B')
    
    # 根据资产类型设置不同的波动率和趋势
    asset_type = info.get("asset_type", "ETF")
    if asset_type == "债券":
        annual_return = 0.04
        volatility = 0.02
        base_price = 100
    elif asset_type == "黄金":
        annual_return = 0.08
        volatility = 0.15
        base_price = 5
    else:
        annual_return = 0.10
        volatility = 0.20
        base_price = 1 if symbol.startswith("5") else 100
    
    # 生成价格序列
    daily_return = annual_return / 252
    daily_vol = volatility / np.sqrt(252)
    
    np.random.seed(int(symbol) % 10000)
    returns = np.random.normal(daily_return, daily_vol, len(dates))
    prices = base_price * np.cumprod(1 + returns)
    
    df = pd.DataFrame({
        "Date": dates,
        "Open": prices * (1 - np.random.uniform(0, 0.01, len(dates))),
        "High": prices * (1 + np.random.uniform(0, 0.02, len(dates))),
        "Low": prices * (1 - np.random.uniform(0, 0.02, len(dates))),
        "Close": prices,
        "Volume": np.random.randint(1000000, 10000000, len(dates)),
        "Adj Close": prices,
    })
    
    print(f"  生成了 {len(df)} 条记录")
    return df


def add_fundamental_data(df: pd.DataFrame, symbol: str, info: dict) -> pd.DataFrame:
    """添加基本面数据"""
    if df.empty:
        return df
    
    # PE百分位 (需要历史数据估算)
    current_pe_rank = info.get("current_pe_rank", 50)
    # 简化处理：根据价格位置估算PE百分位
    price_pct = df["Close"].rank(pct=True) * 100
    pe_rank_series = current_pe_rank * 0.5 + price_pct * 0.5  # 混合当前和价格位置
    
    df["PE"] = 20  # 简化
    df["PE_Rank"] = pe_rank_series.round(2)
    df["PEG"] = 1.5
    df["ROE"] = info.get("yield", 15)  # 用ROE存储Yield for 债券
    df["Asset_Type"] = info.get("asset_type", "ETF")
    df["Name"] = info.get("name", symbol)
    df["Is_Core"] = str(info.get("is_core", False)).lower()
    df["Is_Tech"] = str(info.get("is_tech", False)).lower()
    
    return df


def save_to_csv(df: pd.DataFrame, symbol: str, output_dir: str):
    """保存为CSV"""
    if df.empty:
        return
    
    df["Date"] = pd.to_datetime(df["Date"]).dt.strftime("%Y-%m-%d")
    
    columns = ["Date", "Open", "High", "Low", "Close", "Volume", "Adj Close",
               "PE", "PE_Rank", "PEG", "ROE", "Asset_Type", "Name", "Is_Core", "Is_Tech"]
    df = df[columns]
    
    output_path = os.path.join(output_dir, f"{symbol}.csv")
    df.to_csv(output_path, index=False)
    print(f"已保存: {output_path} ({len(df)} 条记录)")


def main():
    start_date = "2022-01-01"
    end_date = "2026-01-06"
    output_dir = "data/pingan"
    
    os.makedirs(output_dir, exist_ok=True)
    
    print("=" * 50)
    print("平安证券 A股 ETF 历史数据准备")
    print(f"日期范围: {start_date} ~ {end_date}")
    print(f"输出目录: {output_dir}")
    print("=" * 50)
    
    for symbol, info in PINGAN_HOLDINGS.items():
        print(f"\n处理 {symbol} ({info['name']})...")
        
        # 优先使用Yahoo代理数据
        if symbol in YAHOO_PROXIES:
            df = download_proxy_data(symbol, YAHOO_PROXIES[symbol], start_date, end_date)
        else:
            # 无代理的使用模拟数据
            df = generate_simulated_data(symbol, info, start_date, end_date)
        
        if df.empty:
            continue
        
        df = add_fundamental_data(df, symbol, info)
        save_to_csv(df, symbol, output_dir)
    
    print(f"\n{'=' * 50}")
    print("数据准备完成!")
    print(f"文件保存在: {output_dir}/")
    print("运行回测: ./backtest run --config configs/pingan_config.yaml")
    print("=" * 50)


if __name__ == "__main__":
    main()
