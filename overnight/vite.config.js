import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { execFileSync, spawn } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const cargoManifestPath = path.resolve(__dirname, 'src-rust/Cargo.toml');

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

function runBridgeJson(args) {
  const stdout = execFileSync(
    'cargo',
    ['run', '--quiet', '--manifest-path', cargoManifestPath, '--bin', 'overnight-bridge', '--', ...args],
    {
      cwd: __dirname,
      env: sanitizedEnv(),
      encoding: 'utf8',
    }
  );
  return JSON.parse(stdout);
}

function browserBridgePlugin() {
  const jobs = new Map();

  function writeJson(res, payload, statusCode = 200) {
    res.statusCode = statusCode;
    res.setHeader('Content-Type', 'application/json');
    res.end(JSON.stringify(payload));
  }

  function route(reqUrl) {
    return new URL(reqUrl, 'http://localhost:1420');
  }

  return {
    name: 'browser-real-bridge',
    configureServer(server) {
      server.middlewares.use('/api/health', (_req, res) => {
        try {
          writeJson(res, runBridgeJson(['health-check']));
        } catch (error) {
          writeJson(res, { error: String(error) }, 500);
        }
      });

      server.middlewares.use('/api/app-paths', (_req, res) => {
        try {
          writeJson(res, runBridgeJson(['app-paths']));
        } catch (error) {
          writeJson(res, { error: String(error) }, 500);
        }
      });

      server.middlewares.use('/api/available-dates', (_req, res) => {
        try {
          writeJson(res, runBridgeJson(['available-dates']));
        } catch (error) {
          writeJson(res, { error: String(error), dates: [] }, 500);
        }
      });

      server.middlewares.use('/api/signals/generate', async (req, res) => {
        if (req.method !== 'POST') {
          writeJson(res, { error: 'method not allowed' }, 405);
          return;
        }

        try {
          const chunks = [];
          for await (const chunk of req) {
            chunks.push(chunk);
          }
          const body = JSON.parse(Buffer.concat(chunks).toString('utf8') || '{}');
          const date = body?.date;
          const minScore = Number(body?.minScore ?? 8);
          writeJson(res, runBridgeJson(['generate-signals', String(date), String(minScore)]));
        } catch (error) {
          writeJson(res, { error: String(error) }, 500);
        }
      });

      server.middlewares.use('/api/signals', (req, res) => {
        try {
          const url = route(req.url);
          const segments = url.pathname.split('/').filter(Boolean);

          if (req.method === 'GET' && segments.length === 2) {
            const date = url.searchParams.get('date');
            writeJson(res, date ? runBridgeJson(['get-signals', date]) : runBridgeJson(['get-signals']));
            return;
          }

          if (req.method === 'GET' && segments.length === 3) {
            const date = url.searchParams.get('date');
            if (!date) {
              writeJson(res, { error: 'date is required' }, 400);
              return;
            }
            writeJson(res, runBridgeJson(['get-signal-detail', segments[2], date]));
            return;
          }

          writeJson(res, { error: 'unsupported signals route' }, 404);
        } catch (error) {
          writeJson(res, { error: String(error) }, 500);
        }
      });

      server.middlewares.use('/api/backtests/stats', (_req, res) => {
        try {
          writeJson(res, runBridgeJson(['backtest-stats']));
        } catch (error) {
          writeJson(res, { error: String(error) }, 500);
        }
      });

      server.middlewares.use('/api/backtests/runs', (_req, res) => {
        try {
          writeJson(res, runBridgeJson(['list-backtest-runs']));
        } catch (error) {
          writeJson(res, { error: String(error) }, 500);
        }
      });

      server.middlewares.use('/api/backtests/jobs', async (req, res) => {
        const url = route(req.url);
        const segments = url.pathname.split('/').filter(Boolean);

        if (req.method === 'GET' && segments.length === 2) {
          writeJson(res, [...jobs.values()].sort((left, right) => right.startedAt.localeCompare(left.startedAt)));
          return;
        }

        if (req.method === 'GET' && segments.length === 3) {
          const job = jobs.get(segments[2]);
          if (!job) {
            writeJson(res, { error: `job not found: ${segments[2]}` }, 404);
            return;
          }
          writeJson(res, job);
          return;
        }

        writeJson(res, { error: 'unsupported jobs route' }, 404);
      });

      server.middlewares.use('/api/backtests/start', async (req, res) => {
        if (req.method !== 'POST') {
          writeJson(res, { error: 'method not allowed' }, 405);
          return;
        }

        try {
          const chunks = [];
          for await (const chunk of req) {
            chunks.push(chunk);
          }
          const body = JSON.parse(Buffer.concat(chunks).toString('utf8') || '{}');
          const { dateFrom, dateTo, minScore } = body;
          const jobId = `browser-backtest-${Date.now()}`;
          const startedAt = new Date().toISOString();
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
            startedAt,
            finishedAt: null,
          };
          jobs.set(jobId, job);

          const child = spawn(
            'cargo',
            [
              'run',
              '--quiet',
              '--manifest-path',
              cargoManifestPath,
              '--bin',
              'overnight-bridge',
              '--',
              'run-backtest',
              dateFrom,
              dateTo,
              String(Number(minScore ?? 8)),
            ],
            {
              cwd: __dirname,
              env: sanitizedEnv(),
            }
          );

          job.status = 'running';
          let stderr = '';
          let stdoutBuffer = '';
          let sawResultMessage = false;

          function consumeJsonLine(line) {
            const trimmed = line.trim();
            if (!trimmed) return;

            let message;
            try {
              message = JSON.parse(trimmed);
            } catch {
              return;
            }

            if (message.type === 'progress') {
              job.completedSteps = message.completedSteps ?? job.completedSteps;
              job.totalSteps = message.totalSteps ?? job.totalSteps;
              job.currentDate = message.currentDate ?? job.currentDate;
            }
            if (message.type === 'result') {
              sawResultMessage = true;
              job.status = 'completed';
              job.runId = message.payload?.run?.id ?? null;
              job.currentDate = null;
              job.finishedAt = new Date().toISOString();
            }
          }

          child.stdout.on('data', (chunk) => {
            stdoutBuffer += chunk.toString();
            const lines = stdoutBuffer.split('\n');
            stdoutBuffer = lines.pop() ?? '';

            for (const line of lines) consumeJsonLine(line);
          });

          child.stderr.on('data', (chunk) => {
            stderr += chunk.toString();
          });

          child.on('close', (code) => {
            if (stdoutBuffer.trim().length > 0) {
              consumeJsonLine(stdoutBuffer);
            }

            if (job.status !== 'completed') {
              job.status = 'failed';
              job.finishedAt = new Date().toISOString();
              job.error = code === 0
                ? (sawResultMessage ? null : 'backtest finished without result payload')
                : (stderr.trim() || `backtest process exited with code ${code}`);
            }
          });

          writeJson(res, { jobId });
        } catch (error) {
          writeJson(res, { error: String(error) }, 500);
        }
      });

      server.middlewares.use('/api/backtests/', (req, res) => {
        try {
          const url = route(req.url);
          const segments = url.pathname.split('/').filter(Boolean);
          if (segments.length < 3) {
            writeJson(res, { error: 'unsupported route' }, 404);
            return;
          }

          const runId = segments[2];
          if (segments.length === 3 && req.method === 'GET') {
            writeJson(res, runBridgeJson(['get-backtest-detail', runId]));
            return;
          }

          if (segments.length === 4 && segments[3] === 'day-review' && req.method === 'GET') {
            const date = url.searchParams.get('date');
            writeJson(res, runBridgeJson(['get-backtest-day-review', runId, date]));
            return;
          }

          writeJson(res, { error: 'unsupported route' }, 404);
        } catch (error) {
          writeJson(res, { error: String(error) }, 500);
        }
      });
    },
  };
}

export default defineConfig({
  plugins: [svelte(), browserBridgePlugin()],
  server: {
    port: 1420,
    strictPort: true,
  },
  clearScreen: false,
});
