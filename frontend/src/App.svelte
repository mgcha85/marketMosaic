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

  // Data arrays
  let candles = [];
  let news = [];
  let themes = [];
  let filings = [];

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
    console.log("Hash changed to:", hash, "currentTab:", currentTab);
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

    // Fetch Candles
    try {
      const ts = Math.floor(new Date(selectedDate).getTime() / 1000);
      const res = await fetch(
        `/candle/stocks?market=KR&symbol=${stockCode}&timeframe=1d&limit=100&to=${ts}`,
      );
      if (res.ok) {
        const data = await res.json();
        candles = (data.candles || [])
          .map((c) => ({
            time: c.ts,
            open: c.open,
            high: c.high,
            low: c.low,
            close: c.close,
          }))
          .sort((a, b) => a.time - b.time);
      }
    } catch (e) {
      console.warn("Candle fetch failed", e);
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

  onMount(() => {
    initDate();
    fetchRealtimeTabs();

    // Listen for hash changes
    window.addEventListener("hashchange", handleHashChange);
    handleHashChange(); // Initial check

    return () => {
      window.removeEventListener("hashchange", handleHashChange);
    };
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

          <!-- Tabs using anchor links -->
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
        <div class="text-xs mt-2 opacity-50">Tab: {currentTab}</div>
      </div>
    </div>

    <!-- Dashboard Tab -->
    {#if currentTab === "dashboard"}
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div class="lg:col-span-2 card bg-base-100 shadow-xl">
          <div class="card-body">
            <h2 class="card-title">Chart ({candles.length} bars)</h2>
            <div class="h-[350px]">
              {#if candles.length > 0}
                <CandleChart data={candles} height={350} />
              {:else}
                <div
                  class="flex items-center justify-center h-full text-base-content/50"
                >
                  Loading chart...
                </div>
              {/if}
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
            <div class="text-right text-xs opacity-50 mt-4">
              Crawled: {realtimeData.crawled_at}
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
          <h2 class="card-title text-2xl mb-6">Fundamental Analysis</h2>
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div class="stat bg-base-200 rounded-box">
              <div class="stat-title">PER</div>
              <div class="stat-value text-primary">12.5</div>
              <div class="stat-desc">Sector: 15.0</div>
            </div>
            <div class="stat bg-base-200 rounded-box">
              <div class="stat-title">PBR</div>
              <div class="stat-value text-secondary">1.3</div>
              <div class="stat-desc">Low valuation</div>
            </div>
            <div class="stat bg-base-200 rounded-box">
              <div class="stat-title">ROE</div>
              <div class="stat-value text-accent">10.2%</div>
              <div class="stat-desc">↘ 2% YoY</div>
            </div>
            <div class="stat bg-base-200 rounded-box">
              <div class="stat-title">Market Cap</div>
              <div class="stat-value text-info text-2xl">450T</div>
              <div class="stat-desc">KRW</div>
            </div>
          </div>
        </div>
      </div>
    {/if}
  </main>
</div>
