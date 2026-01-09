<script>
  import Navbar from "./lib/components/Navbar.svelte";
  import CandleChart from "./lib/components/CandleChart.svelte";
  import { onMount } from "svelte";

  // Stock state
  let stockCode = "005930";
  let stockName = "삼성전자";
  let selectedDate = "";

  // Tab state using URL hash
  let currentTab = "dashboard";

  // Chart timeframe
  let chartTimeframe = "daily"; // daily, weekly, monthly, minute

  // Data arrays
  let candles = []; // Currently displayed candles (max 200)
  let allCandles = []; // All available candles for lazy loading
  const MAX_CANDLES = 200; // Max candles to show at once
  let news = [];
  let themes = [];
  let filings = [];

  // Fundamental data (from API)
  let fundamental = null;

  // Realtime state
  let activeRealtimeCategory = "themes";
  let realtimeTabs = { themes: {}, stocks: {} };
  let activeRealtimeKey = "";
  let realtimeData = null;
  let isRealtimeLoading = false;

  function initDate() {
    const now = new Date();
    now.setDate(now.getDate() - 1);
    now.setHours(15, 20, 0, 0);
    selectedDate = now.toISOString().slice(0, 16);
  }

  // Use hash-based navigation
  function handleHashChange() {
    const hash = window.location.hash.replace("#", "") || "dashboard";
    currentTab = hash;
  }

  async function fetchData() {
    if (!selectedDate) return;
    console.log("Fetching data for", stockCode);

    // Fetch News
    try {
      const res = await fetch(`/news/articles?limit=5`);
      if (res.ok) {
        const data = await res.json();
        news = data.articles || [];
      }
    } catch (e) {
      console.warn("News fetch failed", e);
    }

    // Fetch Themes
    try {
      const res = await fetch(`/judal/stocks/${stockCode}/themes`);
      if (res.ok) {
        const data = await res.json();
        themes = data.themes || [];
      }
    } catch (e) {
      console.warn("Themes fetch failed", e);
    }

    // Fetch DART
    try {
      const res = await fetch(`/dart/filings?stock_code=${stockCode}&limit=5`);
      if (res.ok) {
        const data = await res.json();
        filings = data.data || [];
      }
    } catch (e) {
      console.warn("DART fetch failed", e);
    }

    // Fetch Candles based on timeframe
    await fetchCandles();

    // Fetch Fundamental
    await fetchFundamental();
  }

  async function fetchCandles() {
    console.log("fetchCandles called, timeframe:", chartTimeframe);
    try {
      // Unified endpoint for KR/US
      // Map timeframe to backend/API expected format
      // daily -> D, weekly -> W, monthly -> M
      // minute -> 1 (1 minute)
      let tfParam = "D";
      if (chartTimeframe === "weekly") tfParam = "W";
      if (chartTimeframe === "monthly") tfParam = "M";
      if (chartTimeframe === "minute") tfParam = "1";

      const query = new URLSearchParams({
        market: "KR", // Currently hardcoded for KR as per user context (dashboard is for single stock)
        symbol: stockCode,
        timeframe: tfParam,
        limit: "500", // Fetch plenty for agg
      });

      // Calculate date range if needed? Backend handles defaults.
      // Kiwoom API handles start/end.

      const res = await fetch(`/candle/stocks?${query.toString()}`);
      console.log("Unified API response status:", res.status);

      if (res.ok) {
        const data = await res.json();
        console.log("Candles received:", data.count, "candles");
        let rawCandles = data.candles || [];

        // rawCandles now have 'ts' (unix seconds) from backend normalization
        // Add 'date' string property for aggregation logic if missing?
        // Aggregation functions expect c.date string or need update.
        // Let's update rawCandles to have Date object for easier handling.

        rawCandles = rawCandles.map((c) => ({
          ...c,
          dateObj: new Date(c.ts * 1000), // Create Date object
          date: new Date(c.ts * 1000).toISOString().slice(0, 10), // YYYY-MM-DD for agg
        }));

        if (chartTimeframe === "weekly") {
          rawCandles = aggregateToWeekly(rawCandles);
        } else if (chartTimeframe === "monthly") {
          rawCandles = aggregateToMonthly(rawCandles);
        }

        // Store all candles for lazy loading
        allCandles = rawCandles
          .map((c) => ({
            time: c.ts || new Date(c.date).getTime() / 1000, // Ensure time is seconds
            open: c.open,
            high: c.high,
            low: c.low,
            close: c.close,
          }))
          .filter((c) => !isNaN(c.time))
          .sort((a, b) => a.time - b.time);

        // Display only last MAX_CANDLES initially
        candles = allCandles.slice(-MAX_CANDLES);
        console.log(
          "Processed:",
          allCandles.length,
          "total,",
          candles.length,
          "displayed",
        );
      }
    } catch (e) {
      console.warn("Candle fetch failed", e);
    }
  }

  // Handler for when user scrolls/zooms to load more data
  let loadMoreTimeout = null;
  let lastLoadTime = 0;

  function handleVisibleRangeChange(range) {
    // Debounce and rate limit loading
    if (!range || !range.from || allCandles.length <= MAX_CANDLES) return;
    if (candles.length >= allCandles.length) return; // Already showing all data

    const now = Date.now();
    if (now - lastLoadTime < 500) return; // Rate limit: max once per 500ms

    clearTimeout(loadMoreTimeout);
    loadMoreTimeout = setTimeout(() => {
      const firstVisibleTime = range.from;
      const firstCandleTime = candles[0]?.time;

      // Only load more if viewing within 10 candles of the historical start
      if (firstCandleTime && firstVisibleTime <= firstCandleTime + 10 * 86400) {
        const currentStartIdx = allCandles.findIndex(
          (c) => c.time === candles[0].time,
        );
        if (currentStartIdx > 0) {
          lastLoadTime = Date.now();
          const newStartIdx = Math.max(0, currentStartIdx - 100);
          const newEndIdx = currentStartIdx + candles.length;
          candles = allCandles.slice(
            newStartIdx,
            Math.min(newEndIdx, allCandles.length),
          );
          console.log(
            "Loaded more:",
            candles.length,
            "/",
            allCandles.length,
            "candles",
          );
        }
      }
    }, 200); // 200ms debounce
  }

  function aggregateToWeekly(dailyCandles) {
    const weeks = {};
    for (const c of dailyCandles) {
      const d = new Date(c.date);
      const weekStart = new Date(d);
      weekStart.setDate(d.getDate() - d.getDay()); // Start of week (Sunday)
      const key = weekStart.toISOString().slice(0, 10);

      if (!weeks[key]) {
        weeks[key] = {
          date: key,
          open: c.open,
          high: c.high,
          low: c.low,
          close: c.close,
        };
      } else {
        weeks[key].high = Math.max(weeks[key].high, c.high);
        weeks[key].low = Math.min(weeks[key].low, c.low);
        weeks[key].close = c.close; // Last close
      }
    }
    return Object.values(weeks).sort((a, b) => a.date.localeCompare(b.date));
  }

  function aggregateToMonthly(dailyCandles) {
    const months = {};
    for (const c of dailyCandles) {
      const key = c.date.slice(0, 7); // YYYY-MM

      if (!months[key]) {
        months[key] = {
          date: key + "-01",
          open: c.open,
          high: c.high,
          low: c.low,
          close: c.close,
        };
      } else {
        months[key].high = Math.max(months[key].high, c.high);
        months[key].low = Math.min(months[key].low, c.low);
        months[key].close = c.close;
      }
    }
    return Object.values(months).sort((a, b) => a.date.localeCompare(b.date));
  }

  async function fetchFundamental() {
    try {
      const res = await fetch(`/candle/fundamental/${stockCode}`);
      if (res.ok) {
        const data = await res.json();
        fundamental = data.data;
      }
    } catch (e) {
      console.warn("Fundamental fetch failed", e);
      fundamental = null;
    }
  }

  async function fetchRealtimeTabs() {
    try {
      const res = await fetch("/judal/realtime/tabs");
      if (res.ok) {
        const data = await res.json();
        realtimeTabs = {
          themes: data.theme_tabs || {},
          stocks: data.stock_tabs || {},
        };
        if (Object.keys(realtimeTabs.themes).length > 0) {
          activeRealtimeKey = "rising";
          fetchRealtimeData();
        }
      }
    } catch (e) {
      console.warn("Realtime tabs failed", e);
    }
  }

  async function fetchRealtimeData() {
    if (!activeRealtimeKey) return;
    isRealtimeLoading = true;
    realtimeData = null;
    try {
      const endpoint =
        activeRealtimeCategory === "themes" ? "themes" : "stocks";
      const res = await fetch(
        `/judal/realtime/${endpoint}/${activeRealtimeKey}`,
      );
      if (res.ok) realtimeData = await res.json();
    } catch (e) {
      console.error("Realtime fetch failed", e);
    }
    isRealtimeLoading = false;
  }

  async function switchTimeframe(tf) {
    chartTimeframe = tf;
    await fetchCandles();
  }

  onMount(() => {
    initDate();
    fetchRealtimeTabs();
    window.addEventListener("hashchange", handleHashChange);
    handleHashChange();
    return () => window.removeEventListener("hashchange", handleHashChange);
  });

  $: if (selectedDate) fetchData();
