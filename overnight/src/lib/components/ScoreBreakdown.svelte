<script>
  let { signal, explanation = null } = $props();

  const items = $derived([
    { label: '테마', value: signal.scoreTheme ?? 0, max: 2 },
    { label: '뉴스', value: signal.scoreNews ?? 0, max: 3 },
    { label: '기술적', value: signal.scoreTechnical ?? 0, max: 3 },
    { label: '리스크', value: signal.scoreRisk ?? 0, max: 2 },
  ]);
</script>

<div class="card bg-base-100 shadow">
  <div class="card-body">
    <h3 class="card-title text-base">점수 분해</h3>
    <div class="space-y-3">
      {#each items as item}
        <div>
          <div class="mb-1 flex justify-between text-sm">
            <span>{item.label}</span>
            <span>{item.value.toFixed(1)} / {item.max}</span>
          </div>
          <progress class="progress progress-primary w-full" value={item.value} max={item.max}></progress>
        </div>
      {/each}
    </div>

    {#if explanation}
      <div class="mt-4 space-y-3 text-xs text-base-content/70">
        {#if explanation.themeReasons?.length}
          <div>
            <div class="mb-1 font-medium text-base-content">테마 근거</div>
            <ul class="list-disc pl-4">
              {#each explanation.themeReasons as reason}
                <li>{reason}</li>
              {/each}
            </ul>
          </div>
        {/if}
        {#if explanation.newsReasons?.length}
          <div>
            <div class="mb-1 font-medium text-base-content">뉴스 근거</div>
            <ul class="list-disc pl-4">
              {#each explanation.newsReasons as reason}
                <li>{reason}</li>
              {/each}
            </ul>
          </div>
        {/if}
        {#if explanation.technicalReasons?.length}
          <div>
            <div class="mb-1 font-medium text-base-content">기술적 근거</div>
            <ul class="list-disc pl-4">
              {#each explanation.technicalReasons as reason}
                <li>{reason}</li>
              {/each}
            </ul>
          </div>
        {/if}
        {#if explanation.riskReasons?.length}
          <div>
            <div class="mb-1 font-medium text-base-content">리스크 근거</div>
            <ul class="list-disc pl-4">
              {#each explanation.riskReasons as reason}
                <li>{reason}</li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
    {/if}
  </div>
</div>
