import express from 'express';
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync, spawn } from 'node:child_process';

const app = express();
const jobs = new Map();
const rootDir = path.resolve(path.dirname(new URL(import.meta.url).pathname), '..');
const cargoManifestPath = path.resolve(rootDir, 'src-rust/Cargo.toml');
const distDir = path.resolve(rootDir, 'dist');
const bridgeBin = process.env.BRIDGE_BIN;
const serverPort = Number(process.env.PORT || 1420);
const localBridgeCandidates = [
  path.resolve(rootDir, 'src-rust/target/release/overnight-bridge'),
  path.resolve(rootDir, 'src-rust/target/debug/overnight-bridge'),
];

function sanitizedEnv() {
  const env = { ...process.env };
  delete env.LD_LIBRARY_PATH;
  delete env.LD_PRELOAD;
  delete env.GTK_PATH;
  delete env.SNAP;
  delete env.SNAP_NAME;
  delete env.SNAP_REVISION;
  delete env.SNAP_ARCH;
  delete env.SNAP_COOKIE;
  delete env.SNAP_DATA;
  delete env.SNAP_COMMON;
  delete env.SNAP_CONTEXT;
  delete env.SNAP_INSTANCE_NAME;
  delete env.SNAP_INSTANCE_KEY;
  delete env.SNAP_LIBRARY_PATH;
  return env;
}

function resolveBridgeBin() {
  if (bridgeBin && fs.existsSync(bridgeBin)) {
    return bridgeBin;
  }

  return localBridgeCandidates.find((candidate) => fs.existsSync(candidate)) ?? null;
}

function bridgeCommand(args) {
  const resolvedBridgeBin = resolveBridgeBin();
  if (resolvedBridgeBin) {
    return {
      cmd: resolvedBridgeBin,
      args,
    };
  }

  return {
    cmd: 'cargo',
    args: ['run', '--quiet', '--manifest-path', cargoManifestPath, '--bin', 'overnight-bridge', '--', ...args],
  };
}

function parseLastJsonLine(stdout) {
  const lines = stdout
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean);
  for (let index = lines.length - 1; index >= 0; index -= 1) {
    try {
      return JSON.parse(lines[index]);
    } catch {
      // keep scanning previous lines
    }
  }
  throw new Error('bridge did not return JSON');
}

function runBridgeJson(args) {
  const { cmd, args: cmdArgs } = bridgeCommand(args);
  const stdout = execFileSync(cmd, cmdArgs, {
    cwd: rootDir,
    env: sanitizedEnv(),
    encoding: 'utf8',
  });
  return parseLastJsonLine(stdout);
}

function parseBridgeLine(job, line) {
  const trimmed = line.trim();
  if (!trimmed) return;

  let payload;
  try {
    payload = JSON.parse(trimmed);
  } catch {
    return;
  }

  if (payload.type === 'progress') {
    job.completedSteps = payload.completedSteps ?? job.completedSteps;
    job.totalSteps = payload.totalSteps ?? job.totalSteps;
    job.currentDate = payload.currentDate ?? job.currentDate;
  }

  if (payload.type === 'result') {
    job.status = 'completed';
    job.runId = payload.payload?.run?.id ?? null;
    job.currentDate = null;
    job.finishedAt = new Date().toISOString();
  }
}

function toErrorMessage(error) {
  if (!error) return 'unknown error';
  if (typeof error === 'string') return error;
  return error.message || String(error);
}

app.use(express.json());

app.get('/api/health', (_req, res) => {
  try {
    res.json(runBridgeJson(['health-check']));
  } catch (error) {
    res.status(500).json({ error: toErrorMessage(error) });
  }
});

app.get('/api/app-paths', (_req, res) => {
  try {
    res.json(runBridgeJson(['app-paths']));
  } catch (error) {
    res.status(500).json({ error: toErrorMessage(error) });
  }
});

app.get('/api/available-dates', (_req, res) => {
  try {
    res.json(runBridgeJson(['available-dates']));
  } catch (error) {
    res.status(500).json({ error: toErrorMessage(error), dates: [] });
  }
});

app.get('/api/signals', (req, res) => {
  try {
    const date = req.query.date;
    const args = date ? ['get-signals', String(date)] : ['get-signals'];
    res.json(runBridgeJson(args));
  } catch (error) {
    res.status(500).json({ error: toErrorMessage(error) });
  }
});