</script>

<div
  class="min-h-screen bg-gradient-to-br from-base-200 via-base-100 to-base-200"
>
  <Navbar title="Market Mosaic" bind:selectedDate />

  <main class="container mx-auto px-4 py-6 max-w-7xl">
    <!-- Stock Header -->
    <div class="card bg-base-100 shadow-xl mb-6">
      <div class="card-body p-6">
        <div
          class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4"
        >
          <div class="flex items-center gap-4">
            <div class="avatar placeholder">
              <div class="bg-primary text-primary-content rounded-lg w-14 h-14">
                <span class="text-xl font-bold">{stockName[0]}</span>
              </div>
            </div>
            <div>
              <h1 class="text-2xl font-bold">{stockName}</h1>
              <div class="flex gap-2 mt-1">
                <span class="badge badge-primary font-mono">{stockCode}</span>
                <span class="badge badge-outline">KOSPI</span>
              </div>
            </div>
          </div>

          <!-- Main Tabs -->
          <div class="tabs tabs-boxed bg-base-200 p-1">
            <a
              href="#dashboard"
              class="tab"
              class:tab-active={currentTab === "dashboard"}>Dashboard</a
            >
            <a
              href="#themes"
              class="tab"
              class:tab-active={currentTab === "themes"}>Themes</a
            >
            <a
              href="#fundamental"
              class="tab"
              class:tab-active={currentTab === "fundamental"}>Fundamental</a
            >
          </div>
        </div>
      </div>
    </div>

    <!-- Dashboard Tab -->
    {#if currentTab === "dashboard"}
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div class="lg:col-span-2 card bg-base-100 shadow-xl">
          <div class="card-body">
            <div class="flex justify-between items-center">
              <h2 class="card-title">
                Chart ({candles.length}/{allCandles.length} bars)
              </h2>
              <!-- Timeframe Tabs -->
              <div class="tabs tabs-boxed tabs-sm">
                <button
                  class="tab"
                  class:tab-active={chartTimeframe === "daily"}
                  on:click={() => switchTimeframe("daily")}>일봉</button
                >
                <button
                  class="tab"
                  class:tab-active={chartTimeframe === "weekly"}
                  on:click={() => switchTimeframe("weekly")}>주봉</button
                >
                <button
                  class="tab"
                  class:tab-active={chartTimeframe === "monthly"}
                  on:click={() => switchTimeframe("monthly")}>월봉</button
                >
                <button
                  class="tab"
                  class:tab-active={chartTimeframe === "minute"}
                  on:click={() => switchTimeframe("minute")}>분봉</button
                >
              </div>
            </div>
            <div class="h-[350px]">
              {#key chartTimeframe + "-" + candles.length}
                {#if candles.length > 0}
                  <CandleChart
                    data={candles}
                    height={350}
                    onVisibleRangeChange={handleVisibleRangeChange}
                  />
                {:else}
                  <div
                    class="flex items-center justify-center h-full text-base-content/50"
                  >
                    Loading chart...
                  </div>
                {/if}
              {/key}
            </div>
          </div>
        </div>

        <div class="card bg-base-100 shadow-xl">
          <div class="card-body">
            <h2 class="card-title">News ({news.length})</h2>
            <div class="space-y-2 max-h-[350px] overflow-y-auto">
              {#each news as item}
                <a
                  href={item.url}
                  target="_blank"
                  class="block p-2 bg-base-200 rounded hover:bg-base-300"
                >
                  <div class="font-medium text-sm line-clamp-2">
                    {item.title}
                  </div>
                  <div class="text-xs opacity-60 mt-1">
                    {item.published_at?.slice(0, 10)} • {item.source ||
                      "Unknown"}
                  </div>
                </a>
              {:else}
                <div class="text-base-content/50 text-center py-8">
                  No news available
                </div>
              {/each}
            </div>
          </div>
        </div>

        <div class="card bg-base-100 shadow-xl">
          <div class="card-body">
            <h2 class="card-title">Related Themes</h2>
            <div class="flex flex-wrap gap-2">
              {#each themes as t}
                <span class="badge badge-primary badge-outline">{t.name}</span>
              {:else}
                <span class="text-base-content/50">No themes found</span>
              {/each}
            </div>
          </div>
        </div>

        <div class="lg:col-span-2 card bg-base-100 shadow-xl">
          <div class="card-body">
            <h2 class="card-title">DART Filings</h2>
            <div class="overflow-x-auto">
              <table class="table table-zebra">
                <thead
                  ><tr><th>Date</th><th>Report</th><th>Filer</th></tr></thead
                >
                <tbody>
                  {#each filings as f}
                    <tr
                      ><td>{f.rcept_dt}</td><td>{f.report_nm}</td><td
                        >{f.flr_nm}</td
                      ></tr
                    >
                  {:else}
                    <tr
                      ><td colspan="3" class="text-center text-base-content/50"
                        >No filings</td
                      ></tr
                    >
                  {/each}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>

      <!-- Themes Tab -->
    {:else if currentTab === "themes"}
      <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <div class="flex justify-between items-center mb-4">
            <h2 class="card-title text-2xl">Market Themes & Trends</h2>
            <div class="join">
              <button
                class="join-item btn"
                class:btn-primary={activeRealtimeCategory === "themes"}
                on:click={() => {
                  activeRealtimeCategory = "themes";
                  activeRealtimeKey = Object.keys(realtimeTabs.themes)[0] || "";
                  fetchRealtimeData();
                }}>Themes</button
              >
              <button
                class="join-item btn"
                class:btn-primary={activeRealtimeCategory === "stocks"}
                on:click={() => {
                  activeRealtimeCategory = "stocks";
                  activeRealtimeKey = Object.keys(realtimeTabs.stocks)[0] || "";
                  fetchRealtimeData();
                }}>Stocks</button
              >
            </div>
          </div>

          <div class="tabs tabs-boxed mb-4 overflow-x-auto">
            {#each Object.entries(realtimeTabs[activeRealtimeCategory] || {}) as [key, label]}
              <button
                class="tab whitespace-nowrap"
                class:tab-active={activeRealtimeKey === key}
                on:click={() => {
                  activeRealtimeKey = key;
                  fetchRealtimeData();
                }}>{label}</button
              >
            {/each}
          </div>

          {#if isRealtimeLoading}
            <div class="flex justify-center p-12">
              <span class="loading loading-spinner loading-lg"></span>
            </div>
          {:else if realtimeData?.items}
            <div class="overflow-x-auto">
              <table class="table table-zebra">
                <thead>
                  <tr>
                    {#if activeRealtimeCategory === "themes"}
                      <th>Index</th><th>Name</th><th>Details</th>
                    {:else}
                      <th>Code</th><th>Name</th><th>Price</th><th>Change</th>
                    {/if}
                  </tr>
                </thead>
                <tbody>
                  {#each realtimeData.items as item}
                    <tr class="hover">
                      {#if activeRealtimeCategory === "themes"}
                        <td class="font-mono text-sm">{item.theme_idx}</td>
                        <td class="font-bold">{item.name}</td>
                        <td
                          >{#each item.values || [] as v}<span
                              class="badge badge-sm mr-1">{v}</span
                            >{/each}</td
                        >
                      {:else}
                        <td class="font-mono">{item.code}</td>
                        <td
                          class="font-bold cursor-pointer hover:text-primary"
                          on:click={() => {
                            stockCode = item.code;
                            stockName = item.name;
                            window.location.hash = "dashboard";
                            fetchData();
                          }}>{item.name}</td
                        >
                        <td>{item.current_price?.toLocaleString() || "-"}</td>
                        <td
                          class:text-error={item.change_rate > 0}
                          class:text-info={item.change_rate < 0}
                          >{item.change_rate > 0
                            ? "+"
                            : ""}{item.change_rate}%</td
                        >
                      {/if}
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {:else}
            <div class="text-center py-12 text-base-content/50">
              Select a category
            </div>
          {/if}
        </div>
      </div>

      <!-- Fundamental Tab -->
    {:else if currentTab === "fundamental"}
      <div class="card bg-base-100 shadow-xl">
        <div class="card-body">
          <h2 class="card-title text-2xl mb-6">
            Fundamental Analysis - {stockName}
          </h2>

          {#if fundamental}
            <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
              <div class="stat bg-base-200 rounded-box">
                <div class="stat-title">PER</div>
                <div class="stat-value text-primary">
                  {fundamental.PER?.toFixed(2) || "-"}
                </div>
                <div class="stat-desc">Price/Earnings</div>
              </div>
              <div class="stat bg-base-200 rounded-box">
                <div class="stat-title">PBR</div>
                <div class="stat-value text-secondary">
                  {fundamental.PBR?.toFixed(2) || "-"}
                </div>
                <div class="stat-desc">Price/Book</div>
              </div>
              <div class="stat bg-base-200 rounded-box">
                <div class="stat-title">EPS</div>
                <div class="stat-value text-accent">
                  {fundamental.EPS?.toLocaleString() || "-"}
                </div>
                <div class="stat-desc">원</div>
              </div>
              <div class="stat bg-base-200 rounded-box">
                <div class="stat-title">BPS</div>
                <div class="stat-value text-info">
                  {fundamental.BPS?.toLocaleString() || "-"}
                </div>
                <div class="stat-desc">원</div>
              </div>
              {#if fundamental.DIV}
                <div class="stat bg-base-200 rounded-box">
                  <div class="stat-title">배당수익률</div>
                  <div class="stat-value text-success">
                    {fundamental.DIV?.toFixed(2)}%
                  </div>
                  <div class="stat-desc">Dividend Yield</div>
                </div>
              {/if}
              {#if fundamental.DPS}
                <div class="stat bg-base-200 rounded-box">
                  <div class="stat-title">DPS</div>
                  <div class="stat-value text-warning">
                    {fundamental.DPS?.toLocaleString()}
                  </div>
                  <div class="stat-desc">원/주</div>
                </div>
              {/if}
            </div>
            <div class="text-right text-xs opacity-50 mt-4">
              기준일: {fundamental.date}
            </div>
          {:else}
            <div class="alert alert-warning">
              <span
                >Fundamental 데이터를 가져오려면 .env에 KIWOOM_REST_API_URL을
                설정하세요.</span
              >
            </div>
            <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mt-4 opacity-50">
              <div class="stat bg-base-200 rounded-box">
                <div class="stat-title">PER</div>
                <div class="stat-value text-primary">-</div>
              </div>
              <div class="stat bg-base-200 rounded-box">
                <div class="stat-title">PBR</div>
                <div class="stat-value text-secondary">-</div>
              </div>
              <div class="stat bg-base-200 rounded-box">
                <div class="stat-title">EPS</div>
                <div class="stat-value text-accent">-</div>
              </div>
              <div class="stat bg-base-200 rounded-box">
                <div class="stat-title">BPS</div>
                <div class="stat-value text-info">-</div>
              </div>
            </div>
          {/if}
        </div>
      </div>
    {/if}
  </main>
</div>
