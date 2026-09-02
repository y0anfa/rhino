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
  tr.run { cursor: pointer; }
  tr.run:hover td { background: #1c2128; }
  tr.tasks td { background: #0f141a; padding: 6px 14px 10px 28px; }
  .task-row { display: flex; gap: 14px; padding: 4px 0; font-size: 0.85em; align-items: baseline; }
  .task-name { min-width: 160px; }
  .task-dur { color: var(--dim); min-width: 60px; }
  .badge { padding: 2px 8px; border-radius: 12px; font-size: 0.8em; font-weight: 600; }
  .badge-success { background: #1a3a2a; color: var(--green); }
  .badge-failed { background: #3a1a1a; color: var(--red); }
  .badge-running { background: #3a2a0a; color: var(--yellow); }
  .badge-skipped { background: #22272e; color: var(--dim); }
  .mono { font-family: 'SF Mono', Monaco, monospace; font-size: 0.85em; color: var(--dim); }
  .error { color: var(--red); font-size: 0.8em; white-space: pre-wrap; }
  .refresh { color: var(--dim); font-size: 0.8em; margin-top: 16px; }
  button { background: #21262d; color: var(--text); border: 1px solid var(--border); border-radius: 6px; padding: 4px 10px; cursor: pointer; font-size: 0.8em; }
  button:hover { background: #30363d; }
  button:disabled { opacity: 0.5; cursor: default; }
  #notice { color: var(--dim); font-size: 0.85em; min-height: 1.2em; margin: 8px 0; }
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
  <h2>Workflows</h2>
  <table>
    <thead><tr><th>Name</th><th></th></tr></thead>
    <tbody id="wf-body"><tr><td colspan="2">Loading...</td></tr></tbody>
  </table>
  <div id="notice"></div>
  <h2>Recent Runs <span class="mono">(click a run to see its tasks)</span></h2>
  <table>
    <thead><tr><th>ID</th><th>Workflow</th><th>Status</th><th>Trigger</th><th>Started</th><th>Duration</th></tr></thead>
    <tbody id="runs-body"><tr><td colspan="6">Loading...</td></tr></tbody>
  </table>
  <div class="refresh">Auto-refreshes every 5 seconds</div>
</div>
<script>
const expanded = new Set();
function esc(s) {
  return String(s == null ? '' : s).replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}
function badge(status) {
  const known = {success: 'badge-success', failed: 'badge-failed', skipped: 'badge-skipped'};
  return '<span class="badge ' + (known[status] || 'badge-running') + '">' + esc(status) + '</span>';
}
function fmt(iso) {
  if (!iso) return '-';
  return new Date(iso).toLocaleTimeString();
}
function ms(v) {
  if (v < 1000) return v + 'ms';
  return (v / 1000).toFixed(1) + 's';
}
function duration(start, end) {
  if (!start || !end) return '-';
  return ms(new Date(end) - new Date(start));
}
function notice(text) {
  document.getElementById('notice').textContent = text;
}
async function triggerRun(name, btn) {
  btn.disabled = true;
  try {
    const res = await fetch('/api/workflows/' + encodeURIComponent(name) + '/run', {method: 'POST'});
    const body = await res.json();
    notice(res.ok ? 'Triggered ' + name + ' (run ' + body.run_id + ')' : 'Failed to trigger ' + name + ': ' + (body.error || res.status));
    refresh();
  } catch (e) {
    notice('Failed to trigger ' + name + ': ' + e);
  } finally {
    btn.disabled = false;
  }
}
async function toggleTasks(runId) {
  if (expanded.has(runId)) expanded.delete(runId); else expanded.add(runId);
  refresh();
}
async function renderTasks(runId) {
  const row = document.getElementById('tasks-' + runId);
  if (!row) return;
  try {
    const res = await fetch('/api/runs/' + encodeURIComponent(runId));
    const detail = await res.json();
    const tasks = detail.tasks || [];
    if (tasks.length === 0) {
      row.innerHTML = '<td colspan="6" class="mono">No task executions recorded</td>';
      return;
    }
    row.innerHTML = '<td colspan="6">' + tasks.map(t =>
      '<div class="task-row"><span class="task-name">' + esc(t.task_name) + '</span>' + badge(t.status) +
      '<span class="task-dur">' + ms(t.duration_ms || 0) + '</span>' +
      (t.retries ? '<span class="mono">retries: ' + t.retries + '</span>' : '') +
      (t.error ? '<span class="error">' + esc(t.error) + '</span>' : '') + '</div>'
    ).join('') + '</td>';
  } catch (e) {
    row.innerHTML = '<td colspan="6" class="error">' + esc(e) + '</td>';
  }
}
async function refresh() {
  try {
    const [wfRes, runsRes, healthRes] = await Promise.all([
      fetch('/api/workflows'), fetch('/api/runs'), fetch('/api/health')
    ]);
    const workflows = (await wfRes.json()) || [];
    const runs = (await runsRes.json()) || [];
    const health = await healthRes.json();

    document.getElementById('wf-count').textContent = workflows.length;
    document.getElementById('run-count').textContent = runs.length;
    document.getElementById('uptime').textContent = health.uptime || '-';

    const ok = runs.filter(r => r.status === 'success').length;
    document.getElementById('success-rate').textContent = runs.length ? Math.round(ok/runs.length*100) + '%' : '-';

    const wfBody = document.getElementById('wf-body');
    wfBody.innerHTML = workflows.length === 0
      ? '<tr><td colspan="2">No workflows found</td></tr>'
      : workflows.map(name =>
          '<tr><td>' + esc(name) + '</td><td style="text-align:right"><button data-wf="' + esc(name) + '">Run</button></td></tr>'
        ).join('');
    wfBody.querySelectorAll('button[data-wf]').forEach(btn =>
      btn.addEventListener('click', () => triggerRun(btn.dataset.wf, btn)));

    const tbody = document.getElementById('runs-body');
    if (runs.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6">No runs yet</td></tr>';
      return;
    }
    tbody.innerHTML = runs.map(r => {
      const id = r.id.length > 14 ? r.id.slice(0, 14) : r.id;
      let html = '<tr class="run" data-run="' + esc(r.id) + '"><td class="mono">' + esc(id) + '</td><td>' + esc(r.workflow_name) +
        '</td><td>' + badge(r.status) + '</td><td>' + esc(r.trigger_type || '-') +
        '</td><td>' + fmt(r.started_at) + '</td><td>' + duration(r.started_at, r.completed_at) + '</td></tr>';
      if (expanded.has(r.id)) {
        html += '<tr class="tasks" id="tasks-' + esc(r.id) + '"><td colspan="6" class="mono">Loading tasks...</td></tr>';
      }
      return html;
    }).join('');
    tbody.querySelectorAll('tr.run').forEach(tr =>
      tr.addEventListener('click', () => toggleTasks(tr.dataset.run)));
    expanded.forEach(renderTasks);
  } catch(e) { console.error(e); }
}
refresh();
setInterval(refresh, 5000);
</script>
</body>
</html>`
