<script>
  import { overnightApi } from '../api/overnight';
  import BacktestDayCandidateTable from './BacktestDayCandidateTable.svelte';
  import BacktestTradeTable from './BacktestTradeTable.svelte';
  import EquityCurve from './EquityCurve.svelte';
  import ScoreBreakdown from './ScoreBreakdown.svelte';
  import StrategyCheckpoints from './StrategyCheckpoints.svelte';
  import ThemeSnapshotGrid from './ThemeSnapshotGrid.svelte';

  let { runId, onBack = null } = $props();

  let runDetail = $state(null);
  let reviewDate = $state('');
  let dayReview = $state(null);
  let selectedCandidate = $state(null);
  let isLoading = $state(true);
  let errorMessage = $state('');

  $effect(() => {
    runId;
    bootstrap();
  });

  async function bootstrap() {
    isLoading = true;
    errorMessage = '';
    runDetail = null;
    dayReview = null;
    selectedCandidate = null;
    reviewDate = '';

    try {
      runDetail = await overnightApi.getBacktestDetail(runId);
      const reviewDates = runDetail?.candidateDates ?? [];
      reviewDate = reviewDates[0] ?? runDetail?.run?.dateTo ?? '';
      if (reviewDate) {
        await loadDayReview(reviewDate);
      }
    } catch (error) {
      errorMessage = String(error);
    } finally {
      isLoading = false;
    }
  }

  async function loadDayReview(date) {
    if (!date) {
      dayReview = null;
      selectedCandidate = null;
      return;
    }

    dayReview = await overnightApi.getBacktestDayReview(runId, date);
    selectedCandidate = dayReview?.candidates?.[0] ?? null;
  }

  async function handleChangeDate() {
    errorMessage = '';
    await loadDayReview(reviewDate);
  }

  function handleSelectCandidate(candidate) {
    selectedCandidate = candidate;
  }

  const reviewDates = $derived(
    runDetail?.candidateDates ?? []
  );

  const reviewSignal = $derived.by(() => {
    if (!selectedCandidate) return null;
    return {
      code: selectedCandidate.code,
      name: selectedCandidate.name,
      themeName: selectedCandidate.themeName,
      signalDate: dayReview?.date ?? reviewDate,
      score: selectedCandidate.score,
      scoreTheme: selectedCandidate.scoreTheme,
      scoreNews: selectedCandidate.scoreNews,
      scoreTechnical: selectedCandidate.scoreTechnical,
      scoreRisk: selectedCandidate.scoreRisk,
      reasoning: selectedCandidate.reasoning,
      structuredReasoning: selectedCandidate.structuredReasoning,
      changeRate: selectedCandidate.changeRate,
      newsCount: selectedCandidate.newsCount,
      ma5Support: selectedCandidate.ma5Support,
    };
  });

  const reviewDetail = $derived.by(() => {
    if (!selectedCandidate || !reviewSignal) return null;
    return {
      signal: reviewSignal,
      history: selectedCandidate.history ?? [],
      relatedNews: selectedCandidate.relatedNews ?? [],
      explanation: selectedCandidate.explanation,
      candleSignal: selectedCandidate.candleSignal,
      strategyCheckpoints: selectedCandidate.strategyCheckpoints ?? [],
    };
  });
</script>