app.post('/api/signals/generate', (req, res) => {
  try {
    const { date, minScore } = req.body || {};
    if (!date) {
      res.status(400).json({ error: 'date is required' });
      return;
    }
    const args = ['generate-signals', String(date), String(Number(minScore ?? 8))];
    res.json(runBridgeJson(args));
  } catch (error) {
    res.status(500).json({ error: toErrorMessage(error) });
  }
});

app.get('/api/signals/:code', (req, res) => {
  try {
    const date = req.query.date;
    if (!date) {
      res.status(400).json({ error: 'date is required' });
      return;
    }
    res.json(runBridgeJson(['get-signal-detail', req.params.code, String(date)]));
  } catch (error) {
    res.status(500).json({ error: toErrorMessage(error) });
  }
});

app.get('/api/backtests/stats', (_req, res) => {
  try {
    res.json(runBridgeJson(['backtest-stats']));
  } catch (error) {
    res.status(500).json({ error: toErrorMessage(error) });
  }
});

app.get('/api/backtests/runs', (_req, res) => {
  try {
    res.json(runBridgeJson(['list-backtest-runs']));
  } catch (error) {
    res.status(500).json({ error: toErrorMessage(error) });
  }
});

app.post('/api/backtests/start', (req, res) => {
  try {
    const { dateFrom, dateTo, minScore } = req.body || {};
    if (!dateFrom || !dateTo) {
      res.status(400).json({ error: 'dateFrom and dateTo are required' });
      return;
    }

    const jobId = `server-backtest-${Date.now()}`;
    const job = {
      jobId,
      status: 'queued',
      dateFrom,
      dateTo,
      minScore: Number(minScore ?? 8),
      completedSteps: 0,
      totalSteps: 0,
      currentDate: null,
      runId: null,
      error: null,
      startedAt: new Date().toISOString(),
      finishedAt: null,
    };
    jobs.set(jobId, job);

    const { cmd, args } = bridgeCommand([
      'run-backtest',
      String(dateFrom),
      String(dateTo),
      String(Number(minScore ?? 8)),
    ]);

    const child = spawn(cmd, args, {
      cwd: rootDir,
      env: sanitizedEnv(),
    });

    job.status = 'running';
    let stderr = '';
    let stdoutBuffer = '';

    child.stdout.on('data', (chunk) => {
      stdoutBuffer += chunk.toString();
      const lines = stdoutBuffer.split('\n');
      stdoutBuffer = lines.pop() ?? '';
      for (const line of lines) {
        parseBridgeLine(job, line);
      }
    });

    child.stderr.on('data', (chunk) => {
      stderr += chunk.toString();
    });

    child.on('close', (code) => {
      if (stdoutBuffer.trim().length > 0) {
        parseBridgeLine(job, stdoutBuffer);
      }
      if (job.status !== 'completed') {
        job.status = 'failed';
        job.finishedAt = new Date().toISOString();
        job.error = code === 0 ? 'backtest finished without result payload' : (stderr.trim() || `backtest process exited with code ${code}`);
      }
    });

    res.json({ jobId });
  } catch (error) {
    res.status(500).json({ error: toErrorMessage(error) });
  }
});

app.get('/api/backtests/jobs', (_req, res) => {
  const payload = [...jobs.values()].sort((left, right) => right.startedAt.localeCompare(left.startedAt));
  res.json(payload);
});

app.get('/api/backtests/jobs/:jobId', (req, res) => {
  const job = jobs.get(req.params.jobId);
  if (!job) {
    res.status(404).json({ error: `job not found: ${req.params.jobId}` });
    return;
  }
  res.json(job);
});

app.get('/api/backtests/:runId/day-review', (req, res) => {
  try {
    const { runId } = req.params;
    const date = req.query.date;
    if (!date) {
      res.status(400).json({ error: 'date is required' });
      return;
    }
    res.json(runBridgeJson(['get-backtest-day-review', String(runId), String(date)]));
  } catch (error) {
    res.status(500).json({ error: toErrorMessage(error) });
  }
});

app.get('/api/backtests/:runId', (req, res) => {
  try {
    res.json(runBridgeJson(['get-backtest-detail', String(req.params.runId)]));
  } catch (error) {
    res.status(500).json({ error: toErrorMessage(error) });
  }
});

if (fs.existsSync(distDir)) {
  app.use(express.static(distDir));
  app.get('*', (_req, res) => {
    res.sendFile(path.resolve(distDir, 'index.html'));
  });
}

app.listen(serverPort, () => {
  process.stdout.write(`overnight server listening on :${serverPort}\n`);
});
