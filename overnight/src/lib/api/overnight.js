async function browserRequest(url, options) {
  const response = await fetch(url, options);
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(payload?.error ?? `request failed: ${response.status}`);
  }
  return payload;
}

async function invokeBrowserBridge(command, args = {}) {
  switch (command) {
    case 'health_check':
      return browserRequest('/api/health');
    case 'get_app_paths':
      return browserRequest('/api/app-paths');
    case 'get_available_history_dates':
      return (await browserRequest('/api/available-dates')).dates ?? [];
    case 'get_signals':
      return browserRequest(`/api/signals${args.date ? `?date=${encodeURIComponent(args.date)}` : ''}`);
    case 'generate_signals':
      return browserRequest('/api/signals/generate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(args),
      });
    case 'get_signal_detail':
      return browserRequest(`/api/signals/${encodeURIComponent(args.code)}?date=${encodeURIComponent(args.date)}`);
    case 'start_backtest':
      return browserRequest('/api/backtests/start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(args),
      });
    case 'get_backtest_job_status':
      return browserRequest(`/api/backtests/jobs/${args.jobId}`);
    case 'list_backtest_jobs':
      return browserRequest('/api/backtests/jobs');
    case 'list_backtest_runs':
      return browserRequest('/api/backtests/runs');
    case 'get_backtest_stats':
      return browserRequest('/api/backtests/stats');
    case 'get_backtest_detail':
      return browserRequest(`/api/backtests/${args.runId}`);
    case 'get_backtest_day_review':
      return browserRequest(`/api/backtests/${args.runId}/day-review?date=${encodeURIComponent(args.date)}`);
    default:
      throw new Error(`Browser bridge does not support command: ${command}`);
  }
}

export const overnightApi = {
  healthCheck() {
    return invokeBrowserBridge('health_check');
  },
  getAppPaths() {
    return invokeBrowserBridge('get_app_paths');
  },
  getAvailableHistoryDates() {
    return invokeBrowserBridge('get_available_history_dates');
  },
  getSignals(date) {
    return invokeBrowserBridge('get_signals', { date });
  },
  generateSignals(date, minScore) {
    return invokeBrowserBridge('generate_signals', { date, minScore });
  },
  getSignalDetail(code, date) {
    return invokeBrowserBridge('get_signal_detail', { code, date });
  },
  startBacktest(dateFrom, dateTo, minScore) {
    return invokeBrowserBridge('start_backtest', { dateFrom, dateTo, minScore });
  },
  getBacktestJobStatus(jobId) {
    return invokeBrowserBridge('get_backtest_job_status', { jobId });
  },
  listBacktestJobs() {
    return invokeBrowserBridge('list_backtest_jobs');
  },
  runBacktest(dateFrom, dateTo, minScore) {
    return invokeBrowserBridge('run_backtest', { dateFrom, dateTo, minScore });
  },
  listBacktestRuns() {
    return invokeBrowserBridge('list_backtest_runs');
  },
  getBacktestStats() {
    return invokeBrowserBridge('get_backtest_stats');
  },
  getBacktestDetail(runId) {
    return invokeBrowserBridge('get_backtest_detail', { runId });
  },
  getBacktestDayReview(runId, date) {
    return invokeBrowserBridge('get_backtest_day_review', { runId, date });
  },
};
