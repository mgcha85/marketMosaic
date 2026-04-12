<script>
  let { signal, detail = null } = $props();

  function first(items, fallback) {
    return items?.[0] ?? fallback;
  }

  const checkpoints = $derived.by(() => {
    if (!signal) return [];

    if (detail?.strategyCheckpoints?.length) {
      return detail.strategyCheckpoints;
    }

    const structuredReasoning = detail?.signal?.structuredReasoning ?? signal.structuredReasoning ?? {};
    const explanation = detail?.explanation ?? {};
    const candleSignal = detail?.candleSignal ?? {};

    return [
      {
        label: 'A급 점수 기준',
        source: '전략 3·5',
        passed: (signal.score ?? 0) >= 8,
        note: `총점 ${signal.score?.toFixed?.(1) ?? '-'} / 10`,
      },
      {
        label: 'Judal 주도 테마',
        source: '전략 1·3·6',
        passed: Boolean((explanation.themeReasons ?? []).length || signal.themeName),
        note: first(explanation.themeReasons, signal.themeName ?? '테마 정보 없음'),
      },
      {
        label: '재료 명분 확보',
        source: '전략 3·5·6',
        passed: (signal.newsCount ?? 0) > 0 || Boolean((structuredReasoning.keyNewsPoints ?? []).length),
        note: first(structuredReasoning.keyNewsPoints, first(explanation.newsReasons, `관련 뉴스 ${signal.newsCount ?? 0}건`)),
      },
      {
        label: '지지·방어 확인',
        source: '전략 2·4',
        passed: Boolean(signal.ma5Support || candleSignal.aboveVwap || candleSignal.bullishClose),
        note: first(
          explanation.technicalReasons,
          signal.ma5Support ? '현재가가 MA5 위에서 지지' : '캔들 방어 신호 확인 필요'
        ),
      },
      {
        label: '과열 리스크 점검',
        source: '전략 1·5',
        passed: (signal.scoreRisk ?? 0) >= 1,
        note: first(explanation.riskReasons, `리스크 점수 ${signal.scoreRisk?.toFixed?.(1) ?? '-'}`),
      },
      {
        label: '종합 의견 정리',
        source: '전략 전체 요약',
        passed: Boolean((structuredReasoning.reasoning ?? signal.reasoning)?.trim?.()),
        note: structuredReasoning.reasoning ?? signal.reasoning,
      },
    ];
  });
</script>

<div class="card bg-base-100 shadow">
  <div class="card-body">
    <div class="flex items-center justify-between gap-3">
      <h3 class="card-title text-base">전략 체크포인트</h3>
      <span class="badge badge-outline">제공된 strategy 기준</span>
    </div>

    <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {#each checkpoints as checkpoint}
        <div class={`rounded-2xl border p-4 ${checkpoint.passed ? 'border-success/30 bg-success/5' : 'border-warning/30 bg-warning/5'}`}>
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="text-sm font-semibold">{checkpoint.label}</div>
              <div class="text-xs text-base-content/50">{checkpoint.source}</div>
            </div>
            <span class={`badge ${checkpoint.passed ? 'badge-success' : 'badge-warning'}`}>
              {checkpoint.passed ? '통과' : '주의'}
            </span>
          </div>

          <p class="mt-3 text-xs leading-5 text-base-content/70">{checkpoint.note}</p>
        </div>
      {/each}
    </div>
  </div>
</div>
