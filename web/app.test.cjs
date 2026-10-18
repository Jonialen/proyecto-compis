const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const source = fs.readFileSync(path.join(__dirname, 'app.js'), 'utf8');

function browser(response) {
  const elements = new Map();
  function node() {
    return {
      innerHTML: '', textContent: '', className: '', disabled: false, children: [], listeners: {},
      classList: { add() {}, remove() {}, toggle() {} }, style: {},
      appendChild(child) { this.children.push(child); return child; },
      replaceChildren(...children) { this.children = children; this.innerHTML = ''; this.textContent = ''; },
      addEventListener(name, fn) { this.listeners[name] = fn; },
      click() { return this.listeners.click?.(); }, focus() {}, setAttribute() {},
    };
  }
  const document = {
    addEventListener() {},
    createElement: node,
    getElementById(id) {
      if (!elements.has(id)) elements.set(id, node());
      return elements.get(id);
    },
    querySelectorAll(selector) {
      return selector.startsWith('#sidebar') ? [{ value: 'll1' }] : [];
    },
  };
  const context = vm.createContext({ document, console, fetch: async () => ({ ok: true, json: async () => response }) });
  vm.runInContext(source, context);
  vm.runInContext(`
    state.editors.yalp = { getValue: () => '%token INT\\n%%\\ns : INT ;' };
    state.editors.yal = { getValue: () => 'lexer' };
    state.editors.input = { getValue: () => '' };
    for (const key of ['yal', 'yalp', 'input', 'cps']) {
      const previous = state.editors[key];
      let value = previous ? previous.getValue() : 'let n = 1;';
      const listeners = [];
      state.editors[key] = {
        getValue: () => value,
        setValue: next => { value = next; listeners.forEach(fn => fn()); },
        on: (event, fn) => listeners.push(fn), refresh() {}, focus() {},
      };
    }
  `, context);
  return { context, elements };
}

for (const scenario of [
  { name: 'accepted empty input', methods: { ll1: { accepted: true } }, type: 'ok' },
  { name: 'table only', methods: { ll1: { accepted: null } }, type: 'ok' },
  { name: 'method build failure', methods: { ll1: { accepted: null, error: 'left recursion' } }, type: 'error' },
  { name: 'rejected input', methods: { ll1: { accepted: false, error: 'syntax error' } }, type: 'error' },
  { name: 'mixed method outcomes', methods: { ll1: { accepted: true }, slr: { accepted: null, error: 'conflict' } }, type: 'error' },
  { name: 'lexical failure', methods: { ll1: { accepted: true } }, lexical_errors: ['unknown character'], type: 'error' },
]) {
  test(`Run status: ${scenario.name}`, async () => {
    const { context, elements } = browser({ success: true, tokens: [], lexical_errors: [], ...scenario });
    await vm.runInContext('runAnalysis()', context);
    assert.equal(elements.get('status-text').className, scenario.type);
    assert.match(elements.get('status-text').textContent, scenario.type === 'ok' ? /Analysis complete/ : /completed with errors/);
    assert.equal(elements.get('btn-run').disabled, false);
  });
}

const hostile = '<img src=x onerror=alert(1)>';
const compilation = {
  tac: { instructions: [] },
  tacText: `main:\n  ${hostile}`,
  diagnostics: [],
  scopes: [],
  codegen: {
    unit: 'abstract-cell',
    symbols: [{
      id: hostile, scopeId: 1,
      symbol: {
        name: hostile, kind: 'variable', type: { kind: 'integer' },
        mutable: true, captured: false, span: { start: { line: 1, column: 2 } },
      },
      locations: [{ space: 'globals', offset: 0, size: 1, role: 'global' }],
      labels: [hostile],
    }],
    spaces: [{ id: 'globals', kind: 'globals', cells: 1, slots: [] }],
    labels: [{
      name: hostile, kind: 'procedure', owner: 'frame:f',
      span: { start: { line: 1, column: 2 } },
    }],
  },
};
function text(node) { return [node.textContent, ...node.children.map(text)].join(' '); }
function assertTextOnly(node) {
  assert.equal(node.innerHTML, '');
  node.children.forEach(assertTextOnly);
}
function deferred() { let resolve; const promise = new Promise(r => { resolve = r; }); return { promise, resolve }; }
function observeChanges(context) { vm.runInContext('bindSourceChanges()', context); }

