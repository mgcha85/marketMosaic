# Lightweight Charts Integration

## Installation

```bash
npm install lightweight-charts
```

## Basic Line Chart

### EquityCurve.svelte

```svelte
<script>
  import { createChart, ColorType } from 'lightweight-charts';
  
  let { data, title = 'Equity Curve' } = $props();
  
  let chartContainer;
  let chart = $state(null);
  let lineSeries = $state(null);
  
  // Create chart on mount
  $effect(() => {
    if (!chartContainer) return;
    
    chart = createChart(chartContainer, {
      layout: {
        background: { type: ColorType.Solid, color: 'transparent' },
        textColor: 'hsl(var(--bc))', // DaisyUI base-content
      },
      grid: {
        vertLines: { color: 'hsl(var(--bc) / 0.1)' },
        horzLines: { color: 'hsl(var(--bc) / 0.1)' },
      },
      width: chartContainer.clientWidth,
      height: 300,
      rightPriceScale: {
        borderVisible: false,
      },
      timeScale: {
        borderVisible: false,
        timeVisible: true,
      },
    });
    
    lineSeries = chart.addLineSeries({
      color: 'hsl(var(--p))', // DaisyUI primary
      lineWidth: 2,
      priceFormat: {
        type: 'percent',
      },
    });
    
    // Handle resize
    const resizeObserver = new ResizeObserver(entries => {
      if (entries.length === 0 || entries[0].target !== chartContainer) return;
      const { width } = entries[0].contentRect;
      chart.applyOptions({ width });
    });
    
    resizeObserver.observe(chartContainer);
    
    return () => {
      resizeObserver.disconnect();
      chart.remove();
    };
  });
  
  // Update data when it changes
  $effect(() => {
    if (lineSeries && data) {
      lineSeries.setData(data);
      chart.timeScale().fitContent();
    }
  });
</script>

<div class="card bg-base-100 shadow">
  <div class="card-body">
    <h3 class="card-title">{title}</h3>
    <div bind:this={chartContainer} class="w-full"></div>
  </div>
</div>
```

## Candlestick Chart

### CandleChart.svelte

```svelte
<script>
  import { createChart, ColorType } from 'lightweight-charts';
  
  let { data, title = 'Price Chart' } = $props();
  
  let chartContainer;
  let chart = $state(null);
  let candleSeries = $state(null);
  let volumeSeries = $state(null);
  
  $effect(() => {
    if (!chartContainer) return;
    
    chart = createChart(chartContainer, {
      layout: {
        background: { type: ColorType.Solid, color: 'transparent' },
        textColor: 'hsl(var(--bc))',
      },
      grid: {
        vertLines: { color: 'hsl(var(--bc) / 0.1)' },
        horzLines: { color: 'hsl(var(--bc) / 0.1)' },
      },
      width: chartContainer.clientWidth,
      height: 400,
    });
    
    // Candlestick series
    candleSeries = chart.addCandlestickSeries({
      upColor: '#22c55e',      // Green
      downColor: '#ef4444',    // Red
      borderUpColor: '#22c55e',
      borderDownColor: '#ef4444',
      wickUpColor: '#22c55e',
      wickDownColor: '#ef4444',
    });
    
    // Volume series
    volumeSeries = chart.addHistogramSeries({
      color: 'hsl(var(--p) / 0.3)',
      priceFormat: {
        type: 'volume',
      },
      priceScaleId: '',
      scaleMargins: {
        top: 0.8,
        bottom: 0,
      },
    });
    
    const resizeObserver = new ResizeObserver(entries => {
      if (entries.length === 0) return;
      const { width } = entries[0].contentRect;
      chart.applyOptions({ width });
    });
    
    resizeObserver.observe(chartContainer);
    
    return () => {
      resizeObserver.disconnect();
      chart.remove();
    };
  });
  
  $effect(() => {
    if (!candleSeries || !data) return;
    
    // Format: { time: '2026-04-11', open: 100, high: 105, low: 98, close: 103, volume: 1000000 }
    const candleData = data.map(d => ({
      time: d.time,
      open: d.open,
      high: d.high,
      low: d.low,
      close: d.close,
    }));
    
    const volumeData = data.map(d => ({
      time: d.time,
      value: d.volume,
      color: d.close >= d.open ? 'rgba(34, 197, 94, 0.5)' : 'rgba(239, 68, 68, 0.5)',
    }));
    
    candleSeries.setData(candleData);
    volumeSeries.setData(volumeData);
    chart.timeScale().fitContent();
  });
</script>

<div class="card bg-base-100 shadow">
  <div class="card-body">
    <h3 class="card-title">{title}</h3>
    <div bind:this={chartContainer} class="w-full"></div>
  </div>
</div>
```

