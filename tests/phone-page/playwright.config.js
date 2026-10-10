import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: '.',
  testMatch: 'question-page.spec.js',
  fullyParallel: false,
  workers: 1,
  reporter: 'line',
  globalSetup: './setup.js',
  use: {
    trace: 'retain-on-failure'
  },
  projects: [
    {
      name: 'webkit-iphone',
      use: {
        ...devices['iPhone 13'],
        browserName: 'webkit'
      }
    },
    {
      name: 'chromium-android',
      use: {
        ...devices['Pixel 5'],
        browserName: 'chromium'
      }
    }
  ]
});
