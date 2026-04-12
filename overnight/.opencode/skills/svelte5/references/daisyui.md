# DaisyUI Component Patterns

## Setup

### tailwind.config.js

```js
/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte}'],
  theme: {
    extend: {},
  },
  plugins: [require('daisyui')],
  daisyui: {
    themes: ['light', 'dark', 'corporate'],
    darkTheme: 'dark',
  },
}
```

### Theme Switcher

```svelte
<script>
  let theme = $state('light');
  
  $effect(() => {
    document.documentElement.setAttribute('data-theme', theme);
  });
</script>

<select class="select select-bordered" bind:value={theme}>
  <option value="light">Light</option>
  <option value="dark">Dark</option>
  <option value="corporate">Corporate</option>
</select>
```

## Cards

### Signal Card

```svelte
<script>
  let { signal, onClick } = $props();
  
  let scoreColor = $derived(
    signal.score >= 8.5 ? 'badge-success' :
    signal.score >= 8.0 ? 'badge-primary' :
    'badge-warning'
  );
</script>

<div 
  class="card bg-base-100 shadow-xl hover:shadow-2xl transition-shadow cursor-pointer"
  onclick={() => onClick?.(signal)}
>
  <div class="card-body">
    <div class="flex justify-between items-start">
      <div>
        <h2 class="card-title">
          {signal.name}
          <span class="text-sm text-base-content/60">({signal.code})</span>
        </h2>
        <p class="text-sm text-base-content/70">테마: {signal.theme_name}</p>
      </div>
      <div class="badge {scoreColor} badge-lg">
        {signal.score.toFixed(1)}점
      </div>
    </div>
    
    <div class="divider my-2"></div>
    
    <p class="text-sm">{signal.reasoning}</p>
    
    <div class="flex gap-2 mt-2">
      <div class="badge badge-outline">뉴스 {signal.news_count}건</div>
      {#if signal.ma5_support}
        <div class="badge badge-outline badge-success">5일선 지지</div>
      {/if}
    </div>
    
    <div class="card-actions justify-end mt-4">
      <button class="btn btn-sm btn-outline">점수 상세</button>
      <button class="btn btn-sm btn-primary">캔들 보기</button>
    </div>
  </div>
</div>
```

### Score Breakdown Card

```svelte
<script>
  let { signal } = $props();
  
  let scores = $derived([
    { label: '테마', value: signal.score_theme, max: 2 },
    { label: '뉴스', value: signal.score_news, max: 3 },
    { label: '기술적', value: signal.score_technical, max: 3 },
    { label: '리스크', value: signal.score_risk, max: 2 },
  ]);
</script>

<div class="card bg-base-100 shadow-lg">
  <div class="card-body">
    <h3 class="card-title text-lg">점수 분석</h3>
    
    <div class="space-y-3">
      {#each scores as score}
        <div>
          <div class="flex justify-between text-sm mb-1">
            <span>{score.label}</span>
            <span>{score.value.toFixed(1)} / {score.max}</span>
          </div>
          <progress 
            class="progress progress-primary" 
            value={score.value} 
            max={score.max}
          ></progress>
        </div>
      {/each}
    </div>
    
    <div class="divider"></div>
    
    <div class="flex justify-between font-bold">
      <span>총점</span>
      <span class="text-primary">{signal.score.toFixed(1)} / 10</span>
    </div>
  </div>
</div>
```

## Stats

### Backtest Summary Stats

```svelte
<script>
  let { stats } = $props();
</script>

<div class="stats stats-vertical lg:stats-horizontal shadow w-full">
  <div class="stat">
    <div class="stat-figure text-primary">
      <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
      </svg>
    </div>
    <div class="stat-title">총 거래</div>
    <div class="stat-value text-primary">{stats.total_trades}</div>
    <div class="stat-desc">{stats.date_from} ~ {stats.date_to}</div>
  </div>
  
  <div class="stat">
    <div class="stat-title">승률</div>
    <div class="stat-value">{stats.win_rate.toFixed(1)}%</div>
    <div class="stat-desc">{stats.win_count}승 / {stats.loss_count}패</div>
  </div>
  
  <div class="stat">
    <div class="stat-title">평균 수익</div>
    <div class="stat-value text-secondary">
      {stats.avg_return > 0 ? '+' : ''}{stats.avg_return.toFixed(2)}%
    </div>
  </div>
  
  <div class="stat">
    <div class="stat-title">누적 수익</div>
    <div class="stat-value" class:text-success={stats.total_return > 0} class:text-error={stats.total_return < 0}>
      {stats.total_return > 0 ? '+' : ''}{stats.total_return.toFixed(2)}%
    </div>
    <div class="stat-desc text-error">MDD: {stats.max_drawdown.toFixed(2)}%</div>
  </div>
</div>
```