## Data Formatting

### API Response to Chart Data

```javascript
// api.js
export function formatEquityCurve(trades) {
  let cumulative = 0;
  const data = [];
  
  for (const trade of trades) {
    cumulative += trade.pnl_pct;
    data.push({
      time: trade.exit_date,
      value: cumulative,
    });
  }
  
  return data;
}

export function formatCandleData(history) {
  return history.map(h => ({
    time: h.date,
    open: h.open_price,
    high: h.high_price,
    low: h.low_price,
    close: h.close_price,
    volume: h.volume,
  }));
}
```

## Adding Markers

### Signal Entry/Exit Markers

```svelte
<script>
  $effect(() => {
    if (!candleSeries || !signals) return;
    
    const markers = signals.map(signal => ({
      time: signal.entry_date,
      position: 'belowBar',
      color: '#22c55e',
      shape: 'arrowUp',
      text: `진입 ${signal.score.toFixed(1)}`,
    }));
    
    candleSeries.setMarkers(markers);
  });
</script>
```

## Adding MA Lines

### Moving Average Overlay

```svelte
<script>
  let ma5Series = $state(null);
  let ma20Series = $state(null);
  
  $effect(() => {
    if (!chart) return;
    
    ma5Series = chart.addLineSeries({
      color: '#f59e0b',
      lineWidth: 1,
      title: 'MA5',
    });
    
    ma20Series = chart.addLineSeries({
      color: '#3b82f6',
      lineWidth: 1,
      title: 'MA20',
    });
  });
  
  $effect(() => {
    if (!ma5Series || !data) return;
    
    const ma5Data = calculateMA(data, 5);
    const ma20Data = calculateMA(data, 20);
    
    ma5Series.setData(ma5Data);
    ma20Series.setData(ma20Data);
  });
  
  function calculateMA(data, period) {
    const result = [];
    for (let i = period - 1; i < data.length; i++) {
      const slice = data.slice(i - period + 1, i + 1);
      const avg = slice.reduce((sum, d) => sum + d.close, 0) / period;
      result.push({ time: data[i].time, value: avg });
    }
    return result;
  }
</script>
```

## Tooltip

### Custom Tooltip

```svelte
<script>
  let tooltipData = $state(null);
  
  $effect(() => {
    if (!chart) return;
    
    chart.subscribeCrosshairMove(param => {
      if (!param.time || !param.seriesData) {
        tooltipData = null;
        return;
      }
      
      const candleData = param.seriesData.get(candleSeries);
      if (candleData) {
        tooltipData = {
          time: param.time,
          ...candleData,
          x: param.point?.x ?? 0,
          y: param.point?.y ?? 0,
        };
      }
    });
  });
</script>

{#if tooltipData}
  <div 
    class="absolute bg-base-100 shadow-lg rounded p-2 text-sm pointer-events-none z-10"
    style="left: {tooltipData.x + 20}px; top: {tooltipData.y}px"
  >
    <div class="font-bold">{tooltipData.time}</div>
    <div>시가: {tooltipData.open?.toLocaleString()}</div>
    <div>고가: {tooltipData.high?.toLocaleString()}</div>
    <div>저가: {tooltipData.low?.toLocaleString()}</div>
    <div>종가: {tooltipData.close?.toLocaleString()}</div>
  </div>
{/if}
```

## Integration Example

### OvernightDashboard.svelte

```svelte
<script>
  import EquityCurve from './EquityCurve.svelte';
  import CandleChart from './CandleChart.svelte';
  import { formatEquityCurve, formatCandleData } from '../api.js';
  
  let trades = $state([]);
  let selectedStock = $state(null);
  let stockHistory = $state([]);
  
  let equityData = $derived(formatEquityCurve(trades));
  let candleData = $derived(formatCandleData(stockHistory));
</script>

<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
  <EquityCurve data={equityData} title="수익 곡선" />
  
  {#if selectedStock}
    <CandleChart data={candleData} title="{selectedStock.name} 일봉" />
  {/if}
</div>
```