test('Compile action uses server TAC and safely renders full layouts and symbols', async () => {
  const { context, elements } = browser(compilation);
  let sent;
  context.fetch = async (url, options) => { sent = { url, options }; return { ok: true, json: async () => compilation }; };
  vm.runInContext('bindRunButton(); state.activeEditor = "cps";', context);
  await elements.get('btn-run').click();
  assert.equal(sent.url, '/api/compiscript/compile');
  assert.deepEqual(JSON.parse(sent.options.body), { source: 'let n = 1;' });
  assert.equal(sent.options.method, 'POST');
  assert.equal(sent.options.headers['Content-Type'], 'application/json');
  assert.equal(text(elements.get('rp-tac')).trim(), compilation.tacText);
  for (const value of ['abstract-cell', hostile, 'globals', 'offset', 'size', 'global', 'frame:f', '1:2']) assert.ok(text(elements.get('rp-symbols')).includes(value), value);
  ['rp-source', 'rp-tac', 'rp-symbols', 'status-text'].forEach(id => assertTextOnly(elements.get(id)));
  assert.equal(elements.get('btn-run').disabled, false);
  assert.equal(elements.get('btn-compile').disabled, false);
});

test('Located typed language failures clear prior TAC and codegen', async () => {
  const { context, elements } = browser(compilation);
  await vm.runInContext('runCompiscript("compile")', context);
  context.fetch = async () => ({ ok: true, json: async () => ({ tac: null, tacText: '', diagnostics: [{ phase: 'semantic', code: 'SEM_TYPE', message: hostile, span: { start: { line: 2, column: 3 } } }], scopes: [] }) });
  await vm.runInContext('runCompiscript("compile")', context);
  assert.ok(!text(elements.get('rp-tac')).includes(hostile));
  assert.ok(!text(elements.get('rp-symbols')).includes(hostile));
  assert.match(text(elements.get('rp-diagnostics')), /semantic SEM_TYPE @ 2:3/);
  assert.ok(text(elements.get('rp-diagnostics')).includes(hostile));
  assertTextOnly(elements.get('rp-diagnostics'));
  assert.equal(elements.get('status-text').className, 'error');
});

for (const mode of ['http', 'nonJSON', 'network']) {
  test(`Transport ${mode} clears stale panels and restores controls`, async () => {
    const { context, elements } = browser(compilation);
    await vm.runInContext('runCompiscript("compile")', context);
    context.fetch = async () => {
      if (mode === 'network') throw Error(hostile);
      return { ok: false, status: 500, json: async () => { if (mode === 'nonJSON') throw Error('not JSON'); return { error: hostile }; } };
    };
    await vm.runInContext('runCompiscript("compile")', context);
    for (const id of ['rp-source', 'rp-ast', 'rp-diagnostics', 'rp-environments', 'rp-tac', 'rp-symbols']) {
      assert.ok(!text(elements.get(id)).includes('main:'));
      assertTextOnly(elements.get(id));
    }
    assertTextOnly(elements.get('status-text'));
    assert.equal(elements.get('status-text').className, 'error');
    assert.equal(elements.get('btn-run').disabled, false);
  });
}

test('Source edit, clear and sample invalidate late responses and reset results', async () => {
  const { context, elements } = browser(compilation);
  observeChanges(context);
  vm.runInContext('bindRunButton()', context);
  const old = deferred();
  context.fetch = () => old.promise;
  const first = vm.runInContext('runCompiscript("compile")', context);
  assert.equal(elements.get('btn-compile').disabled, true);
  vm.runInContext('state.editors.cps.setValue("let n = 2;")', context);
  assert.equal(elements.get('btn-compile').disabled, false);
  context.fetch = async () => ({ ok: true, json: async () => compilation });
  await vm.runInContext('runCompiscript("compile")', context);
  old.resolve({ ok: true, json: async () => ({ ...compilation, tacText: 'STALE' }) });
  await first;
  assert.ok(!text(elements.get('rp-tac')).includes('STALE'));
  elements.get('btn-clear-cps').click();
  assert.equal(vm.runInContext('state.editors.cps.getValue()', context), '');
  assert.ok(!text(elements.get('rp-tac')).includes('main:'));
  elements.get('btn-sample-cps').click();
  assert.match(vm.runInContext('state.editors.cps.getValue()', context), /foreach/);
});