<section class="mx-auto flex min-h-screen max-w-7xl flex-col gap-6 p-6">
  <header class="rounded-3xl bg-slate-950 px-6 py-7 text-white shadow-2xl">
    <div class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <button class="btn btn-sm btn-outline text-white" onclick={() => onBack?.()}>← 대시보드로</button>
        <h1 class="mt-3 text-3xl font-semibold">백테스트 결과 분석</h1>
        <p class="mt-2 text-sm text-slate-300">실행 결과 요약, 일자별 pass/fail 후보, 종목별 근거를 한 화면에서 확인합니다.</p>
      </div>

      {#if runDetail?.run}
        <div class="badge badge-lg badge-outline text-white">Run #{runDetail.run.id}</div>
      {/if}
    </div>
  </header>

  {#if errorMessage}
    <div class="alert alert-error shadow"><span>{errorMessage}</span></div>
  {/if}

  {#if isLoading}
    <div class="card bg-base-100 shadow">
      <div class="card-body py-10 text-center text-base-content/60">백테스트 결과를 불러오는 중입니다.</div>
    </div>
  {:else if runDetail?.run}
    <div class="stats stats-vertical xl:stats-horizontal shadow w-full bg-base-100">
      <div class="stat">
        <div class="stat-title">기간</div>
        <div class="stat-value text-lg">{runDetail.run.dateFrom} ~ {runDetail.run.dateTo}</div>
      </div>
      <div class="stat">
        <div class="stat-title">총 거래</div>
        <div class="stat-value">{runDetail.run.totalTrades}</div>
      </div>
      <div class="stat">
        <div class="stat-title">승률</div>
        <div class="stat-value text-secondary">{runDetail.run.winRate.toFixed(1)}%</div>
      </div>
      <div class="stat">
        <div class="stat-title">누적 수익률</div>
        <div class="stat-value {runDetail.run.totalReturn >= 0 ? 'text-success' : 'text-error'}">{runDetail.run.totalReturn >= 0 ? '+' : ''}{runDetail.run.totalReturn.toFixed(2)}%</div>
      </div>
      <div class="stat">
        <div class="stat-title">최대 낙폭</div>
        <div class="stat-value text-warning">{runDetail.run.maxDrawdown.toFixed(2)}%</div>
      </div>
    </div>

    <EquityCurve run={runDetail.run} />

    {#if runDetail?.scoreBuckets?.length}
      <div class="card bg-base-100 shadow">
        <div class="card-body">
          <div class="flex items-center justify-between gap-3">
            <h3 class="card-title text-base">점수대별 결과 상관 요약</h3>
            <span class="badge badge-outline">컷오프 {runDetail.run.minScore.toFixed(1)}점</span>
          </div>
          <p class="text-sm text-base-content/60">
            점수 기준으로 매수 대상을 자르기 전에, 각 점수대 후보가 실제로 다음 결과에서 어떤 성과를 냈는지 확인하는 표입니다.
          </p>

          <div class="overflow-x-auto">
            <table class="table table-sm">
              <thead>
                <tr>
                  <th>점수대</th>
                  <th>후보 수</th>
                  <th>결과 확인 수</th>
                  <th>실매수 수</th>
                  <th>승률</th>
                  <th>평균 점수</th>
                  <th>평균 수익률</th>
                </tr>
              </thead>
              <tbody>
                {#each runDetail.scoreBuckets as bucket}
                  <tr>
                    <td>{bucket.label}</td>
                    <td>{bucket.candidateCount}</td>
                    <td>{bucket.outcomeCount}</td>
                    <td>{bucket.tradedCount}</td>
                    <td>{bucket.winRate.toFixed(1)}%</td>
                    <td>{bucket.avgScore.toFixed(2)}</td>
                    <td class:text-success={bucket.avgReturn >= 0} class:text-error={bucket.avgReturn < 0}>
                      {bucket.avgReturn >= 0 ? '+' : ''}{bucket.avgReturn.toFixed(2)}%
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    {/if}

    <div class="card bg-base-100 shadow">
      <div class="card-body">
        <div class="flex items-center justify-between gap-3">
          <h3 class="card-title text-base">일자별 분석</h3>
          <span class="badge badge-outline">run 내부 drilldown</span>
        </div>

        <label class="form-control max-w-xs gap-2">
          <span class="label-text">분석할 날짜</span>
          <select class="select select-bordered" bind:value={reviewDate} onchange={handleChangeDate}>
            {#each reviewDates as date}
              <option value={date}>{date}</option>
            {/each}
          </select>
        </label>
      </div>
    </div>

    {#if dayReview}
      <ThemeSnapshotGrid themes={dayReview.themes} />

      <div class="stats stats-vertical xl:stats-horizontal shadow w-full bg-base-100">
        <div class="stat">
          <div class="stat-title">검토 일자</div>
          <div class="stat-value text-primary text-2xl">{dayReview.date}</div>
        </div>
        <div class="stat">
          <div class="stat-title">실매매 수</div>
          <div class="stat-value">{dayReview.tradedCount}</div>
        </div>
        <div class="stat">
          <div class="stat-title">통과 수</div>
          <div class="stat-value text-secondary">{dayReview.passCount}</div>
        </div>
        <div class="stat">
          <div class="stat-title">탈락 수</div>
          <div class="stat-value text-warning">{dayReview.failCount}</div>
        </div>
      </div>

      <div class="grid gap-6 2xl:grid-cols-[minmax(0,1fr)_360px]">
        <BacktestDayCandidateTable candidates={dayReview.candidates} selectedCode={selectedCandidate?.code ?? ''} onSelect={handleSelectCandidate} />

        <div class="space-y-6">
          {#if selectedCandidate && reviewSignal && reviewDetail}
            <div class="card bg-base-100 shadow">
              <div class="card-body">
                <div class="flex items-center justify-between gap-3">
                  <h3 class="card-title text-base">종목 분석</h3>
                  <span class={`badge ${selectedCandidate.wasTraded ? 'badge-success' : selectedCandidate.passedScore ? 'badge-primary' : 'badge-warning'}`}>
                    {selectedCandidate.wasTraded ? '실매매' : selectedCandidate.passedScore ? '통과' : '탈락'}
                  </span>
                </div>

                <div class="text-sm">
                  <div><span class="font-medium">종목:</span> {selectedCandidate.name} ({selectedCandidate.code})</div>
                  <div><span class="font-medium">테마:</span> {selectedCandidate.themeName ?? '-'}</div>
                  <div><span class="font-medium">Judal 날짜:</span> {dayReview.date}</div>
                </div>

                {#if !selectedCandidate.passedScore && selectedCandidate.rejectionReasons?.length}
                  <div class="rounded-2xl bg-warning/10 p-4 text-sm text-base-content/80">
                    <div class="mb-2 font-medium">탈락 이유</div>
                    <ul class="list-disc pl-5">
                      {#each selectedCandidate.rejectionReasons as reason}
                        <li>{reason}</li>
                      {/each}
                    </ul>
                  </div>
                {/if}
              </div>
            </div>

            <ScoreBreakdown signal={reviewSignal} explanation={reviewDetail.explanation} />
            <StrategyCheckpoints signal={reviewSignal} detail={reviewDetail} />

            <div class="card bg-base-100 shadow">
              <div class="card-body">
                <h3 class="card-title text-base">뉴스 / 캔들 / 테마 근거</h3>

                <div class="rounded-xl bg-slate-950 p-4 text-slate-50">
                  <div class="mb-2 text-sm font-medium text-cyan-300">종합 의견</div>
                  <p class="text-sm leading-6">{reviewDetail.signal.structuredReasoning?.reasoning ?? reviewDetail.signal.reasoning}</p>
                </div>

                <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
                  <div class="rounded-xl bg-base-200 p-3">
                    <div class="mb-1 font-medium">뉴스 요약</div>
                    <ul class="list-disc pl-5 text-xs text-base-content/70">
                      {#if reviewDetail.relatedNews.length === 0}
                        <li>연결된 뉴스가 없습니다.</li>
                      {:else}
                        {#each reviewDetail.relatedNews as article}
                          <li>{article.title}</li>
                        {/each}
                      {/if}
                    </ul>
                  </div>
                  <div class="rounded-xl bg-base-200 p-3">
                    <div class="mb-1 font-medium">캔들 / 차트</div>
                    <ul class="list-disc pl-5 text-xs text-base-content/70">
                      {#each reviewDetail.explanation?.technicalReasons ?? [] as reason}
                        <li>{reason}</li>
                      {/each}
                    </ul>
                  </div>
                  <div class="rounded-xl bg-base-200 p-3">
                    <div class="mb-1 font-medium">Judal / 테마</div>
                    <ul class="list-disc pl-5 text-xs text-base-content/70">
                      {#each reviewDetail.explanation?.themeReasons ?? [] as reason}
                        <li>{reason}</li>
                      {/each}
                    </ul>
                  </div>
                </div>

                <div>
                  <div class="mb-1 font-medium">최근 히스토리</div>
                  <div class="space-y-1 text-xs text-base-content/70">
                    {#each reviewDetail.history as item}
                      <div class="flex justify-between gap-3 rounded bg-base-200 px-3 py-2">
                        <span>{item.crawlDate}</span>
                        <span>{item.currentPrice?.toLocaleString?.() ?? '-'}원</span>
                      </div>
                    {/each}
                  </div>
                </div>
              </div>
            </div>
          {/if}
        </div>
      </div>
    {/if}

    <BacktestTradeTable trades={runDetail.trades} />
  {/if}
</section>
