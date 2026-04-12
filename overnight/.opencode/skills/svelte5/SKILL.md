---
name: svelte5
description: "Svelte 5 runes, Tauri desktop UI patterns, TailwindCSS + DaisyUI styling, and TypeScript component development. Triggers on: Svelte components, Tauri invoke flows, runes ($state, $derived, $effect, $props), event handling, lightweight-charts integration, or dashboard UI development."
version: 1.0.0
license: MIT
compatibility: opencode
rpi_phase: Implementation
trigger:
  - "Svelte"
  - ".svelte"
  - "$state"
  - "$derived"
  - "TailwindCSS"
  - "DaisyUI"
  - "dashboard"
  - "chart"
capabilities:
  - Build reactive Svelte 5 components with runes
  - Style with TailwindCSS utility classes and DaisyUI components
  - Integrate lightweight-charts for financial data visualization
  - Create responsive dashboard layouts
  - Handle forms and user interactions
metadata:
  author: marketMosaic
  category: frontend-development
---

# Svelte 5 Frontend Specialist

<role_definition>
You are the **Svelte 5 Frontend Specialist** for the Overnight Strategy dashboard. You build reactive, accessible desktop UIs using Svelte 5's runes syntax, Tauri invoke APIs, TailwindCSS for styling, and DaisyUI for component library. Your output must be production-ready with proper TypeScript types.
</role_definition>

<resources>
- **Runes Reference**: See `references/runes.md` for $state, $derived, $effect patterns
- **DaisyUI Components**: See `references/daisyui.md` for card, table, button patterns
- **Charts**: See `references/charts.md` for lightweight-charts integration
</resources>

## Quick Reference

| Topic | When to Use | Reference |
|-------|-------------|-----------|
| Reactive State | $state, $derived, $effect | [runes.md](references/runes.md) |
| UI Components | Cards, tables, modals | [daisyui.md](references/daisyui.md) |
| Financial Charts | Candlestick, line charts | [charts.md](references/charts.md) |

## Essential Patterns

### Reactive State with Runes

```svelte
<script>
  // Reactive state
  let signals = $state([]);
  let selectedDate = $state(new Date().toISOString().split('T')[0]);
  let isLoading = $state(false);
  
  // Derived values
  let highScoreSignals = $derived(
    signals.filter(s => s.score >= 8.0)
  );
  
  let totalCount = $derived(signals.length);
  
  // Side effects
  $effect(() => {
    console.log(`Selected date changed to: ${selectedDate}`);
    fetchSignals(selectedDate);
  });
</script>
```

### Props with $props()

```svelte
<script>
  // Define props with defaults
  let { 
    signal,
    onSelect = () => {},
    showDetails = false 
  } = $props();
</script>

<div class="card" onclick={() => onSelect(signal)}>
  <h3>{signal.name}</h3>
  {#if showDetails}
    <p>Score: {signal.score}</p>
  {/if}
</div>
```

### Tauri Invoke Pattern

```svelte
<script>
  import { invoke } from '@tauri-apps/api/core';
  
  let signals = $state([]);
  let error = $state(null);
  let isLoading = $state(false);
  
  async function fetchSignals(date) {
    isLoading = true;
    error = null;
    
    try {
      signals = await invoke('get_signals', { date });
    } catch (e) {
      error = e.message;
    } finally {
      isLoading = false;
    }
  }
</script>

{#if isLoading}
  <span class="loading loading-spinner"></span>
{:else if error}
  <div class="alert alert-error">{error}</div>
{:else}
  {#each signals as signal}
    <SignalCard {signal} />
  {/each}
{/if}
```

## DaisyUI Component Patterns

### Card Component

```svelte
<div class="card bg-base-100 shadow-xl">
  <div class="card-body">
    <h2 class="card-title">
      {signal.name}
      <div class="badge badge-primary">{signal.score.toFixed(1)}점</div>
    </h2>
    <p class="text-sm text-base-content/70">{signal.reasoning}</p>
    <div class="card-actions justify-end">
      <button class="btn btn-sm btn-outline">상세보기</button>
    </div>
  </div>
</div>
```

### Stats Display

```svelte
<div class="stats shadow">
  <div class="stat">
    <div class="stat-title">총 거래</div>
    <div class="stat-value">{stats.totalTrades}</div>
  </div>
  <div class="stat">
    <div class="stat-title">승률</div>
    <div class="stat-value text-primary">{stats.winRate}%</div>
  </div>
  <div class="stat">
    <div class="stat-title">누적 수익</div>
    <div class="stat-value text-success">+{stats.totalReturn}%</div>
  </div>
</div>
```

### Data Table

```svelte
<div class="overflow-x-auto">
  <table class="table table-zebra">
    <thead>
      <tr>
        <th>종목</th>
        <th>점수</th>
        <th>수익률</th>
      </tr>
    </thead>
    <tbody>
      {#each trades as trade}
        <tr>
          <td>{trade.name}</td>
          <td>{trade.score}</td>
          <td class:text-success={trade.pnl > 0} class:text-error={trade.pnl < 0}>
            {trade.pnl > 0 ? '+' : ''}{trade.pnl.toFixed(2)}%
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
```

## Project Structure

```
overnight/
├── src/
│   ├── lib/
│   │   ├── components/
│   │   │   ├── SignalCard.svelte
│   │   │   ├── ScoreBreakdown.svelte
│   │   │   ├── BacktestSummary.svelte
│   │   │   ├── EquityCurve.svelte
│   │   │   └── BacktestTradeTable.svelte
│   │   ├── stores/
│   │   └── api/
│   └── routes/
└── src-tauri/
```

## Common Mistakes

1. **Using fetch() for app-local actions** - Prefer Tauri `invoke()` for Rust-side commands
2. **Using `let` without `$state`** - Variables are not reactive without `$state()`
3. **Using `$effect` for derived values** - Use `$derived` instead, it's more efficient
4. **Mutating $state arrays directly** - Use `signals = [...signals, newSignal]` or `signals.push()` (Svelte 5 tracks mutations)
5. **Forgetting async in $effect** - $effect callbacks can't be async, use IIFE: `$effect(() => { (async () => { ... })(); });`

## Package Dependencies

```json
{
  "dependencies": {
    "@tauri-apps/api": "^2",
    "lightweight-charts": "^5.1.0",
    "daisyui": "^5.5.14"
  },
  "devDependencies": {
    "@sveltejs/vite-plugin-svelte": "^3.0.2",
    "svelte": "^5.0.0",
    "tailwindcss": "^3.4.17",
    "vite": "^5.1.0"
  }
}
```
  - "Tauri"
  - "invoke"
