const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const source = fs.readFileSync(path.join(__dirname, 'app.js'), 'utf8');

function browser(response) {
  const elements = new Map();
  const document = {
    addEventListener() {},
    getElementById(id) {
      if (!elements.has(id)) elements.set(id, { innerHTML: '', className: '', disabled: false });
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
    assert.match(elements.get('status-text').innerHTML, scenario.type === 'ok' ? /Analysis complete/ : /completed with errors/);
    assert.equal(elements.get('btn-run').disabled, false);
  });
}

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
