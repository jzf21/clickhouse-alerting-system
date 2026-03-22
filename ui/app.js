const API = '';
let historyOffset = 0;
const historyLimit = 20;

// Tab navigation
document.getElementById('nav').addEventListener('click', e => {
  if (e.target.tagName !== 'BUTTON') return;
  document.querySelectorAll('#nav button').forEach(b => b.classList.remove('active'));
  document.querySelectorAll('.tab-content').forEach(t => t.classList.add('hidden'));
  e.target.classList.add('active');
  const tab = e.target.dataset.tab;
  document.getElementById('tab-' + tab).classList.remove('hidden');
  loadTab(tab);
});

function loadTab(tab) {
  switch (tab) {
    case 'alerts': loadAlerts(); break;
    case 'rules': loadRules(); break;
    case 'history': historyOffset = 0; loadHistory(); break;
    case 'silences': loadSilences(); break;
    case 'channels': loadChannels(); break;
  }
}

async function api(path, opts = {}) {
  const res = await fetch(API + path, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  });
  if (res.status === 204) return null;
  return res.json();
}

// Alerts
async function loadAlerts() {
  const alerts = await api('/api/alerts');
  const el = document.getElementById('alerts-table');
  if (!alerts.length) {
    el.innerHTML = '<div class="empty">No active alerts</div>';
    return;
  }
  el.innerHTML = `<table>
    <thead><tr><th>Status</th><th>Rule</th><th>Severity</th><th>Value</th><th>Since</th><th>Actions</th></tr></thead>
    <tbody>${alerts.map(a => `<tr>
      <td><span class="badge badge-${a.state}">${a.state}</span></td>
      <td>${esc(a.rule_name)}</td>
      <td><span class="badge badge-${a.severity}">${a.severity}</span></td>
      <td>${a.last_eval_value != null ? Number(a.last_eval_value).toFixed(2) : '-'}</td>
      <td>${a.firing_since ? timeAgo(a.firing_since) : a.pending_since ? timeAgo(a.pending_since) : '-'}</td>
      <td class="actions">
        ${(a.state === 'firing' || a.state === 'pending') ? `<button class="btn btn-sm btn-danger" onclick="silenceAlert('${esc(a.rule_name)}')">Silence</button>` : ''}
      </td>
    </tr>`).join('')}</tbody></table>`;
}

function silenceAlert(ruleName) {
  const endsAt = new Date(Date.now() + 2 * 3600 * 1000).toISOString().slice(0, 16);
  showModal('Create Silence', `
    <div class="form-group"><label>Matchers (JSON)</label>
      <textarea id="sil-matchers">[{"label":"alertname","value":"${esc(ruleName)}"}]</textarea>
    </div>
    <div class="form-group"><label>Comment</label><input id="sil-comment" value="Silenced from dashboard"></div>
    <div class="form-group"><label>Ends At</label><input type="datetime-local" id="sil-ends" value="${endsAt}"></div>
    <div class="form-actions">
      <button class="btn btn-primary" onclick="createSilenceFromForm()">Create</button>
      <button class="btn" onclick="hideModal()">Cancel</button>
    </div>
  `);
}

// Rules
async function loadRules() {
  const rules = await api('/api/rules');
  const el = document.getElementById('rules-table');
  if (!rules.length) {
    el.innerHTML = '<div class="empty">No rules configured</div>';
    return;
  }
  el.innerHTML = `<table>
    <thead><tr><th>Name</th><th>Severity</th><th>Condition</th><th>Interval</th><th>Enabled</th><th>Actions</th></tr></thead>
    <tbody>${rules.map(r => `<tr>
      <td>${esc(r.name)}</td>
      <td><span class="badge badge-${r.severity}">${r.severity}</span></td>
      <td><code>${esc(r.column)} ${opSym(r.operator)} ${r.threshold}</code></td>
      <td>${r.eval_interval}s</td>
      <td><span class="toggle" onclick="toggleRule('${r.id}', ${!r.enabled})">${r.enabled ? 'On' : 'Off'}</span></td>
      <td class="actions">
        <button class="btn btn-sm btn-primary" onclick="editRule('${r.id}')">Edit</button>
        <button class="btn btn-sm btn-danger" onclick="deleteRule('${r.id}')">Delete</button>
      </td>
    </tr>`).join('')}</tbody></table>`;
}

