<script>
  import { onMount } from 'svelte';

  import { overnightApi } from '../api/overnight';
  import { defaultBacktestStats, defaultHealth } from '../stores/overnight';
  import BacktestSummary from './BacktestSummary.svelte';
  import DailySignalTable from './DailySignalTable.svelte';
  import SignalCard from './SignalCard.svelte';
  import ScoreBreakdown from './ScoreBreakdown.svelte';
  import StrategyCheckpoints from './StrategyCheckpoints.svelte';

  let { onOpenRun = null, analysisMode = false } = $props();

  let health = $state({ ...defaultHealth });
  let appPaths = $state(null);
  let dates = $state([]);
  let selectedDate = $state('');
  let dateFrom = $state('');
  let dateTo = $state('');
  let minScore = $state(8);
  let signals = $state([]);
  let selectedSignal = $state(null);
  let selectedDetail = $state(null);
  let backtestStats = $state({ ...defaultBacktestStats });
  let backtestRuns = $state([]);
  let backtestJobs = $state([]);
  let isBooting = $state(true);
  let isGenerating = $state(false);
  let isSubmittingBacktest = $state(false);
  let errorMessage = $state('');
  let pollingHandle = null;

  onMount(async () => {
    await bootstrap();

    return () => {
      if (pollingHandle) clearInterval(pollingHandle);
    };
  });

  async function bootstrap() {
    isBooting = true;
    errorMessage = '';

    try {
      const [healthResult, paths, historyDates, stats, runs, jobs] = await Promise.all([
        overnightApi.healthCheck(),
        overnightApi.getAppPaths(),
        overnightApi.getAvailableHistoryDates(),
        overnightApi.getBacktestStats(),
        overnightApi.listBacktestRuns(),
        overnightApi.listBacktestJobs().catch(() => []),
      ]);

      health = healthResult;
      appPaths = paths;
      dates = historyDates;
      selectedDate = historyDates[0] ?? '';
      dateTo = historyDates[0] ?? '';
      dateFrom = historyDates[historyDates.length - 1] ?? historyDates[0] ?? '';
      backtestStats = stats;
      backtestRuns = runs;
      backtestJobs = jobs;

      if (selectedDate) {
        await loadSignalsForDate(selectedDate);
      }

      const runningJob = jobs.find((job) => job.status === 'queued' || job.status === 'running');
      if (runningJob) {
        startPollingJob(runningJob.jobId);
      }
    } catch (error) {
      errorMessage = String(error);
    } finally {
      isBooting = false;
    }
  }

  async function refreshBacktestCollections() {
    const [stats, runs, jobs] = await Promise.all([
      overnightApi.getBacktestStats(),
      overnightApi.listBacktestRuns(),
      overnightApi.listBacktestJobs().catch(() => []),
    ]);
    backtestStats = stats;
    backtestRuns = runs;
    backtestJobs = jobs;
  }

  async function handleGenerateSignals() {
    if (!selectedDate) return;

    isGenerating = true;
    errorMessage = '';
    try {
      const result = await overnightApi.generateSignals(selectedDate, Number(minScore));
      signals = result.signals;
      selectedSignal = signals[0] ?? null;
      selectedDetail = null;
      if (selectedSignal) {
        await loadSignalDetail(selectedSignal);
      }
    } catch (error) {
      errorMessage = String(error);
    } finally {
      isGenerating = false;
    }
  }

  async function handleSubmitBacktest() {
    if (!dateFrom || !dateTo) return;

    isSubmittingBacktest = true;
    errorMessage = '';
    try {
      const started = await overnightApi.startBacktest(dateFrom, dateTo, Number(minScore));
      await refreshBacktestCollections();
      startPollingJob(started.jobId);
    } catch (error) {
      errorMessage = String(error);
    } finally {
      isSubmittingBacktest = false;
    }
  }

  function startPollingJob(jobId) {
    if (pollingHandle) clearInterval(pollingHandle);

    pollingHandle = setInterval(async () => {
      try {
        const status = await overnightApi.getBacktestJobStatus(jobId);
        backtestJobs = [
          status,
          ...backtestJobs.filter((job) => job.jobId !== status.jobId),
        ].sort((left, right) => right.startedAt.localeCompare(left.startedAt));

        if (status.status === 'completed' && status.runId) {
          clearInterval(pollingHandle);
          pollingHandle = null;
          await refreshBacktestCollections();
        }

        if (status.status === 'failed') {
          clearInterval(pollingHandle);
          pollingHandle = null;
          errorMessage = status.error ?? '백테스트 실행이 실패했습니다.';
          await refreshBacktestCollections();
        }
      } catch (error) {
        clearInterval(pollingHandle);
        pollingHandle = null;
        errorMessage = String(error);
      }
    }, 1200);
  }

  async function loadSignalsForDate(date) {
    if (!date) {
      signals = [];
      selectedSignal = null;
      selectedDetail = null;
      return;
    }

    const loadedSignals = await overnightApi.getSignals(date);
    signals = loadedSignals;
    selectedSignal = loadedSignals[0] ?? null;
    selectedDetail = null;

    if (selectedSignal) {
      await loadSignalDetail(selectedSignal, date);
    }
  }

  async function loadSignalDetail(signal, signalDate = selectedDate) {
    if (!signal || !signalDate) return;

    try {
      selectedDetail = await overnightApi.getSignalDetail(signal.code, signalDate);
    } catch (error) {
      errorMessage = String(error);
      selectedDetail = null;
    }
  }

  async function handleSelectSignal(signal) {
    selectedSignal = signal;
    await loadSignalDetail(signal);
  }

  async function handleChangeDate() {
    errorMessage = '';
    await loadSignalsForDate(selectedDate);
  }

  function openRun(runId) {
    onOpenRun?.(runId);
  }

  function openRunInAnalysisTab(runId) {
    openRun(runId);
  }

  function progressPercent(job) {
    if (!job?.totalSteps) return 0;
    return Math.min(100, Math.round((job.completedSteps / job.totalSteps) * 100));
  }

  function isJobRunning(job) {
    return job?.status === 'queued' || job?.status === 'running';
  }
