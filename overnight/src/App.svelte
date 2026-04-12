<script>
  import OvernightDashboard from './lib/components/OvernightDashboard.svelte';

  const initialParams = new URLSearchParams(window.location.search);
  const initialRunId = Number(initialParams.get('runId'));
  const hasInitialRunId = Number.isFinite(initialRunId) && initialRunId > 0;
  const initialDashboardMode =
    initialParams.get('mode') === 'analysis' || initialParams.get('view') === 'backtest'
      ? 'analysis'
      : 'launcher';

  let activeView = $state('dashboard');
  let selectedRunId = $state(hasInitialRunId ? initialRunId : null);
  let dashboardMode = $state(initialDashboardMode);

  function handleOpenRun(runId) {
    selectedRunId = runId;
    activeView = 'dashboard';
    dashboardMode = 'analysis';
  }

  function handleOpenLauncherTab() {
    activeView = 'dashboard';
    dashboardMode = 'launcher';
  }

  function handleOpenAnalysisTab() {
    activeView = 'dashboard';
    dashboardMode = 'analysis';
  }
</script>

<main class="min-h-screen bg-base-200">
  <div class="border-b border-base-300 bg-base-100/90 backdrop-blur">
    <div class="mx-auto flex max-w-7xl items-center gap-2 px-6 py-3">
      <button
        class={`btn btn-sm ${activeView === 'dashboard' && dashboardMode === 'launcher' ? 'btn-primary' : 'btn-ghost'}`}
        onclick={handleOpenLauncherTab}
      >
        백테스트 런처
      </button>
      <button
        class={`btn btn-sm ${activeView === 'dashboard' && dashboardMode === 'analysis' ? 'btn-primary' : 'btn-ghost'}`}
        onclick={handleOpenAnalysisTab}
      >
        Judal 분석 탭
      </button>
      {#if activeView === 'dashboard' && dashboardMode === 'analysis' && selectedRunId}
        <span class="ml-2 text-sm text-base-content/60">Run #{selectedRunId}</span>
      {/if}
    </div>
  </div>

  <OvernightDashboard onOpenRun={handleOpenRun} analysisMode={dashboardMode === 'analysis'} />
</main>
