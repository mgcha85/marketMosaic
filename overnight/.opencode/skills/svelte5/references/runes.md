# Svelte 5 Runes Reference

## $state - Reactive State

### Basic Usage

```svelte
<script>
  // Primitive state
  let count = $state(0);
  let name = $state('');
  let isOpen = $state(false);
  
  // Object state (deeply reactive)
  let signal = $state({
    code: '005930',
    name: '삼성전자',
    score: 8.5
  });
  
  // Array state (deeply reactive)
  let signals = $state([]);
</script>

<button onclick={() => count++}>
  Clicked {count} times
</button>

<input bind:value={name} />

<!-- Object mutation is tracked -->
<button onclick={() => signal.score += 0.1}>
  Increase Score
</button>

<!-- Array mutation is tracked -->
<button onclick={() => signals.push(newSignal)}>
  Add Signal
</button>
```

### Class State

```svelte
<script>
  class SignalStore {
    signals = $state([]);
    selectedCode = $state(null);
    
    get selected() {
      return this.signals.find(s => s.code === this.selectedCode);
    }
    
    add(signal) {
      this.signals.push(signal);
    }
    
    remove(code) {
      this.signals = this.signals.filter(s => s.code !== code);
    }
  }
  
  const store = new SignalStore();
</script>
```

## $derived - Computed Values

### Basic Derived

```svelte
<script>
  let signals = $state([]);
  let minScore = $state(8.0);
  
  // Automatically recalculates when dependencies change
  let filteredSignals = $derived(
    signals.filter(s => s.score >= minScore)
  );
  
  let count = $derived(filteredSignals.length);
  
  let avgScore = $derived(
    signals.length > 0
      ? signals.reduce((sum, s) => sum + s.score, 0) / signals.length
      : 0
  );
</script>

<p>Showing {count} signals with score >= {minScore}</p>
```

### Derived with Complex Logic

```svelte
<script>
  let trades = $state([]);
  
  let stats = $derived(() => {
    if (trades.length === 0) {
      return { winRate: 0, avgReturn: 0, totalReturn: 0 };
    }
    
    const wins = trades.filter(t => t.pnl > 0).length;
    const winRate = (wins / trades.length) * 100;
    const avgReturn = trades.reduce((sum, t) => sum + t.pnl, 0) / trades.length;
    const totalReturn = trades.reduce((acc, t) => acc * (1 + t.pnl / 100), 1) - 1;
    
    return {
      winRate: winRate.toFixed(1),
      avgReturn: avgReturn.toFixed(2),
      totalReturn: (totalReturn * 100).toFixed(2)
    };
  });
</script>
```

## $effect - Side Effects

### Basic Effect

```svelte
<script>
  let selectedDate = $state('2026-04-11');
  let signals = $state([]);
  
  // Runs when selectedDate changes
  $effect(() => {
    console.log(`Fetching signals for ${selectedDate}`);
    fetchSignals(selectedDate);
  });
  
  async function fetchSignals(date) {
    const res = await fetch(`/api/signals?date=${date}`);
    signals = await res.json();
  }
</script>
```

### Effect with Cleanup

```svelte
<script>
  let chartContainer;
  let chart = $state(null);
  
  $effect(() => {
    // Create chart
    chart = createChart(chartContainer, { width: 600, height: 400 });
    
    // Cleanup function (runs before re-running or on destroy)
    return () => {
      chart.remove();
    };
  });
</script>

<div bind:this={chartContainer}></div>
```

### Async in Effect

```svelte
<script>
  let code = $state('005930');
  let history = $state([]);
  let loading = $state(false);
  
  // IIFE pattern for async
  $effect(() => {
    const currentCode = code; // Capture for closure
    
    (async () => {
      loading = true;
      try {
        const res = await fetch(`/api/stocks/${currentCode}/history`);
        // Only update if code hasn't changed
        if (currentCode === code) {
          history = await res.json();
        }
      } finally {
        if (currentCode === code) {
          loading = false;
        }
      }
    })();
  });
</script>
```

## $props - Component Props

### Basic Props

```svelte
<!-- SignalCard.svelte -->
<script>
  let { signal, onSelect, showDetails = false } = $props();
</script>

<div class="card" onclick={() => onSelect?.(signal)}>
  <h3>{signal.name}</h3>
  {#if showDetails}
    <p>Score: {signal.score}</p>
  {/if}
</div>
```

### Props with TypeScript

```svelte
<script lang="ts">
  interface Signal {
    code: string;
    name: string;
    score: number;
  }
  
  interface Props {
    signal: Signal;
    onSelect?: (signal: Signal) => void;
    showDetails?: boolean;
  }
  
  let { signal, onSelect, showDetails = false }: Props = $props();
</script>
```

### Spreading Props

```svelte
<script>
  let { class: className, ...rest } = $props();
</script>

<div class="card {className}" {...rest}>
  <slot />
</div>
```

## $bindable - Two-way Binding Props

```svelte
<!-- RangeSlider.svelte -->
<script>
  let { value = $bindable(0), min = 0, max = 100 } = $props();
</script>

<input type="range" bind:value {min} {max} />
<span>{value}</span>

<!-- Usage -->
<script>
  let score = $state(8.0);
</script>

<RangeSlider bind:value={score} min={0} max={10} />
```

## Snippets (Replacing Slots)

### Basic Snippet

```svelte
<!-- Card.svelte -->
<script>
  let { header, children } = $props();
</script>

<div class="card">
  {#if header}
    <div class="card-header">
      {@render header()}
    </div>
  {/if}
  <div class="card-body">
    {@render children()}
  </div>
</div>

<!-- Usage -->
<Card>
  {#snippet header()}
    <h2>Signal Details</h2>
  {/snippet}
  
  <p>Card content here</p>
</Card>
```

### Snippet with Parameters

```svelte
<!-- List.svelte -->
<script>
  let { items, renderItem } = $props();
</script>

<ul>
  {#each items as item, index}
    <li>
      {@render renderItem(item, index)}
    </li>
  {/each}
</ul>

<!-- Usage -->
<List items={signals}>
  {#snippet renderItem(signal, i)}
    <span>{i + 1}. {signal.name} - {signal.score}</span>
  {/snippet}
</List>
```

## Migration from Svelte 4

| Svelte 4 | Svelte 5 |
|----------|----------|
| `export let prop` | `let { prop } = $props()` |
| `$: derived = x * 2` | `let derived = $derived(x * 2)` |
| `$: { sideEffect() }` | `$effect(() => { sideEffect() })` |
| `<slot />` | `{@render children()}` |
| `<slot name="x" />` | `{@render x?.()}` |
| `let:item` | `{#snippet item(data)}` |
