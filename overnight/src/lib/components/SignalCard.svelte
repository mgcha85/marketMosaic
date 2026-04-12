<script>
  let { signal, onSelect } = $props();

  const badgeTone = $derived(
    signal.score >= 9 ? 'badge-success' : signal.score >= 8 ? 'badge-primary' : 'badge-warning'
  );
  const statusTone = $derived(signal.status === 'pass' ? 'badge-success' : 'badge-warning');

  const firstKeyPoint = $derived(signal.structuredReasoning?.keyNewsPoints?.[0] ?? null);
  const firstCaution = $derived(signal.structuredReasoning?.cautionPoints?.[0] ?? null);
</script>

<button class="card bg-base-100 shadow hover:shadow-lg transition-all text-left" onclick={() => onSelect?.(signal)}>
  <div class="card-body gap-3">
    <div class="flex items-start justify-between gap-3">
      <div>
        <h3 class="card-title text-lg">{signal.name}</h3>
        <p class="text-sm text-base-content/60">{signal.code} · {signal.themeName ?? '테마 없음'}</p>
      </div>
      <div class="flex flex-col items-end gap-2">
        <span class={`badge badge-outline ${statusTone}`}>{signal.status === 'pass' ? '통과' : '탈락'}</span>
        <span class={`badge badge-lg ${badgeTone}`}>{signal.score.toFixed(1)}점</span>
      </div>
    </div>

    <p class="text-sm leading-6 text-base-content/80">{signal.reasoning}</p>

    {#if firstKeyPoint}
      <div class="rounded-xl bg-primary/10 px-3 py-2 text-xs text-primary-content/80 bg-primary/10 text-slate-700">
        핵심 포인트: {firstKeyPoint}
      </div>
    {/if}

    {#if firstCaution}
      <div class="badge badge-warning badge-outline w-fit">주의: {firstCaution}</div>
    {/if}

    <div class="flex flex-wrap gap-2 text-xs">
      <span class="badge badge-outline">뉴스 {signal.newsCount}건</span>
      <span class="badge badge-outline">등락률 {signal.changeRate?.toFixed?.(2) ?? '-'}%</span>
      {#if signal.ma5Support}
        <span class="badge badge-outline badge-success">5일선 지지</span>
      {/if}
    </div>
  </div>
</button>
