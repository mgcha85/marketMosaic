<script>
  let { candidates = [], selectedCode = '', onSelect = null } = $props();

  function statusBadge(candidate) {
    if (candidate.wasTraded) return 'badge-success';
    if (candidate.passedScore) return 'badge-primary';
    return 'badge-warning';
  }

  function statusLabel(candidate) {
    if (candidate.wasTraded) return '실매매';
    if (candidate.passedScore) return '통과';
    return '탈락';
  }
</script>

<div class="card bg-base-100 shadow">
  <div class="card-body">
    <div class="flex items-center justify-between gap-3">
      <h3 class="card-title text-base">일자별 후보 판정표</h3>
      <span class="badge badge-outline">{candidates.length}개 후보</span>
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
            <th>차트</th>
            <th>리스크</th>
            <th>판정 이유</th>
          </tr>
        </thead>
        <tbody>
          {#if candidates.length === 0}
            <tr>
              <td colspan="9" class="text-center text-base-content/60">선택한 날짜에 대한 평가 후보가 없습니다.</td>
            </tr>
          {:else}
            {#each candidates as candidate}
              <tr
                class={`cursor-pointer ${candidate.code === selectedCode ? 'bg-primary/5' : ''}`}
                onclick={() => onSelect?.(candidate)}
              >
                <td><span class={`badge ${statusBadge(candidate)}`}>{statusLabel(candidate)}</span></td>
                <td>
                  <div class="font-medium">{candidate.name}</div>
                  <div class="text-xs text-base-content/60">{candidate.code}</div>
                </td>
                <td>{candidate.themeName ?? '-'}</td>
                <td>{candidate.score.toFixed(1)}</td>
                <td>{candidate.scoreTheme.toFixed(1)}</td>
                <td>{candidate.scoreNews.toFixed(1)}</td>
                <td>{candidate.scoreTechnical.toFixed(1)}</td>
                <td>{candidate.scoreRisk.toFixed(1)}</td>
                <td class="max-w-sm text-xs text-base-content/70">
                  {#if candidate.passedScore}
                    <div>{candidate.structuredReasoning?.reasoning ?? candidate.reasoning}</div>
                    {#if candidate.pnlPct !== null}
                      <div class="mt-1 text-[11px] text-base-content/50">
                        결과: {candidate.pnlPct >= 0 ? '+' : ''}{candidate.pnlPct?.toFixed?.(2) ?? '-'}%
                      </div>
                    {/if}
                  {:else}
                    <div>{candidate.rejectionReasons?.[0] ?? '기준 미달'}</div>
                    {#if candidate.rejectionReasons?.[1]}
                      <div class="mt-1 text-[11px] text-base-content/50">{candidate.rejectionReasons[1]}</div>
                    {/if}
                    {#if candidate.pnlPct !== null}
                      <div class="mt-1 text-[11px] text-base-content/50">
                        참고 결과: {candidate.pnlPct >= 0 ? '+' : ''}{candidate.pnlPct?.toFixed?.(2) ?? '-'}%
                      </div>
                    {/if}
                  {/if}
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>
  </div>
</div>
