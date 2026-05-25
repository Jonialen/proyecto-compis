/* ═══════════════════════════════════════════════════════════════
   ParserIDE — Frontend
   ═══════════════════════════════════════════════════════════════ */

// ── State ─────────────────────────────────────────────────────────────────────
const state = {
  editors: { yal: null, yalp: null, input: null },
  fileNames: { yal: null, yalp: null, input: null },
  activeEditor: 'yalp',
  activeResult: 'summary',
  lastResponse: null,
  uwuMode: false,
  loading: false,
  vizInstance: null,
};

const UWU_SPARKLES = ['✨','⭐','🌸','💫','♡','✿','🌟','💖','🎀','🍬'];
const UWU_RUN_TEXTS   = ['Run nya~! (≧◡≦)', '▶ Desu!', '▶ Nyaa~ Run!', '▶ Let\'s go uwu!'];
const NORM_RUN_TEXT   = '▶ Run';

// ── Init ──────────────────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  initEditors();
  bindEditorTabs();
  bindResultsTabs();
  bindSidebar();
  bindRunButton();
  bindUWUButton();
  initPaymentModal();
  initViz();
  shortcutRun();
});

function initEditors() {
  const makeEditor = (id, placeholder) => {
    const panel = document.getElementById(`ep-${id}`);
    const ta = document.createElement('textarea');
    ta.placeholder = placeholder;
    panel.appendChild(ta);
    const cm = CodeMirror.fromTextArea(ta, {
      theme: 'dracula',
      lineNumbers: true,
      indentWithTabs: false,
      tabSize: 2,
      lineWrapping: false,
      autofocus: id === 'yalp',
    });
    cm.setSize('100%', '100%');
    state.editors[id] = cm;
  };

  makeEditor('yal',   '/* Load or paste your .yal lexer specification here */');
  makeEditor('yalp',  '/* Load or paste your .yalp grammar specification here */\n');
  makeEditor('input', '/* Load or paste the input string(s) to parse here */');
}

async function initViz() {
  if (typeof Viz !== 'undefined') {
    try {
      state.vizInstance = new Viz();
    } catch (e) {
      console.warn('Viz.js init failed:', e);
    }
  }
}

// ── Editor tabs ───────────────────────────────────────────────────────────────
function bindEditorTabs() {
  document.querySelectorAll('#editor-tabs .tab').forEach(btn => {
    btn.addEventListener('click', () => {
      const target = btn.dataset.editor;
      switchEditorTab(target);
    });
  });
}

function switchEditorTab(tab) {
  state.activeEditor = tab;
  document.querySelectorAll('#editor-tabs .tab').forEach(b =>
    b.classList.toggle('active', b.dataset.editor === tab));
  document.querySelectorAll('.editor-panel').forEach(p =>
    p.classList.toggle('active', p.id === `ep-${tab}`));
  state.editors[tab].refresh();
  state.editors[tab].focus();
}

// ── Results tabs ──────────────────────────────────────────────────────────────
function bindResultsTabs() {
  document.querySelectorAll('#results-tabs .tab').forEach(btn => {
    btn.addEventListener('click', () => {
      const target = btn.dataset.result;
      switchResultsTab(target);
    });
  });
}

function switchResultsTab(tab) {
  state.activeResult = tab;
  document.querySelectorAll('#results-tabs .tab').forEach(b =>
    b.classList.toggle('active', b.dataset.result === tab));
  document.querySelectorAll('.result-panel').forEach(p =>
    p.classList.toggle('active', p.id === `rp-${tab}`));

  // Lazy-render automaton SVG when the tab becomes visible
  if (tab === 'automaton' && state.lastResponse) {
    renderAutomatonFromState();
  }
}

