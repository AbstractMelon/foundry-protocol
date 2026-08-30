const fs = require('fs');
const path = require('path');

const APP_DIR = __dirname;
const RUN_DIR = path.join(APP_DIR, 'run');

const env = { NODE_ENV: 'production' };
try {
  const raw = fs.readFileSync(path.join(RUN_DIR, 'deploy.env'), 'utf8');
  const apiKey = raw.match(/^FOUNDRY_GATEWAY_API_KEY=(.*)$/m)?.[1];
  if (apiKey) env.FOUNDRY_GATEWAY_API_KEY = apiKey.trim();
} catch {
  // No deploy.env yet; the gateway will refuse registration calls until one exists.
}

module.exports = {
  apps: [
    {
      name: 'foundry-world',
      cwd: APP_DIR,
      script: path.join(APP_DIR, 'bin', 'world'),
      args: '-addr :36743 -world main -autosave-ticks 600',
      interpreter: 'none',
      env,
      autorestart: true,
      max_restarts: 10,
      restart_delay: 3000,
      exp_backoff_restart_delay: 100,
      kill_timeout: 5000,
      out_file: path.join(RUN_DIR, 'world.out.log'),
      error_file: path.join(RUN_DIR, 'world.err.log'),
      merge_logs: true,
      time: true,
    },
    {
      name: 'foundry-gateway',
      cwd: APP_DIR,
      script: path.join(APP_DIR, 'bin', 'gateway'),
      args: '-config deploy/gateway.prod.yaml',
      interpreter: 'none',
      env,
      autorestart: true,
      max_restarts: 10,
      restart_delay: 3000,
      exp_backoff_restart_delay: 100,
      kill_timeout: 5000,
      out_file: path.join(RUN_DIR, 'gateway.out.log'),
      error_file: path.join(RUN_DIR, 'gateway.err.log'),
      merge_logs: true,
      time: true,
    },
  ],
};
