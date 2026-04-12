<script>
  let { stats } = $props();
</script>

<div class="stats stats-vertical xl:stats-horizontal shadow w-full bg-base-100">
  <div class="stat">
    <div class="stat-title">백테스트 실행 수</div>
    <div class="stat-value text-primary">{stats.totalRuns}</div>
  </div>
  <div class="stat">
    <div class="stat-title">총 거래</div>
    <div class="stat-value">{stats.totalTrades}</div>
  </div>
  <div class="stat">
    <div class="stat-title">평균 승률</div>
    <div class="stat-value text-secondary">{stats.avgWinRate.toFixed(1)}%</div>
  </div>
  <div class="stat">
    <div class="stat-title">평균 수익률</div>
    <div class="stat-value">{stats.avgReturn.toFixed(2)}%</div>
  </div>
</div>