async function toggleRule(id, enabled) {
  await api(`/api/rules/${id}`, { method: 'PUT', body: JSON.stringify({ enabled }) });
  loadRules();
}

async function deleteRule(id) {
  if (!confirm('Delete this rule?')) return;
  await api(`/api/rules/${id}`, { method: 'DELETE' });
  loadRules();
}

async function editRule(id) {
  const r = await api(`/api/rules/${id}`);
  showRuleForm(r);
}

async function showRuleForm(r) {
  const isEdit = !!r;
  const selectedIds = isEdit ? (r.channel_ids || []) : [];
  let channels = [];
  try { channels = await api('/api/channels'); } catch(e) {}
  const channelsHtml = channels.length
    ? channels.map(c => `<label class="channel-option"><input type="checkbox" value="${c.id}" ${selectedIds.includes(c.id) ? 'checked' : ''}> ${esc(c.name)} <span class="channel-type">(${c.type})</span></label>`).join('')
    : '<div class="text-muted">No channels configured yet</div>';
  showModal(isEdit ? 'Edit Rule' : 'New Rule', `
    <div class="form-group"><label>Name</label><input id="rf-name" value="${isEdit ? esc(r.name) : ''}"></div>
    <div class="form-group"><label>Query (SQL)</label><textarea id="rf-query">${isEdit ? esc(r.query) : ''}</textarea></div>
    <div class="form-row">
      <div class="form-group"><label>Column</label><input id="rf-column" value="${isEdit ? esc(r.column) : ''}"></div>
      <div class="form-group"><label>Operator</label>
        <select id="rf-op">
          ${['gt','gte','lt','lte','eq','neq'].map(o => `<option value="${o}" ${isEdit && r.operator === o ? 'selected' : ''}>${opSym(o)}</option>`).join('')}
        </select>
      </div>
    </div>
    <div class="form-row">
      <div class="form-group"><label>Threshold</label><input type="number" step="any" id="rf-threshold" value="${isEdit ? r.threshold : ''}"></div>
      <div class="form-group"><label>Severity</label>
        <select id="rf-severity">
          ${['warning','critical','info'].map(s => `<option value="${s}" ${isEdit && r.severity === s ? 'selected' : ''}>${s}</option>`).join('')}
        </select>
      </div>
    </div>
    <div class="form-row">
      <div class="form-group"><label>Eval Interval (seconds)</label><input type="number" id="rf-interval" value="${isEdit ? r.eval_interval : 60}"></div>
      <div class="form-group"><label>For Duration (seconds)</label><input type="number" id="rf-for" value="${isEdit ? r.for_duration : 0}"></div>
    </div>
    <div class="form-group"><label>Labels (JSON)</label><textarea id="rf-labels">${isEdit ? JSON.stringify(r.labels || {}, null, 2) : '{}'}</textarea></div>
    <div class="form-group"><label>Channels</label><div id="rf-channels" class="channel-checklist">${channelsHtml}</div></div>
    <div class="form-actions">
      <button class="btn btn-primary" onclick="saveRule(${isEdit ? `'${r.id}'` : 'null'})">${isEdit ? 'Update' : 'Create'}</button>
      <button class="btn" onclick="hideModal()">Cancel</button>
    </div>
  `);
}

async function saveRule(id) {
  const body = {
    name: document.getElementById('rf-name').value,
    query: document.getElementById('rf-query').value,
    column: document.getElementById('rf-column').value,
    operator: document.getElementById('rf-op').value,
    threshold: parseFloat(document.getElementById('rf-threshold').value),
    severity: document.getElementById('rf-severity').value,
    eval_interval: parseInt(document.getElementById('rf-interval').value),
    for_duration: parseInt(document.getElementById('rf-for').value),
    labels: JSON.parse(document.getElementById('rf-labels').value),
    channel_ids: [...document.querySelectorAll('#rf-channels input[type="checkbox"]:checked')].map(cb => cb.value),
    enabled: true,
  };
  if (id) {
    await api(`/api/rules/${id}`, { method: 'PUT', body: JSON.stringify(body) });
  } else {
    await api('/api/rules', { method: 'POST', body: JSON.stringify(body) });
  }
  hideModal();
  loadRules();
}

