<script>
  let { signals = [], selectedCode = '', onSelect = null } = $props();
</script>

<div class="card bg-base-100 shadow">
  <div class="card-body">
    <div class="flex items-center justify-between gap-3">
      <h3 class="card-title text-base">일자별 후보 요약</h3>
      <span class="badge badge-outline">{signals.length}개 후보</span>
    </div>

    <div class="overflow-x-auto">
      <table class="table table-sm">
        <thead>
          <tr>
            <th>상태</th>
            <th>종목</th>
            <th>테마</th>
            <th>총점</th>
            <th>Judal</th>
            <th>뉴스</th>
            <th>캔들/차트</th>
            <th>리스크</th>
            <th>핵심 의견</th>
          </tr>
        </thead>
        <tbody>
          {#if signals.length === 0}
            <tr>
              <td colspan="9" class="text-center text-base-content/60">선택한 날짜에 저장된 후보가 없습니다.</td>
            </tr>
          {:else}
            {#each signals as signal}
              <tr
                class={`cursor-pointer ${signal.code === selectedCode ? 'bg-primary/5' : ''}`}
                onclick={() => onSelect?.(signal)}
              >
                <td>
                  <span class={`badge ${signal.status === 'pass' ? 'badge-success' : 'badge-warning'}`}>
                    {signal.status === 'pass' ? '통과' : '탈락'}
                  </span>
                </td>
                <td>
                  <div class="font-medium">{signal.name}</div>
                  <div class="text-xs text-base-content/60">{signal.code}</div>
                </td>
                <td>{signal.themeName ?? '-'}</td>
                <td>
                  <span class="badge badge-primary badge-outline">{signal.score.toFixed(1)}</span>
                </td>
                <td>{signal.scoreTheme?.toFixed?.(1) ?? '-'}</td>
                <td>{signal.scoreNews?.toFixed?.(1) ?? '-'}</td>
                <td>{signal.scoreTechnical?.toFixed?.(1) ?? '-'}</td>
                <td>{signal.scoreRisk?.toFixed?.(1) ?? '-'}</td>
                <td class="max-w-sm text-xs text-base-content/70">{signal.structuredReasoning?.reasoning ?? signal.reasoning}</td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>
  </div>
</div>
