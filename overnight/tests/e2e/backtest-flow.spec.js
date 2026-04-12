import { test, expect } from '@playwright/test';

test('backtest job reaches terminal state', async ({ page }) => {
  await page.goto('/');

  const startButton = page.getByRole('button', { name: '백테스트 시작' });
  await expect(startButton).toBeVisible();
  await startButton.click();

  const completedBadge = page.locator('.badge', { hasText: 'completed' }).first();
  const failedBadge = page.locator('.badge', { hasText: 'failed' }).first();

  await expect
    .poll(
      async () => {
        const done = await completedBadge.count();
        if (done > 0) return 'completed';
        const failed = await failedBadge.count();
        if (failed > 0) return 'failed';
        return 'running';
      },
      {
        timeout: 120_000,
        intervals: [1000, 1500, 2000],
      }
    )
    .toMatch(/completed|failed/);

  const finalState = await (async () => {
    if ((await completedBadge.count()) > 0) return 'completed';
    if ((await failedBadge.count()) > 0) return 'failed';
    return 'running';
  })();

  expect(finalState).not.toBe('running');
});