// ── Sidebar: file loading & save ──────────────────────────────────────────────
function bindSidebar() {
  const fileInput = document.getElementById('file-input');
  let pendingTarget = null;

  document.getElementById('load-yal').addEventListener('click', () => {
    pendingTarget = 'yal';
    fileInput.accept = '.yal,.txt,text/plain';
    fileInput.click();
  });
  document.getElementById('load-yalp').addEventListener('click', () => {
    pendingTarget = 'yalp';
    fileInput.accept = '.yalp,.txt,text/plain';
    fileInput.click();
  });
  document.getElementById('load-input').addEventListener('click', () => {
    pendingTarget = 'input';
    fileInput.accept = '.txt,.expr,text/plain';
    fileInput.click();
  });

  fileInput.addEventListener('change', () => {
    const file = fileInput.files[0];
    if (!file || !pendingTarget) return;
    const reader = new FileReader();
    reader.onload = e => {
      state.editors[pendingTarget].setValue(e.target.result);
      state.fileNames[pendingTarget] = file.name;
      markFileLoaded(pendingTarget, file.name);
      switchEditorTab(pendingTarget);
      setStatus(`Loaded: ${file.name}`, 'ok');
    };
    reader.readAsText(file);
    fileInput.value = '';
  });

  document.getElementById('btn-save').addEventListener('click', saveCurrentFile);
}

function markFileLoaded(target, name) {
  const el = document.getElementById(`file-status-${target}`);
  if (!el) return;
  el.classList.add('loaded');
  const short = name.length > 22 ? name.slice(0, 20) + '…' : name;
  el.innerHTML = `<span class="fs-dot"></span> .${target === 'input' ? 'txt' : target} — ${short}`;
}

function saveCurrentFile() {
  const tab = state.activeEditor;
  const content = state.editors[tab].getValue();
  const name = state.fileNames[tab] || `file.${tab === 'input' ? 'txt' : tab}`;
  downloadText(content, name);
  setStatus(`Saved: ${name}`, 'ok');
}

function downloadText(text, filename) {
  const a = document.createElement('a');
  a.href = URL.createObjectURL(new Blob([text], { type: 'text/plain' }));
  a.download = filename;
  a.click();
  URL.revokeObjectURL(a.href);
}

// ── Run button ────────────────────────────────────────────────────────────────
function bindRunButton() {
  document.getElementById('btn-run').addEventListener('click', () => openPaymentModal());
}

function shortcutRun() {
  document.addEventListener('keydown', e => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault();
      openPaymentModal();
    }
  });
}

// ── Payment modal ─────────────────────────────────────────────────────────────
function openPaymentModal() {
  if (state.loading) return;

  const overlay = document.getElementById('payment-overlay');
  overlay.setAttribute('aria-hidden', 'false');
  overlay.classList.add('visible');

  // Reset form to fresh state every time
  resetPaymentForm();

  document.getElementById('cc-name').focus();
}

function closePaymentModal() {
  const overlay = document.getElementById('payment-overlay');
  overlay.classList.remove('visible');
  overlay.setAttribute('aria-hidden', 'true');
}

function resetPaymentForm() {
  const form = document.getElementById('payment-form');
  form.style.display = '';
  form.reset();
  ['name', 'number', 'expiry', 'cvv'].forEach(f => {
    const errEl = document.getElementById(`err-${f}`);
    if (errEl) errEl.textContent = '';
    const inputEl = document.getElementById(`cc-${f}`);
    if (inputEl) inputEl.classList.remove('valid', 'invalid');
  });
  document.getElementById('card-brand').textContent = '💳';

  // Remove any processing overlay left from a previous run
  const proc = document.getElementById('pm-processing-overlay');
  if (proc) proc.remove();

  const payBtn = document.getElementById('pm-pay-btn');
  if (payBtn) payBtn.disabled = false;
}

function initPaymentModal() {
  // Card number — format as XXXX XXXX XXXX XXXX while typing
  document.getElementById('cc-number').addEventListener('input', e => {
    let v = e.target.value.replace(/\D/g, '').slice(0, 16);
    e.target.value = v.replace(/(.{4})/g, '$1 ').trim();
    updateCardBrand(v);
  });

  // Expiry — auto-insert slash
  document.getElementById('cc-expiry').addEventListener('input', e => {
    let v = e.target.value.replace(/\D/g, '').slice(0, 4);
    if (v.length >= 3) v = v.slice(0, 2) + '/' + v.slice(2);
    e.target.value = v;
  });

  // CVV — digits only
  document.getElementById('cc-cvv').addEventListener('input', e => {
    e.target.value = e.target.value.replace(/\D/g, '').slice(0, 4);
  });

  // Form submit
  document.getElementById('payment-form').addEventListener('submit', e => {
    e.preventDefault();
    if (validatePaymentForm()) {
      processPayment();
    }
  });

  // Close button
  document.getElementById('pm-close').addEventListener('click', closePaymentModal);

  // Click outside to close
  document.getElementById('payment-overlay').addEventListener('click', e => {
    if (e.target === document.getElementById('payment-overlay')) closePaymentModal();
  });

  // Escape key
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape') closePaymentModal();
  });
}