</script>

<section class="mx-auto flex min-h-screen max-w-7xl flex-col gap-6 p-6">
  <header class="rounded-3xl bg-slate-950 px-6 py-7 text-white shadow-2xl">
    <div class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
      <div>
        <p class="text-sm uppercase tracking-[0.3em] text-cyan-300">MarketMosaic / Overnight</p>
        <h1 class="mt-2 text-3xl font-semibold">Overnight 전략 런처</h1>
        <p class="mt-3 max-w-3xl text-sm text-slate-300">
          백테스트는 비동기로 실행하고, 완료된 run은 별도 분석 화면에서 일자별·종목별로 drilldown 합니다.
        </p>
      </div>

      <div class="flex flex-wrap gap-2 text-sm">
        <span class="badge badge-lg {health.status === 'ok' ? 'badge-success' : 'badge-warning'}">{health.status}</span>
        <span class="badge badge-lg badge-outline text-white">v{health.version}</span>
      </div>
    </div>
  </header>

  {#if errorMessage}
    <div class="alert alert-error shadow"><span>{errorMessage}</span></div>
  {/if}

  <div class="grid gap-6 xl:grid-cols-[360px_minmax(0,1fr)]">
    <aside class="space-y-6">
      {#if !analysisMode}
        <div class="card bg-base-100 shadow">
          <div class="card-body">
            <h2 class="card-title text-base">백테스트 실행</h2>

          <label class="form-control gap-2">
            <span class="label-text">시작 날짜</span>
            <select class="select select-bordered" bind:value={dateFrom}>
              {#each dates as date}
                <option value={date}>{date}</option>
              {/each}
            </select>
          </label>

          <label class="form-control gap-2">
            <span class="label-text">종료 날짜</span>
            <select class="select select-bordered" bind:value={dateTo}>
              {#each dates as date}
                <option value={date}>{date}</option>
              {/each}
            </select>
          </label>

          <label class="form-control gap-2">
            <span class="label-text">최소 점수</span>
            <input class="range range-primary" type="range" min="6" max="10" step="0.5" bind:value={minScore} />
            <div class="text-right text-sm text-base-content/70">{Number(minScore).toFixed(1)}점 이상</div>
          </label>

            <button class="btn btn-primary" onclick={handleSubmitBacktest} disabled={isSubmittingBacktest || !dateFrom || !dateTo}>
              {#if isSubmittingBacktest}<span class="loading loading-spinner loading-sm"></span>{/if}
              백테스트 시작
            </button>
            <div class="text-xs leading-5 text-base-content/50">
              실행은 비동기로 돌아가며, 완료된 run을 클릭하면 같은 페이지의 Judal 분석 탭으로 전환됩니다.
            </div>
          </div>
        </div>
      {/if}

      {#if analysisMode}
        <div class="card bg-base-100 shadow">
          <div class="card-body">
            <h2 class="card-title text-base">일일 신호 재계산</h2>

          <label class="form-control gap-2">
            <span class="label-text">기준 날짜</span>
            <select class="select select-bordered" bind:value={selectedDate} onchange={handleChangeDate}>
              {#each dates as date}
                <option value={date}>{date}</option>
              {/each}
            </select>
          </label>

          <button class="btn btn-secondary" onclick={handleGenerateSignals} disabled={isGenerating || !selectedDate}>
            {#if isGenerating}<span class="loading loading-spinner loading-sm"></span>{/if}
            선택 날짜 신호 재계산
          </button>

            <div class="text-xs leading-5 text-base-content/50">
              선택 날짜의 Judal 후보를 다시 점수화해서 통과/탈락 목록을 저장합니다. 장중 실매수 주문이 아니라, 그 날짜에 어떤 종목이 전략 조건에 걸렸는지 확인하는 단계입니다.
            </div>
          </div>
        </div>
      {/if}

      {#if analysisMode}
        <div class="card bg-base-100 shadow">
          <div class="card-body text-sm">
            <h2 class="card-title text-base">로컬 경로</h2>
            {#if appPaths}
              <div class="space-y-3 break-all text-xs text-base-content/70">
                <div>
                  <div class="font-medium text-base-content">overnight.db</div>
                  <div>{appPaths.overnightDbPath}</div>
                </div>
                <div>
                  <div class="font-medium text-base-content">judal.db</div>
                  <div>{appPaths.judalDbPath}</div>
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/if}
    </aside>

    <div class="space-y-6">
      {#if analysisMode}
        <BacktestSummary stats={backtestStats} />
      {/if}

      <div class="grid gap-6 xl:grid-cols-2">
        <div class="card bg-base-100 shadow">
          <div class="card-body">
            <div class="flex items-center justify-between gap-3">
              <h3 class="card-title text-base">실행 중 / 최근 작업</h3>
              <span class="badge badge-outline">jobs {backtestJobs.length}</span>
            </div>

            <div class="space-y-3">
              {#if backtestJobs.length === 0}
                <div class="rounded-2xl border border-dashed border-base-300 p-6 text-sm text-base-content/60">아직 실행된 백테스트 작업이 없습니다.</div>
              {:else}
                {#each backtestJobs as job}
                  <div class="rounded-2xl border border-base-300 p-4">
                    <div class="flex items-center justify-between gap-3">
                      <div class="min-w-0 flex-1">
                        <div class="text-sm font-semibold">{job.dateFrom} ~ {job.dateTo}</div>
                        <div class="text-xs text-base-content/50">최소점수 {job.minScore.toFixed(1)} · job {job.jobId}</div>
                      </div>
                      <div class="flex items-center gap-2">
                        {#if isJobRunning(job)}
                          <span class="loading loading-spinner loading-sm text-primary"></span>
                        {/if}
                        <span class={`badge ${job.status === 'completed' ? 'badge-success' : job.status === 'failed' ? 'badge-error' : 'badge-primary'}`}>
                          {job.status}
                        </span>
                      </div>
                    </div>

                    {#if isJobRunning(job)}
                      <progress class="progress progress-primary mt-3 w-full" value={progressPercent(job)} max="100"></progress>
                      <div class="mt-2 flex items-center justify-between gap-3 text-xs text-base-content/60">
                        <span>{job.currentDate ? `${job.currentDate} 처리 중` : '대기 중'}</span>
                        <span>{job.completedSteps}/{job.totalSteps || '?'}</span>
                      </div>
                    {/if}

                    {#if job.status === 'completed' && job.runId}
                      <div class="mt-3 flex justify-end">
                        <button class="btn btn-sm btn-primary" onclick={() => openRunInAnalysisTab(job.runId)}>결과 보기</button>
                      </div>
                    {/if}

                    {#if job.status === 'failed' && job.error}
                      <div class="mt-3 text-xs text-error">{job.error}</div>
                    {/if}
                  </div>
                {/each}
              {/if}
            </div>
          </div>
        </div>

        <div class="card bg-base-100 shadow">
          <div class="card-body">
            <div class="flex items-center justify-between gap-3">
              <h3 class="card-title text-base">완료된 백테스트 runs</h3>
              <span class="badge badge-outline">{backtestRuns.length}개</span>
            </div>

            <div class="space-y-3">
              {#if backtestRuns.length === 0}
                <div class="rounded-2xl border border-dashed border-base-300 p-6 text-sm text-base-content/60">완료된 run이 없습니다.</div>
              {:else}
                {#each backtestRuns as run}
                  <button class="w-full rounded-2xl border border-base-300 p-4 text-left transition hover:border-primary hover:bg-primary/5" onclick={() => openRunInAnalysisTab(run.id)}>
                    <div class="flex items-center justify-between gap-3">
                      <div>
                        <div class="font-semibold">Run #{run.id}</div>
                        <div class="text-sm text-base-content/60">{run.dateFrom} ~ {run.dateTo}</div>
                      </div>
                      <span class={`badge ${run.totalReturn >= 0 ? 'badge-success' : 'badge-error'}`}>
                        {run.totalReturn >= 0 ? '+' : ''}{run.totalReturn.toFixed(2)}%
                      </span>
                    </div>
                    <div class="mt-3 grid grid-cols-3 gap-2 text-xs text-base-content/60">
                      <div>거래 {run.totalTrades}</div>
                      <div>승률 {run.winRate.toFixed(1)}%</div>
                      <div>DD {run.maxDrawdown.toFixed(2)}%</div>
                    </div>
                  </button>
                {/each}
              {/if}
            </div>
          </div>
        </div>
      </div>

      {#if analysisMode}
        <div class="card bg-base-100 shadow">
          <div class="card-body py-4">
            <h2 class="card-title text-base">Judal Dashboard</h2>
          </div>
        </div>

        <div class="grid gap-6 2xl:grid-cols-[minmax(0,1fr)_360px]">
          <div class="card bg-base-100 shadow">
            <div class="card-body">
              <div class="flex items-center justify-between gap-3">
                <h2 class="card-title">조건 검색식 1차 후보</h2>
                <span class="badge badge-outline">{signals.length}개</span>
              </div>

              {#if isBooting}
                <div class="py-12 text-center text-base-content/60">앱을 초기화하는 중입니다.</div>
              {:else if signals.length === 0}
                <div class="py-12 text-center text-base-content/60">저장된 신호가 없습니다. 먼저 선택 날짜 신호 재계산을 실행하세요.</div>
              {:else}
                <DailySignalTable signals={signals} selectedCode={selectedSignal?.code ?? ''} onSelect={handleSelectSignal} />

                <div class="grid gap-4 xl:grid-cols-2">
                  {#each signals as signal}
                    <SignalCard {signal} onSelect={handleSelectSignal} />
                  {/each}
                </div>
              {/if}
            </div>
          </div>

          <div class="space-y-6">
            {#if selectedSignal}
              <ScoreBreakdown signal={selectedSignal} explanation={selectedDetail?.explanation} />
              <StrategyCheckpoints signal={selectedSignal} detail={selectedDetail} />
            {/if}
          </div>
        </div>
      {/if}
    </div>
  </div>
</section>