// History
async function loadHistory() {
  const events = await api(`/api/alerts/history?limit=${historyLimit}&offset=${historyOffset}`);
  const el = document.getElementById('history-table');
  if (!events.length) {
    el.innerHTML = '<div class="empty">No alert history</div>';
    document.getElementById('history-pagination').innerHTML = '';
    return;
  }
  el.innerHTML = `<table>
    <thead><tr><th>Time</th><th>Rule</th><th>State</th><th>Severity</th><th>Value</th></tr></thead>
    <tbody>${events.map(e => `<tr>
      <td>${new Date(e.created_at).toLocaleString()}</td>
      <td>${esc(e.rule_name)}</td>
      <td><span class="badge badge-${e.state}">${e.state}</span></td>
      <td><span class="badge badge-${e.severity}">${e.severity}</span></td>
      <td>${Number(e.value).toFixed(2)}</td>
    </tr>`).join('')}</tbody></table>`;

  const pag = document.getElementById('history-pagination');
  pag.innerHTML = '';
  if (historyOffset > 0) {
    const prev = document.createElement('button');
    prev.className = 'btn btn-sm btn-primary';
    prev.textContent = 'Previous';
    prev.onclick = () => { historyOffset -= historyLimit; loadHistory(); };
    pag.appendChild(prev);
  }
  if (events.length === historyLimit) {
    const next = document.createElement('button');
    next.className = 'btn btn-sm btn-primary';
    next.textContent = 'Next';
    next.onclick = () => { historyOffset += historyLimit; loadHistory(); };
    pag.appendChild(next);
  }
}

// Silences
async function loadSilences() {
  const silences = await api('/api/silences');
  const el = document.getElementById('silences-table');
  if (!silences.length) {
    el.innerHTML = '<div class="empty">No silences</div>';
    return;
  }
  const now = new Date();
  el.innerHTML = `<table>
    <thead><tr><th>Matchers</th><th>Comment</th><th>Ends At</th><th>Status</th><th>Actions</th></tr></thead>
    <tbody>${silences.map(s => {
      const active = new Date(s.starts_at) <= now && new Date(s.ends_at) > now;
      return `<tr>
        <td><code>${esc(s.matchers)}</code></td>
        <td>${esc(s.comment)}</td>
        <td>${new Date(s.ends_at).toLocaleString()}</td>
        <td><span class="badge ${active ? 'badge-firing' : 'badge-inactive'}">${active ? 'active' : 'expired'}</span></td>
        <td class="actions"><button class="btn btn-sm btn-danger" onclick="deleteSilence('${s.id}')">Delete</button></td>
      </tr>`;
    }).join('')}</tbody></table>`;
}

function showSilenceForm() {
  const endsAt = new Date(Date.now() + 2 * 3600 * 1000).toISOString().slice(0, 16);
  showModal('New Silence', `
    <div class="form-group"><label>Matchers (JSON array)</label>
      <textarea id="sil-matchers">[{"label":"","value":""}]</textarea>
    </div>
    <div class="form-group"><label>Comment</label><input id="sil-comment"></div>
    <div class="form-group"><label>Created By</label><input id="sil-by"></div>
    <div class="form-group"><label>Ends At</label><input type="datetime-local" id="sil-ends" value="${endsAt}"></div>
    <div class="form-actions">
      <button class="btn btn-primary" onclick="createSilenceFromForm()">Create</button>
      <button class="btn" onclick="hideModal()">Cancel</button>
    </div>
  `);
}

async function createSilenceFromForm() {
  const body = {
    matchers: JSON.parse(document.getElementById('sil-matchers').value),
    comment: document.getElementById('sil-comment').value,
    created_by: document.getElementById('sil-by') ? document.getElementById('sil-by').value : '',
    ends_at: new Date(document.getElementById('sil-ends').value).toISOString(),
  };
  await api('/api/silences', { method: 'POST', body: JSON.stringify(body) });
  hideModal();
  loadSilences();
}

async function deleteSilence(id) {
  if (!confirm('Delete this silence?')) return;
  await api(`/api/silences/${id}`, { method: 'DELETE' });
  loadSilences();
}