test('Feature switch suppresses old request output without changing source', async () => {
  const { context, elements } = browser(compilation);
  const late = deferred(); context.fetch = () => late.promise;
  vm.runInContext('state.activeEditor = "cps"', context);
  const run = vm.runInContext('runCompiscript("compile")', context);
  vm.runInContext('switchEditorTab("yalp")', context);
  late.resolve({ ok: true, json: async () => compilation }); await run;
  assert.ok(!text(elements.get('rp-tac')).includes('main:'));
  assert.equal(vm.runInContext('state.activeEditor', context), 'yalp');
});

test('File reads capture YAPar targets and ignore superseded or edited source', () => {
  const { context, elements } = browser(compilation);
  const readers = [];
  context.FileReader = class { constructor() { readers.push(this); } readAsText() {} };
  observeChanges(context);
  vm.runInContext('bindSidebar()', context);
  const input = elements.get('file-input');
  const choose = (target, name, type = 'text/plain') => {
    elements.get(`load-${target}`).click(); input.files = [{ name, type }]; input.listeners.change();
    return readers.at(-1);
  };
  const yal = choose('yal', 'lexer.yal');
  const cps = choose('cps', hostile + '.cps');
  yal.onload({ target: { result: 'new lexer' } });
  assert.equal(vm.runInContext('state.editors.yal.getValue()', context), 'new lexer');
  cps.onload({ target: { result: 'let n = 3;' } });
  assert.equal(vm.runInContext('state.editors.cps.getValue()', context), 'let n = 3;');
  assert.ok(elements.get('status-text').textContent.includes(hostile)); assertTextOnly(elements.get('status-text'));
  const stale = choose('cps', 'old.cps');
  const fresh = choose('cps', 'new.cps');
  fresh.onload({ target: { result: 'new' } }); stale.onload({ target: { result: 'old' } });
  assert.equal(vm.runInContext('state.editors.cps.getValue()', context), 'new');
  const edit = choose('cps', 'edit.cps');
  vm.runInContext('state.editors.cps.setValue("typed")', context);
  edit.onload({ target: { result: 'overwritten' } });
  assert.equal(vm.runInContext('state.editors.cps.getValue()', context), 'typed');
  const count = readers.length;
  choose('cps', 'image.png', 'image/png');
  assert.equal(readers.length, count);
  assert.equal(vm.runInContext('state.editors.cps.getValue()', context), 'typed');
});

test('Legacy Compiscript Analyze remains available without TAC or layouts', async () => {
  const { context, elements } = browser({ ast: { kind: 'program', label: 'program', children: [] }, scopes: [], diagnostics: [] });
  let url;
  context.fetch = async target => { url = target; return { ok: true, json: async () => ({ ast: { kind: 'program' }, scopes: [], diagnostics: [] }) }; };
  await vm.runInContext('runCompiscript()', context);
  assert.equal(url, '/api/compiscript/analyze');
  assert.match(text(elements.get('rp-ast')), /program/);
});

test('An obsolete response cannot unlock controls during a newer request', async () => {
  const { context, elements } = browser(compilation);
  observeChanges(context);
  const old = deferred();
  context.fetch = () => old.promise;
  const first = vm.runInContext('runCompiscript("compile")', context);
  vm.runInContext('state.editors.cps.setValue("let n = 2;")', context);
  const fresh = deferred();
  context.fetch = () => fresh.promise;
  const second = vm.runInContext('runCompiscript("compile")', context);
  old.resolve({ ok: false, json: async () => ({ error: hostile }) });
  await first;
  assert.equal(elements.get('btn-compile').disabled, true);
  assert.equal(elements.get('status-text').className, 'loading');
  fresh.resolve({ ok: true, json: async () => compilation });
  await second;
  assert.equal(elements.get('btn-compile').disabled, false);
  assert.match(text(elements.get('rp-tac')), /main:/);
});

test('Late YAPar responses cannot replace Compiscript results', async () => {
  const { context, elements } = browser(compilation);
  const old = deferred();
  context.fetch = () => old.promise;
  const yapar = vm.runInContext('runAnalysis()', context);
  vm.runInContext('switchEditorTab("cps")', context);
  context.fetch = async () => ({ ok: true, json: async () => compilation });
  await vm.runInContext('runCompiscript("compile")', context);
  const status = elements.get('status-text').textContent;
  old.resolve({ ok: false, json: async () => ({ error: hostile }) });
  await yapar;
  assert.equal(elements.get('status-text').textContent, status);
  assert.equal(vm.runInContext('state.activeResult', context), 'tac');
});