const CARD_BRANDS = [
  { prefix: /^4/,          emoji: '💙', name: 'Visa' },
  { prefix: /^5[1-5]/,     emoji: '🟠', name: 'Mastercard' },
  { prefix: /^3[47]/,      emoji: '💚', name: 'Amex' },
  { prefix: /^6011|^65/,   emoji: '🔶', name: 'Discover' },
];

function updateCardBrand(digits) {
  const brand = CARD_BRANDS.find(b => b.prefix.test(digits));
  document.getElementById('card-brand').textContent = brand ? brand.emoji : '💳';
}

function validatePaymentForm() {
  let ok = true;

  // Name — not empty
  const name = document.getElementById('cc-name').value.trim();
  ok = setFieldValidity('cc-name', 'err-name',
    name.length > 0,
    'Please enter the cardholder name') && ok;

  // Card number — 16 digits
  const rawNum = document.getElementById('cc-number').value.replace(/\s/g, '');
  ok = setFieldValidity('cc-number', 'err-number',
    /^\d{16}$/.test(rawNum),
    'Card number must be exactly 16 digits') && ok;

  // Expiry — MM/YY, month 01-12, not obviously in the past (year >= current)
  const expiry = document.getElementById('cc-expiry').value;
  const expiryOk = validateExpiry(expiry);
  ok = setFieldValidity('cc-expiry', 'err-expiry',
    expiryOk.valid,
    expiryOk.msg) && ok;

  // CVV — 3 or 4 digits
  const cvv = document.getElementById('cc-cvv').value;
  ok = setFieldValidity('cc-cvv', 'err-cvv',
    /^\d{3,4}$/.test(cvv),
    'CVV must be 3 or 4 digits') && ok;

  return ok;
}

function validateExpiry(val) {
  if (!/^\d{2}\/\d{2}$/.test(val))
    return { valid: false, msg: 'Use MM/YY format' };

  const [mm, yy] = val.split('/').map(Number);
  if (mm < 1 || mm > 12)
    return { valid: false, msg: 'Month must be 01–12' };

  const now   = new Date();
  const curYY = now.getFullYear() % 100;
  const curMM = now.getMonth() + 1;
  if (yy < curYY || (yy === curYY && mm < curMM))
    return { valid: false, msg: 'Card is expired (even fake cards need a future date)' };

  return { valid: true, msg: '' };
}

function setFieldValidity(inputId, errId, isValid, msg) {
  const input = document.getElementById(inputId);
  const errEl = document.getElementById(errId);
  input.classList.toggle('valid',   isValid);
  input.classList.toggle('invalid', !isValid);
  errEl.textContent = isValid ? '' : msg;
  return isValid;
}

function processPayment() {
  const payBtn = document.getElementById('pm-pay-btn');
  payBtn.disabled = true;

  // Show a fake "processing" animation inside the modal
  const form = document.getElementById('payment-form');
  const proc = document.createElement('div');
  proc.id = 'pm-processing-overlay';
  proc.className = 'pm-processing';
  proc.innerHTML = `
    <span class="spinner"></span>
    <div class="pm-processing-text" id="pm-proc-text">Contacting ParsePay™ servers…</div>
  `;
  form.style.display = 'none';
  document.getElementById('payment-modal').appendChild(proc);

  // Fake steps for dramatic effect
  const steps = [
    [600,  'Verifying card details…'],
    [1100, 'Authorizing $0.99…'],
    [1700, 'Charging your card (lol jk)…'],
    [2300, 'Unlocking premium results…'],
    [2700, '✅ Payment approved! (not really)'],
  ];

  steps.forEach(([delay, text]) => {
    setTimeout(() => {
      const el = document.getElementById('pm-proc-text');
      if (el) el.textContent = text;
    }, delay);
  });

  // After the show, close modal and actually run the analysis
  setTimeout(() => {
    closePaymentModal();
    runAnalysis();
  }, 3200);
}