// Channels
async function loadChannels() {
  const channels = await api('/api/channels');
  const el = document.getElementById('channels-table');
  if (!channels.length) {
    el.innerHTML = '<div class="empty">No channels configured</div>';
    return;
  }
  el.innerHTML = `<table>
    <thead><tr><th>Name</th><th>Type</th><th>Enabled</th><th>Actions</th></tr></thead>
    <tbody>${channels.map(c => `<tr>
      <td>${esc(c.name)}</td>
      <td>${c.type}</td>
      <td>${c.enabled ? 'Yes' : 'No'}</td>
      <td class="actions">
        <button class="btn btn-sm btn-success" onclick="testChannel('${c.id}')">Test</button>
        <button class="btn btn-sm btn-primary" onclick="editChannel('${c.id}')">Edit</button>
        <button class="btn btn-sm btn-danger" onclick="deleteChannel('${c.id}')">Delete</button>
      </td>
    </tr>`).join('')}</tbody></table>`;
}

function showChannelForm(c) {
  const isEdit = !!c;
  showModal(isEdit ? 'Edit Channel' : 'New Channel', `
    <div class="form-group"><label>Name</label><input id="cf-name" value="${isEdit ? esc(c.name) : ''}"></div>
    <div class="form-group"><label>Type</label>
      <select id="cf-type">
        <option value="slack" ${isEdit && c.type === 'slack' ? 'selected' : ''}>Slack</option>
        <option value="webhook" ${isEdit && c.type === 'webhook' ? 'selected' : ''}>Webhook</option>
      </select>
    </div>
    <div class="form-group"><label>Config (JSON)</label>
      <textarea id="cf-config">${isEdit ? JSON.stringify(JSON.parse(c.config || '{}'), null, 2) : '{"webhook_url":""}'}</textarea>
    </div>
    <div class="form-group"><label><input type="checkbox" id="cf-enabled" ${!isEdit || c.enabled ? 'checked' : ''}> Enabled</label></div>
    <div class="form-actions">
      <button class="btn btn-primary" onclick="saveChannel(${isEdit ? `'${c.id}'` : 'null'})">${isEdit ? 'Update' : 'Create'}</button>
      <button class="btn" onclick="hideModal()">Cancel</button>
    </div>
  `);
}

async function editChannel(id) {
  const c = await api(`/api/channels/${id}`);
  showChannelForm(c);
}

async function saveChannel(id) {
  const body = {
    name: document.getElementById('cf-name').value,
    type: document.getElementById('cf-type').value,
    config: JSON.parse(document.getElementById('cf-config').value),
    enabled: document.getElementById('cf-enabled').checked,
  };
  if (id) {
    await api(`/api/channels/${id}`, { method: 'PUT', body: JSON.stringify(body) });
  } else {
    await api('/api/channels', { method: 'POST', body: JSON.stringify(body) });
  }
  hideModal();
  loadChannels();
}

async function testChannel(id) {
  const result = await api(`/api/channels/${id}/test`, { method: 'POST' });
  if (result && result.error) {
    alert('Test failed: ' + result.error);
  } else {
    alert('Test notification sent!');
  }
}

async function deleteChannel(id) {
  if (!confirm('Delete this channel?')) return;
  await api(`/api/channels/${id}`, { method: 'DELETE' });
  loadChannels();
}

// Modal
function showModal(title, body) {
  document.getElementById('modal-container').innerHTML = `
    <div class="modal-overlay" onclick="if(event.target===this)hideModal()">
      <div class="modal"><h3>${title}</h3>${body}</div>
    </div>`;
}

function hideModal() {
  document.getElementById('modal-container').innerHTML = '';
}

// Helpers
function esc(s) {
  if (s == null) return '';
  const d = document.createElement('div');
  d.textContent = String(s);
  return d.innerHTML;
}

function opSym(op) {
  return { gt: '>', gte: '>=', lt: '<', lte: '<=', eq: '==', neq: '!=' }[op] || op;
}

function timeAgo(ts) {
  const s = Math.floor((Date.now() - new Date(ts).getTime()) / 1000);
  if (s < 60) return s + 's ago';
  if (s < 3600) return Math.floor(s / 60) + 'm ago';
  if (s < 86400) return Math.floor(s / 3600) + 'h ago';
  return Math.floor(s / 86400) + 'd ago';
}

// Initial load
loadAlerts();

// Auto-refresh alerts every 30s
setInterval(() => {
  const active = document.querySelector('#nav button.active');
  if (active && active.dataset.tab === 'alerts') loadAlerts();
}, 30000);
