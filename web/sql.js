'use strict';

const SAMPLE = `CREATE TABLE estudiantes (
  id INT PRIMARY KEY,
  nombre VARCHAR(50) NOT NULL,
  edad INT,
  promedio FLOAT,
  activo BOOLEAN,
  nacimiento DATE
);

INSERT INTO estudiantes VALUES
  (1, 'Ana', 20, 88.5, TRUE, '2005-03-14'),
  (2, 'Luis', 17, 71.0, TRUE, '2008-11-02'),
  (3, 'Marta', 22, NULL, FALSE, '2003-02-28');

CREATE INDEX idx_edad ON estudiantes (edad);

SELECT nombre FROM estudiantes WHERE edad >= 18 ORDER BY nombre;
SELECT activo, COUNT(*) AS total, AVG(promedio) FROM estudiantes GROUP BY activo;

-- Se borra al final para poder ejecutar el ejemplo otra vez.
DROP TABLE estudiantes;
`;

const TABS = [
  ['tokens', 'Tokens'],
  ['parse', 'Árbol sintáctico'],
  ['ast', 'AST'],
  ['errors', 'Errores'],
  ['results', 'Resultados'],
  ['schema', 'Esquema'],
];

const state = { tab: 'errors', report: null };
const $ = (id) => document.getElementById(id);

function el(tag, cls, text) {
  const n = document.createElement(tag);
  if (cls) n.className = cls;
  if (text !== undefined) n.textContent = text;
  return n;
}

function setStatus(msg) { $('status').textContent = msg; }

function renderTabs() {
  const tabs = $('tabs');
  tabs.replaceChildren();
  const errs = state.report ? state.report.diagnostics.length : 0;
  for (const [id, label] of TABS) {
    const b = el('button', id === state.tab ? 'active' : '', label);
    if (id === 'errors' && errs > 0) b.appendChild(el('span', 'badge', String(errs)));
    b.addEventListener('click', () => { state.tab = id; render(); });
    tabs.appendChild(b);
  }
}

function table(headers, rows) {
  const t = el('table');
  const head = el('tr');
  headers.forEach((h) => head.appendChild(el('th', '', h)));
  t.appendChild(head);
  rows.forEach((r) => {
    const tr = el('tr');
    r.forEach((c) => tr.appendChild(el('td', '', c === null ? 'NULL' : typeof c === 'boolean' ? (c ? 'TRUE' : 'FALSE') : String(c))));
    t.appendChild(tr);
  });
  return t;
}

function treeNode(n, depth) {
  const label = (parent) => {
    parent.appendChild(el('span', 'name', n.name));
    if (n.value) parent.appendChild(el('span', 'val', ' ' + n.value));
    if (n.line) parent.appendChild(el('span', 'muted', `  ${n.line}:${n.col}`));
  };
  if (!n.children || n.children.length === 0) {
    const leaf = el('div', 'leaf');
    label(leaf);
    return leaf;
  }
  const d = el('details');
  d.open = depth < 3;
  const s = el('summary');
  label(s);
  d.appendChild(s);
  n.children.forEach((c) => d.appendChild(treeNode(c, depth + 1)));
  return d;
}

function render() {
  renderTabs();
  const out = $('output');
  out.replaceChildren();
  const r = state.report;
  if (!r) { out.appendChild(el('div', 'muted', 'Analiza o ejecuta una sentencia para ver los resultados.')); return; }

  switch (state.tab) {
    case 'tokens':
      out.appendChild(table(['Línea', 'Col', 'Tipo', 'Texto'], r.tokens.map((t) => [t.line, t.col, t.type, t.text])));
      break;
    case 'parse':
      out.appendChild(treeNode(r.parseTree, 0));
      break;
    case 'ast':
      out.appendChild(treeNode(r.ast, 0));
      break;
    case 'errors':
      if (r.diagnostics.length === 0) {
        out.appendChild(el('div', 'ok', 'Sin errores léxicos, sintácticos ni semánticos.'));
      }
      r.diagnostics.forEach((d) => {
        const row = el('div', 'diag');
        row.appendChild(el('span', 'phase err', d.phase));
        const where = `línea ${d.line}, col ${d.col}` + (d.statement ? `, sentencia ${d.statement}` : '');
        row.appendChild(el('span', 'muted', where + ' — '));
        row.appendChild(el('span', '', d.message));
        out.appendChild(row);
      });
      break;
    case 'results':
      if (!r.executed) {
        out.appendChild(el('div', 'muted', r.diagnostics.length ? 'No se ejecutó: corrige los errores primero.' : 'Solo se analizó. Usa Ejecutar para aplicar las sentencias.'));
      }
      r.results.forEach((x) => {
        out.appendChild(el('div', 'ok', `[${x.statement}] ${x.message}`));
        if (x.columns) {
          out.appendChild(table(x.columns, x.rows || []));
        }
        (x.plan || []).forEach((p) => out.appendChild(el('div', 'muted', '  plan: ' + p)));
        out.appendChild(el('div', '', ' '));
      });
      break;
    case 'schema':
      if (r.schema.length === 0) out.appendChild(el('div', 'muted', 'La base de datos no tiene tablas.'));
      r.schema.forEach((t) => {
        out.appendChild(el('div', 'name', `${t.name}  (${t.rows >= 0 ? t.rows + ' filas' : 'filas no disponibles'})`));
        out.appendChild(table(['Columna', 'Tipo', 'PK', 'NOT NULL'], t.columns.map((c) => [c.name, c.type, c.primaryKey ? 'sí' : '', c.notNull ? 'sí' : ''])));
        out.appendChild(el('div', '', ' '));
      });
      (r.indexes || []).forEach((ix) => out.appendChild(el('div', 'muted', `índice ${ix.name} sobre ${ix.table}(${ix.column})`)));
      break;
  }
}

async function run(execute) {
  const source = $('source').value;
  setStatus(execute ? 'Ejecutando…' : 'Analizando…');
  try {
    const res = await fetch('/api/sql/run', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ source, execute }),
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || res.statusText);
    state.report = data;
    state.tab = data.diagnostics.length ? 'errors' : (execute ? 'results' : 'ast');
    render();
    const n = data.diagnostics.length;
    setStatus(n ? `${n} error(es).` : (execute ? 'Ejecución correcta.' : 'Análisis correcto.'));
  } catch (err) {
    setStatus('Error: ' + err.message);
  }
}

$('btn-run').addEventListener('click', () => run(true));
$('btn-analyze').addEventListener('click', () => run(false));
$('btn-sample').addEventListener('click', () => { $('source').value = SAMPLE; });
$('source').addEventListener('keydown', (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') { e.preventDefault(); run(!e.shiftKey); }
});

$('source').value = SAMPLE;
render();