async function runAnalysis() {
  if (state.loading) return;

  const yaparContent = state.editors.yalp.getValue().trim();
  if (!yaparContent) {
    setStatus('Error: .yalp content is empty', 'error');
    flashPanel('rp-summary', renderErrorBox('Please load or type a .yalp grammar specification.'));
    switchResultsTab('summary');
    return;
  }

  const selectedMethods = [...document.querySelectorAll('#sidebar input[type=checkbox]:checked')]
    .map(cb => cb.value);
  if (selectedMethods.length === 0) {
    setStatus('Error: select at least one method', 'error');
    return;
  }

  setLoading(true);
  setStatus('<span class="spinner"></span> Analysing…', 'loading');

  const body = {
    yalex_content: state.editors.yal.getValue(),
    yapar_content: yaparContent,
    input_content: state.editors.input.getValue(),
    methods: selectedMethods,
  };

  try {
    const res = await fetch('/api/process', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    const data = await res.json();

    if (!res.ok) {
      throw new Error(data.error || `HTTP ${res.status}`);
    }

    state.lastResponse = data;
    displayResults(data);

    const hasAny = data.success;
    setStatus(hasAny ? '✓ Analysis complete' : '⚠ Analysis completed with errors', hasAny ? 'ok' : 'error');
  } catch (err) {
    setStatus('Error: ' + err.message, 'error');
    flashPanel('rp-summary', renderErrorBox(err.message));
    switchResultsTab('summary');
  } finally {
    setLoading(false);
  }
}

// ── Result rendering ──────────────────────────────────────────────────────────
function displayResults(data) {
  renderSummary(data);
  renderTokensPanel(data.tokens || [], data.lexical_errors || []);
  renderMethodTable('ll1',  data.methods?.ll1);
  renderMethodTable('slr',  data.methods?.slr);
  renderMethodTable('lalr', data.methods?.lalr);

  // Automaton is lazy-rendered when the tab is active
  if (state.activeResult === 'automaton') {
    renderAutomatonFromState();
  }

  switchResultsTab('summary');
}

function renderSummary(data) {
  const el = document.getElementById('rp-summary');
  let html = '';

  if (data.grammar_error) {
    html += renderErrorBox('Grammar / Spec Error\n\n' + data.grammar_error);
  }

  if (data.lexical_errors && data.lexical_errors.length > 0) {
    html += `<div class="lexer-error-box">
      <strong>Lexical Errors (${data.lexical_errors.length})</strong><br>
      ${escHtml(data.lexical_errors.join('\n'))}
    </div>`;
  }

  const methodOrder = ['ll1', 'slr', 'lalr'];
  const cards = methodOrder.map(key => {
    const r = data.methods?.[key];
    if (!r) return '';
    const label = { ll1: 'LL(1)', slr: 'SLR(1)', lalr: 'LALR' }[key];
    let badgeClass = 'pending';
    let badgeText  = 'No Input';
    if (r.error && r.accepted === null) {
      badgeClass = 'error'; badgeText = 'Error';
    } else if (r.accepted !== null && r.accepted !== undefined) {
      badgeClass = r.accepted ? 'accepted' : 'rejected';
      badgeText  = r.accepted ? 'Accepted ✓' : 'Rejected ✗';
    } else if (!r.error) {
      badgeClass = 'pending'; badgeText = 'Table Ready';
    }
    const errLine = r.error
      ? `<div class="mc-error-text">⚠ ${escHtml(r.error)}</div>`
      : '';
    return `<div class="method-card">
      <div>
        <div class="mc-name">${label}</div>
        ${errLine}
      </div>
      <span class="mc-badge ${badgeClass}">${badgeText}</span>
    </div>`;
  }).join('');

  html += `<div class="result-section-title">Parse Methods</div>
           <div class="summary-cards">${cards || '<div style="color:var(--text-muted)">No results yet.</div>'}</div>`;

  if ((data.tokens || []).length > 0) {
    html += `<div class="result-section-title" style="margin-top:16px">Tokens (${data.tokens.length})</div>
             <div style="font-size:11px;color:var(--text-secondary)">
               Switch to the <strong>Tokens</strong> tab for the full token table.
             </div>`;
  }

  el.innerHTML = html;
}

function renderTokensPanel(tokens, lexErrors) {
  const el = document.getElementById('rp-tokens');
  if (!tokens.length && !lexErrors.length) {
    el.innerHTML = '<div class="empty-state"><div class="empty-icon">🏷️</div>No input was tokenized.</div>';
    return;
  }
  let html = '';
  if (lexErrors.length) {
    html += `<div class="lexer-error-box"><strong>Lexical Errors:</strong><br>${escHtml(lexErrors.join('\n'))}</div>`;
  }
  if (tokens.length) {
    html += `<div class="result-section-title">${tokens.length} Tokens</div>
    <div class="tbl-wrap">
    <table class="data-table">
      <thead><tr><th>#</th><th>Line</th><th>Type</th><th>Lexeme</th></tr></thead>
      <tbody>`;
    tokens.forEach((tok, i) => {
      html += `<tr>
        <td style="color:var(--text-muted)">${i + 1}</td>
        <td>${tok.line}</td>
        <td style="color:var(--accent-2)">${escHtml(tok.type)}</td>
        <td>${escHtml(tok.lexeme)}</td>
      </tr>`;
    });
    html += '</tbody></table></div>';
  }
  el.innerHTML = html;
}

function renderMethodTable(key, result) {
  const el = document.getElementById(`rp-${key}`);
  const label = { ll1: 'LL(1)', slr: 'SLR(1)', lalr: 'LALR' }[key];

  if (!result) {
    el.innerHTML = `<div class="empty-state"><div class="empty-icon">📊</div>${label} was not requested.</div>`;
    return;
  }

  if (result.error && !result.table_json) {
    el.innerHTML = renderErrorBox(`${label} Error\n\n${result.error}`);
    return;
  }

  let html = '';
  if (result.error) {
    html += `<div class="lexer-error-box">⚠ ${escHtml(result.error)}</div>`;
  }

  if (result.table_json) {
    try {
      const tbl = typeof result.table_json === 'string'
        ? JSON.parse(result.table_json)
        : result.table_json;
      html += `<div class="result-section-title">${label} Parsing Table</div>`;
      html += buildParseTableHTML(tbl);
    } catch (e) {
      html += renderErrorBox('Failed to parse table JSON: ' + e.message);
    }
  } else {
    html += `<div class="empty-state"><div class="empty-icon">📊</div>No table data.</div>`;
  }

  el.innerHTML = html;
}

function buildParseTableHTML(tbl) {
  // tbl: { method, terminals[], non_terminals[], states[{id, actions, gotos}] }
  const terms   = tbl.terminals   || [];
  const nonterms = tbl.non_terminals || [];
  const states  = tbl.states      || [];

  let html = '<div class="parse-table-wrap"><table class="parse-table">';

  // Header row 1 — section labels
  html += '<thead><tr>';
  html += '<th rowspan="2" class="state-id">State</th>';
  if (terms.length)    html += `<th colspan="${terms.length}" class="th-section">ACTION</th>`;
  if (nonterms.length) html += `<th colspan="${nonterms.length}" class="th-section">GOTO</th>`;
  html += '</tr><tr>';
  terms.forEach(t    => { html += `<th title="${escHtml(t)}">${escHtml(t)}</th>`; });
  nonterms.forEach(nt => { html += `<th title="${escHtml(nt)}">${escHtml(nt)}</th>`; });
  html += '</tr></thead><tbody>';

  states.forEach(s => {
    html += '<tr>';
    html += `<td class="state-id">${s.id}</td>`;
    terms.forEach(t => {
      const val = (s.actions || {})[t] || '';
      const cls = cellClass(val);
      html += `<td class="${cls}">${escHtml(val)}</td>`;
    });
    nonterms.forEach(nt => {
      const val = (s.gotos || {})[nt];
      html += val !== undefined
        ? `<td class="goto-v">${val}</td>`
        : `<td></td>`;
    });
    html += '</tr>';
  });

  html += '</tbody></table></div>';
  return html;
}

function cellClass(val) {
  if (!val) return '';
  if (val.startsWith('s'))   return 'shift';
  if (val.startsWith('r'))   return 'reduce';
  if (val === 'acc')          return 'accept';
  return '';
}

async function renderAutomatonFromState() {
  const el = document.getElementById('rp-automaton');
  const data = state.lastResponse;

  // Prefer SLR automaton, then LALR, then LL1 (which has no LR0 automaton)
  const dot = data?.methods?.slr?.automaton_dot
           || data?.methods?.lalr?.automaton_dot
           || data?.methods?.ll1?.automaton_dot;

  if (!dot) {
    el.innerHTML = '<div class="empty-state"><div class="empty-icon">🔵</div>No LR(0) automaton available (requires SLR or LALR method).</div>';
    return;
  }

  if (!state.vizInstance) {
    el.innerHTML = renderErrorBox('Viz.js not loaded — cannot render automaton.');
    return;
  }

  el.innerHTML = '<div class="empty-state"><span class="spinner"></span> Rendering automaton…</div>';

  try {
    const svg = await state.vizInstance.renderSVGElement(dot);

    // Make the SVG responsive
    svg.setAttribute('width',  '100%');
    svg.setAttribute('height', 'auto');
    svg.removeAttribute('width');
    svg.removeAttribute('height');

    const wrap = document.createElement('div');
    wrap.className = 'automaton-svg-wrap';
    wrap.appendChild(svg);

    el.innerHTML = '<div class="result-section-title">LR(0) Automaton</div>';
    el.appendChild(wrap);
  } catch (err) {
    el.innerHTML = renderErrorBox('Automaton render error: ' + err.message);
  }
}

// ── UWU mode ──────────────────────────────────────────────────────────────────
function bindUWUButton() {
  document.getElementById('btn-uwu').addEventListener('click', toggleUWU);
}

function toggleUWU() {
  state.uwuMode = !state.uwuMode;
  document.body.classList.toggle('uwu-mode', state.uwuMode);

  const runBtn = document.getElementById('btn-run');
  if (state.uwuMode) {
    runBtn.innerHTML = UWU_RUN_TEXTS[Math.floor(Math.random() * UWU_RUN_TEXTS.length)];
    setStatus('UWU mode activated! (≧◡≦) ♡', 'ok');
    startSparkles();
    applyUWUEditorTheme();
  } else {
    runBtn.innerHTML = `<span class="btn-icon">▶</span> ${NORM_RUN_TEXT}`;
    setStatus('Back to normal mode.', 'ok');
    stopSparkles();
    applyNormalEditorTheme();
  }
}

function applyUWUEditorTheme() {
  Object.values(state.editors).forEach(cm => cm.setOption('theme', 'default'));
}
function applyNormalEditorTheme() {
  Object.values(state.editors).forEach(cm => cm.setOption('theme', 'dracula'));
}

// ── Sparkles ──────────────────────────────────────────────────────────────────
let sparkleInterval = null;

function startSparkles() {
  const container = document.getElementById('sparkle-container');
  container.innerHTML = '';

  // Create static background sparkles
  for (let i = 0; i < 18; i++) {
    addSparkle(container, true);
  }

  // Periodically add new ones
  sparkleInterval = setInterval(() => {
    if (document.getElementById('sparkle-container').children.length < 30) {
      addSparkle(container, false);
    }
  }, 600);
}

function stopSparkles() {
  clearInterval(sparkleInterval);
  sparkleInterval = null;
  document.getElementById('sparkle-container').innerHTML = '';
}

function addSparkle(container, isStatic) {
  const el = document.createElement('span');
  el.className = 'sparkle';
  el.textContent = UWU_SPARKLES[Math.floor(Math.random() * UWU_SPARKLES.length)];
  const dur   = (2 + Math.random() * 4).toFixed(1) + 's';
  const delay = isStatic ? (Math.random() * 4).toFixed(1) + 's' : '0s';
  el.style.cssText = `
    left: ${Math.random() * 100}%;
    top:  ${20 + Math.random() * 75}%;
    --dur: ${dur};
    --delay: ${delay};
    font-size: ${12 + Math.random() * 14}px;
  `;
  container.appendChild(el);
  if (!isStatic) {
    el.addEventListener('animationend', () => el.remove(), { once: true });
  }
}

// ── UI helpers ────────────────────────────────────────────────────────────────
function setLoading(on) {
  state.loading = on;
  document.getElementById('btn-run').disabled = on;
}

function setStatus(html, type = '') {
  const el = document.getElementById('status-text');
  el.innerHTML = html;
  el.className = type;
}

function flashPanel(panelId, html) {
  const el = document.getElementById(panelId);
  if (el) el.innerHTML = html;
}

function renderErrorBox(msg) {
  return `<div class="grammar-error-box">${escHtml(msg)}</div>`;
}

function escHtml(str) {
  if (str === null || str === undefined) return '';
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}
