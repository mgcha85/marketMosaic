<script>
  let { trades = [] } = $props();

  let explanationFilter = $state('');
  let positiveOnly = $state(false);
  let minScore = $state('');
  let themeFilter = $state('');
  let cautionOnly = $state(false);
  let candleStrongOnly = $state(false);

  const filteredTrades = $derived(
    trades.filter((trade) => {
      if (positiveOnly && trade.pnlPct < 0) return false;
      if (minScore !== '' && (trade.score ?? 0) < Number(minScore)) return false;
      if (themeFilter && !(trade.themeName ?? '').toLowerCase().includes(themeFilter.toLowerCase())) {
        return false;
      }
      if (
        explanationFilter &&
        !(trade.explanationSummary ?? '').toLowerCase().includes(explanationFilter.toLowerCase())
      ) {
        return false;
      }
      if (
        cautionOnly &&
        !(trade.explanation?.riskReasons ?? []).some((item) => item.toLowerCase().includes('과열') || item.toLowerCase().includes('주의'))
      ) {
        return false;
      }
      if (
        candleStrongOnly &&
        !(trade.explanation?.technicalReasons ?? []).some((item) =>
          item.includes('VWAP') || item.includes('거래량 스파이크') || item.includes('체결건수 스파이크')
        )
      ) {
        return false;
      }
      return true;
    })
  );
</script>

<div class="card bg-base-100 shadow">
  <div class="card-body">
    <h3 class="card-title text-base">거래 내역</h3>
    <div class="grid gap-3 md:grid-cols-3 xl:grid-cols-6">
      <label class="form-control gap-1">
        <span class="label-text text-xs">설명 필터</span>
        <input class="input input-bordered input-sm" bind:value={explanationFilter} placeholder="정책, MA5, 과열..." />
      </label>
      <label class="form-control gap-1">
        <span class="label-text text-xs">최소 점수</span>
        <input class="input input-bordered input-sm" bind:value={minScore} placeholder="8.0" />
      </label>
      <label class="form-control gap-1">
        <span class="label-text text-xs">테마 필터</span>
        <input class="input input-bordered input-sm" bind:value={themeFilter} placeholder="반도체, 로봇..." />
      </label>
      <label class="label cursor-pointer justify-start gap-2 pt-6">
        <input type="checkbox" class="checkbox checkbox-sm" bind:checked={positiveOnly} />
        <span class="label-text text-xs">수익 거래만 보기</span>
      </label>
      <label class="label cursor-pointer justify-start gap-2 pt-6">
        <input type="checkbox" class="checkbox checkbox-sm" bind:checked={cautionOnly} />
        <span class="label-text text-xs">주의 거래만 보기</span>
      </label>
      <label class="label cursor-pointer justify-start gap-2 pt-6">
        <input type="checkbox" class="checkbox checkbox-sm" bind:checked={candleStrongOnly} />
        <span class="label-text text-xs">candle 강세만 보기</span>
      </label>
    </div>
    <div class="overflow-x-auto">
      <table class="table table-zebra table-sm">
        <thead>
          <tr>
            <th>날짜</th>
            <th>종목</th>
            <th>테마</th>
            <th>점수</th>
            <th>진입가</th>
            <th>청산가</th>
            <th>수익률</th>
            <th>설명</th>
          </tr>
        </thead>
        <tbody>
          {#if filteredTrades.length === 0}
            <tr>
              <td colspan="8" class="text-center text-base-content/60">거래 데이터가 없습니다.</td>
            </tr>
          {:else}
            {#each filteredTrades as trade}
              <tr>
                <td>{trade.entryDate}</td>
                <td>
                  <div class="font-medium">{trade.name}</div>
                  <div class="text-xs text-base-content/60">{trade.code}</div>
                </td>
                <td>{trade.themeName ?? '-'}</td>
                <td>{trade.score?.toFixed?.(1) ?? '-'}</td>
                <td>{trade.entryPrice.toLocaleString()}</td>
                <td>{trade.exitPrice.toLocaleString()}</td>
                <td class:text-success={trade.pnlPct >= 0} class:text-error={trade.pnlPct < 0}>
                  {trade.pnlPct >= 0 ? '+' : ''}{trade.pnlPct.toFixed(2)}%
                </td>
                <td class="max-w-xs text-xs text-base-content/70">
                  <div>{trade.explanationSummary ?? '-'}</div>
                  {#if trade.explanation?.newsReasons?.length}
                    <div class="mt-1 text-[11px] text-base-content/50">뉴스: {trade.explanation.newsReasons[0]}</div>
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
