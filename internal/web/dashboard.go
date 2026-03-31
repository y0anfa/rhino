package web

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Rhino Dashboard</title>
<style>
  :root { --bg: #0d1117; --card: #161b22; --border: #30363d; --text: #c9d1d9; --dim: #8b949e; --green: #3fb950; --red: #f85149; --yellow: #d29922; --blue: #58a6ff; }
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', monospace; background: var(--bg); color: var(--text); }
  .container { max-width: 960px; margin: 0 auto; padding: 20px; }
  h1 { color: var(--blue); margin-bottom: 20px; font-size: 1.5em; }
  h2 { color: var(--text); margin: 20px 0 10px; font-size: 1.1em; }
  .stats { display: flex; gap: 16px; margin-bottom: 24px; }
  .stat { background: var(--card); border: 1px solid var(--border); border-radius: 8px; padding: 16px; flex: 1; }
  .stat-value { font-size: 1.8em; font-weight: bold; color: var(--blue); }
  .stat-label { color: var(--dim); font-size: 0.85em; margin-top: 4px; }
  table { width: 100%; border-collapse: collapse; background: var(--card); border-radius: 8px; overflow: hidden; border: 1px solid var(--border); }
  th { text-align: left; padding: 10px 14px; background: var(--border); color: var(--text); font-size: 0.85em; }
  td { padding: 10px 14px; border-top: 1px solid var(--border); font-size: 0.9em; }
  tr:hover td { background: #1c2128; }
  .badge { padding: 2px 8px; border-radius: 12px; font-size: 0.8em; font-weight: 600; }
  .badge-success { background: #1a3a2a; color: var(--green); }
  .badge-failed { background: #3a1a1a; color: var(--red); }
  .badge-running { background: #3a2a0a; color: var(--yellow); }
  .mono { font-family: 'SF Mono', Monaco, monospace; font-size: 0.85em; color: var(--dim); }
  .error { color: var(--red); font-size: 0.8em; }
  .refresh { color: var(--dim); font-size: 0.8em; margin-top: 16px; }
</style>
</head>
<body>
<div class="container">
  <h1>&#129430; Rhino Dashboard</h1>
  <div class="stats">
    <div class="stat"><div class="stat-value" id="wf-count">-</div><div class="stat-label">Workflows</div></div>
    <div class="stat"><div class="stat-value" id="run-count">-</div><div class="stat-label">Total Runs</div></div>
    <div class="stat"><div class="stat-value" id="success-rate">-</div><div class="stat-label">Success Rate</div></div>
    <div class="stat"><div class="stat-value" id="uptime">-</div><div class="stat-label">Uptime</div></div>
  </div>
  <h2>Recent Runs</h2>
  <table>
    <thead><tr><th>ID</th><th>Workflow</th><th>Status</th><th>Trigger</th><th>Started</th><th>Duration</th></tr></thead>
    <tbody id="runs-body"><tr><td colspan="6">Loading...</td></tr></tbody>
  </table>
  <div class="refresh">Auto-refreshes every 5 seconds</div>
</div>
<script>
function badge(status) {
  const cls = status === 'success' ? 'badge-success' : status === 'failed' ? 'badge-failed' : 'badge-running';
  return '<span class="badge ' + cls + '">' + status + '</span>';
}
function fmt(iso) {
  if (!iso) return '-';
  const d = new Date(iso);
  return d.toLocaleTimeString();
}
function duration(start, end) {
  if (!start || !end) return '-';
  const ms = new Date(end) - new Date(start);
  if (ms < 1000) return ms + 'ms';
  return (ms / 1000).toFixed(1) + 's';
}
async function refresh() {
  try {
    const [wfRes, runsRes, healthRes] = await Promise.all([
      fetch('/api/workflows'), fetch('/api/runs'), fetch('/api/health')
    ]);
    const workflows = await wfRes.json();
    const runs = await runsRes.json();
    const health = await healthRes.json();

    document.getElementById('wf-count').textContent = (workflows || []).length;
    document.getElementById('run-count').textContent = (runs || []).length;
    document.getElementById('uptime').textContent = health.uptime || '-';

    const total = (runs || []).length;
    const ok = (runs || []).filter(r => r.status === 'success').length;
    document.getElementById('success-rate').textContent = total ? Math.round(ok/total*100) + '%' : '-';

    const tbody = document.getElementById('runs-body');
    if (!runs || runs.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6">No runs yet</td></tr>';
      return;
    }
    tbody.innerHTML = runs.map(r => {
      const id = r.id.length > 14 ? r.id.slice(0, 14) : r.id;
      return '<tr><td class="mono">' + id + '</td><td>' + r.workflow_name +
        '</td><td>' + badge(r.status) + '</td><td>' + (r.trigger_type||'-') +
        '</td><td>' + fmt(r.started_at) + '</td><td>' + duration(r.started_at, r.completed_at) + '</td></tr>';
    }).join('');
  } catch(e) { console.error(e); }
}
refresh();
setInterval(refresh, 5000);
</script>
</body>
</html>`