test('Deferred YAPar modal work cannot start over newer Compiscript work', async () => {
  const { context, elements } = browser(compilation);
  const timers = [];
  context.setTimeout = (fn, delay) => timers.push({ fn, delay });
  vm.runInContext('processPayment(); switchEditorTab("cps")', context);
  let requests = 0;
  context.fetch = async () => { requests++; return { ok: true, json: async () => compilation }; };
  await vm.runInContext('runCompiscript("compile")', context);
  timers.find(timer => timer.delay === 3200).fn();
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(requests, 1);
  assert.equal(vm.runInContext('state.activeResult', context), 'tac');
  assert.equal(elements.get('status-text').className, 'ok');
});

test('Legacy YAPar modal still starts its unchanged analysis flow', async () => {
  const response = { success: true, methods: { ll1: { accepted: true } } };
  const { context, elements } = browser(response);
  const timers = [];
  context.setTimeout = (fn, delay) => timers.push({ fn, delay });
  let sent;
  context.fetch = async (url, options) => {
    sent = { url, options };
    return { ok: true, json: async () => response };
  };
  vm.runInContext('processPayment()', context);
  timers.find(timer => timer.delay === 3200).fn();
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(sent.url, '/api/process');
  assert.deepEqual(JSON.parse(sent.options.body), {
    yalex_content: 'lexer', yapar_content: '%token INT\n%%\ns : INT ;',
    input_content: '', methods: ['ll1'],
  });
  assert.equal(elements.get('btn-run').disabled, false);
  assert.equal(elements.get('status-text').className, 'ok');
});

test('Existing Save preserves source and filename while status remains text-safe', () => {
  const { context, elements } = browser(compilation);
  const exported = [];
  context.downloadText = (content, filename) => exported.push({ content, filename });
  context.filename = hostile + '.cps';
  vm.runInContext('state.activeEditor = "cps"; state.fileNames.cps = filename; saveCurrentFile()', context);
  assert.deepEqual(exported, [{ content: 'let n = 1;', filename: hostile + '.cps' }]);
  assert.equal(elements.get('status-text').textContent, 'Saved: ' + hostile + '.cps');
  assertTextOnly(elements.get('status-text'));
});

test('File read failures and legacy grammar/input targets retain safe behavior', () => {
  const { context, elements } = browser(compilation);
  const readers = [];
  context.FileReader = class {
    constructor() { readers.push(this); }
    readAsText() {}
  };
  observeChanges(context);
  vm.runInContext('bindSidebar()', context);
  const input = elements.get('file-input');
  for (const [target, name] of [['yalp', 'grammar.yalp'], ['input', 'input.txt']]) {
    elements.get('load-' + target).click();
    input.files = [{ name, type: 'text/plain' }];
    input.listeners.change();
    readers.at(-1).onload({ target: { result: target + ' content' } });
    assert.equal(vm.runInContext(`state.editors.${target}.getValue()`, context), target + ' content');
    assert.equal(vm.runInContext(`state.fileNames.${target}`, context), name);
  }
  elements.get('load-cps').click();
  input.files = [{ name: hostile + '.cps', type: '' }];
  input.listeners.change();
  const failed = readers.at(-1);
  failed.onerror();
  assert.equal(elements.get('status-text').textContent, 'Could not read: ' + hostile + '.cps');
  assertTextOnly(elements.get('status-text'));
  vm.runInContext('state.editors.cps.setValue("new typed source")', context);
  const currentStatus = elements.get('status-text').textContent;
  failed.onerror();
  assert.equal(elements.get('status-text').textContent, currentStatus);
  assert.equal(vm.runInContext('state.editors.cps.getValue()', context), 'new typed source');
});

for (const [kind, heading] of [['spec', 'Specification Error'], ['grammar', 'Grammar Error'], [undefined, 'Grammar / Spec Error']]) {
  test(`Error heading: ${kind || 'legacy'}`, async () => {
    const { context, elements } = browser({ success: false, grammar_error: '<bad>', grammar_error_kind: kind });
    await vm.runInContext('runAnalysis()', context);
    assert.match(elements.get('rp-summary').innerHTML, new RegExp(heading));
    assert.ok(elements.get('rp-summary').innerHTML.includes('&lt;bad&gt;'));
    assert.ok(!elements.get('rp-summary').innerHTML.includes('<bad>'));
    assert.equal(elements.get('status-text').className, 'error');
    assert.equal(elements.get('btn-run').disabled, false);
  });
}