## Tables

### Trade History Table

```svelte
<script>
  let { trades, sortBy = $bindable('entry_date'), sortOrder = $bindable('desc') } = $props();
  
  function toggleSort(column) {
    if (sortBy === column) {
      sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = column;
      sortOrder = 'desc';
    }
  }
  
  let sortedTrades = $derived(
    [...trades].sort((a, b) => {
      const aVal = a[sortBy];
      const bVal = b[sortBy];
      const order = sortOrder === 'asc' ? 1 : -1;
      return aVal > bVal ? order : -order;
    })
  );
</script>

<div class="overflow-x-auto">
  <table class="table table-zebra">
    <thead>
      <tr>
        <th class="cursor-pointer" onclick={() => toggleSort('entry_date')}>
          날짜 {sortBy === 'entry_date' ? (sortOrder === 'asc' ? '↑' : '↓') : ''}
        </th>
        <th>종목</th>
        <th>테마</th>
        <th class="cursor-pointer" onclick={() => toggleSort('score')}>
          점수 {sortBy === 'score' ? (sortOrder === 'asc' ? '↑' : '↓') : ''}
        </th>
        <th>진입가</th>
        <th>익일가</th>
        <th class="cursor-pointer" onclick={() => toggleSort('pnl_pct')}>
          수익률 {sortBy === 'pnl_pct' ? (sortOrder === 'asc' ? '↑' : '↓') : ''}
        </th>
      </tr>
    </thead>
    <tbody>
      {#each sortedTrades as trade}
        <tr>
          <td>{trade.entry_date}</td>
          <td>
            <div class="font-medium">{trade.name}</div>
            <div class="text-sm text-base-content/60">{trade.code}</div>
          </td>
          <td>{trade.theme_name}</td>
          <td>
            <div class="badge badge-sm">{trade.score.toFixed(1)}</div>
          </td>
          <td>{trade.entry_price.toLocaleString()}원</td>
          <td>{trade.exit_price.toLocaleString()}원</td>
          <td class:text-success={trade.pnl_pct > 0} class:text-error={trade.pnl_pct < 0}>
            {trade.pnl_pct > 0 ? '+' : ''}{trade.pnl_pct.toFixed(2)}%
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
```

## Modals

### Confirmation Modal

```svelte
<script>
  let { open = $bindable(false), title, message, onConfirm } = $props();
</script>

<dialog class="modal" class:modal-open={open}>
  <div class="modal-box">
    <h3 class="font-bold text-lg">{title}</h3>
    <p class="py-4">{message}</p>
    <div class="modal-action">
      <button class="btn" onclick={() => open = false}>취소</button>
      <button class="btn btn-primary" onclick={() => { onConfirm?.(); open = false; }}>
        확인
      </button>
    </div>
  </div>
  <form method="dialog" class="modal-backdrop">
    <button onclick={() => open = false}>close</button>
  </form>
</dialog>
```

## Loading States

### Skeleton Loader

```svelte
<script>
  let { loading, count = 3 } = $props();
</script>

{#if loading}
  <div class="space-y-4">
    {#each Array(count) as _}
      <div class="card bg-base-100 shadow">
        <div class="card-body">
          <div class="skeleton h-4 w-1/2"></div>
          <div class="skeleton h-4 w-3/4"></div>
          <div class="skeleton h-4 w-full"></div>
        </div>
      </div>
    {/each}
  </div>
{:else}
  <slot />
{/if}
```

### Button Loading

```svelte
<script>
  let { loading, onclick, children } = $props();
</script>

<button class="btn btn-primary" {onclick} disabled={loading}>
  {#if loading}
    <span class="loading loading-spinner loading-sm"></span>
  {/if}
  {@render children()}
</button>
```

## Alerts

```svelte
<script>
  let { type = 'info', message, dismissible = false, visible = $bindable(true) } = $props();
  
  let alertClass = $derived({
    'info': 'alert-info',
    'success': 'alert-success',
    'warning': 'alert-warning',
    'error': 'alert-error',
  }[type]);
</script>

{#if visible}
  <div class="alert {alertClass}">
    <span>{message}</span>
    {#if dismissible}
      <button class="btn btn-sm btn-ghost" onclick={() => visible = false}>✕</button>
    {/if}
  </div>
{/if}
```
