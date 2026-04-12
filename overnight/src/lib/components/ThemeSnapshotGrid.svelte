<script>
  let { themes = [] } = $props();
</script>

<div class="card bg-base-100 shadow">
  <div class="card-body">
    <div class="flex items-center justify-between gap-3">
      <h3 class="card-title text-base">그날 강했던 테마</h3>
      <span class="badge badge-outline">{themes.length}개 섹션</span>
    </div>

    <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {#if themes.length === 0}
        <div class="rounded-2xl border border-dashed border-base-300 p-6 text-sm text-base-content/60">
          선택한 날짜의 테마 스냅샷이 없습니다.
        </div>
      {:else}
        {#each themes as theme}
          <div class="rounded-2xl border border-base-300 bg-base-50 p-4">
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="text-sm font-semibold">{theme.themeName}</div>
                <div class="text-xs text-base-content/50">
                  후보 {theme.candidateCount}개 · 통과 {theme.passCount}개
                </div>
              </div>
              <span class="badge badge-outline">평균 {theme.averageScore.toFixed(1)}</span>
            </div>

            {#if theme.leadStock}
              <div class="mt-3 text-xs text-base-content/70">대표 종목: {theme.leadStock}</div>
            {/if}
            {#if theme.breadthNote}
              <div class="mt-1 text-xs text-base-content/50">{theme.breadthNote}</div>
            {/if}
          </div>
        {/each}
      {/if}
    </div>
  </div>
</div>
