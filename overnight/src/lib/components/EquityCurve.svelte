<script>
  let { run } = $props();
</script>

<div class="card bg-base-100 shadow">
  <div class="card-body">
    <div class="flex items-center justify-between gap-3">
      <h3 class="card-title text-base">최근 실행 요약</h3>
      {#if run}
        <span class="badge badge-outline">Run #{run.id}</span>
      {/if}
    </div>

    {#if run}
      <div class="grid gap-3 md:grid-cols-3 text-sm">
        <div class="rounded-xl bg-base-200 p-4">
          <div class="text-base-content/60">누적 수익률</div>
          <div class:text-success={run.totalReturn >= 0} class:text-error={run.totalReturn < 0} class="mt-1 text-xl font-semibold">
            {run.totalReturn >= 0 ? '+' : ''}{run.totalReturn.toFixed(2)}%
          </div>
        </div>
        <div class="rounded-xl bg-base-200 p-4">
          <div class="text-base-content/60">평균 수익률</div>
          <div class="mt-1 text-xl font-semibold">{run.avgReturn.toFixed(2)}%</div>
        </div>
        <div class="rounded-xl bg-base-200 p-4">
          <div class="text-base-content/60">최대 낙폭</div>
          <div class="mt-1 text-xl font-semibold text-error">{run.maxDrawdown.toFixed(2)}%</div>
        </div>
      </div>
    {:else}
      <div class="rounded-xl border border-dashed border-base-300 p-8 text-sm text-base-content/60">
        백테스트를 실행하면 최근 실행 요약이 여기에 표시됩니다.
      </div>
    {/if}
  </div>
</div>
