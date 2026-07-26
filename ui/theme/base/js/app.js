// loadOpenApiDoc (openapi.js) and LANGUAGES/generateSnippet (snippet.js) are
// loaded as globals via plain <script> tags before this file, in that order.

const SPEC_URL = window.__DOCS_DOCUMENT_URL__ || 'https://s3.thebeanfamily.org/b/base/petstore.json';

const els = {
  sidebarGroups: document.getElementById('sidebar-groups'),
  main: document.getElementById('doc-main'),
  tryit: document.getElementById('tryit-panel'),
  specVersion: document.getElementById('spec-version'),
  sidebarToggle: document.getElementById('sidebar-toggle'),
  sidebar: document.getElementById('sidebar'),
  authBtn: document.getElementById('btn-authorize'),
  authBtnLabel: document.getElementById('btn-authorize-label'),
  authDialog: document.getElementById('auth-dialog'),
  authDialogBody: document.getElementById('auth-dialog-body'),
  themeButton: document.getElementById('theme-setting'),
};

/** doc: normalized OpenAPI model. tryState: Map<opId, editable request state>. */
let doc = null;
const tryState = new Map();
let currentLang = 'curl';

const THEME_STORAGE_KEY = 'base-docs-theme';

// ---------- oauth2 authorize flow ----------
// Popup landing page is oauth2-redirect.html/js at the site root; it reads
// the token/code the auth server put in the URL and posts it back to us.
const OAUTH_REDIRECT_URI = new URL('oauth2-redirect.html', document.baseURI).href;
const OAUTH_FLOW_LABELS = {
  implicit: 'Implicit',
  password: 'Resource Owner Password Credentials',
  application: 'Client Credentials',
  accessCode: 'Authorization Code',
};
/** pendingOAuth: Map<state, { key, flow, clientId, clientSecret, popup }> awaiting the popup's postMessage. */
const pendingOAuth = new Map();
/** authValues: Map<securityDefinitionKey, { clientId, clientSecret, username, password }> — oauth2 flow inputs only; the actual credential values live in sessionStorage under 'apikey' / 'oauthkey'. */
const authValues = new Map();

/**
 * Returns the currently selected server base URL for the active operation.
 * Falls back to the document default when no operation is selected yet.
 */
function currentSelectedBaseUrl() {
  const op = doc?.findOperation?.(currentOperationFromHash());
  return op ? getOpState(op).baseUrl : doc?.baseUrl || window.location.origin;
}

/**
 * Resolves OAuth endpoint URLs against the selected server when the spec URL is
 * relative or bound to the legacy derived docs host.
 */
function resolveOAuthEndpointUrl(rawUrl, selectedBaseUrl) {
  if (!rawUrl) return '';

  const baseUrl = selectedBaseUrl || currentSelectedBaseUrl();
  try {
    const base = new URL(baseUrl);
    const legacy = new URL(doc?.derivedBaseUrl || doc?.baseUrl || baseUrl);
    const resolved = new URL(rawUrl, `${base.href.replace(/\/$/, '')}/`);

    if (resolved.host === legacy.host) {
      resolved.protocol = base.protocol;
      resolved.username = base.username;
      resolved.password = base.password;
      resolved.host = base.host;
    }

    return resolved.href;
  } catch {
    return String(rawUrl);
  }
}

function getStoredAuth(name) {
  return sessionStorage.getItem(name) || '';
}

function setStoredAuth(name, value) {
  if (value) sessionStorage.setItem(name, value);
  else sessionStorage.removeItem(name);
}

/**
 * Returns whether a session-backed auth field is currently locked by a saved
 * value and should render as read-only until cleared by the user.
 */
function isStoredAuthLocked(name) {
  return !!getStoredAuth(name);
}

function hasAnyStoredAuth() {
  return !!(getStoredAuth('apikey') || getStoredAuth('oauthkey'));
}

function getPreferredTheme() {
  const stored = localStorage.getItem(THEME_STORAGE_KEY);
  if (stored === 'light' || stored === 'dark') return stored;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function applyTheme(theme) {
  document.body.dataset.theme = theme;
  if (!els.themeButton) return;
  const nextTheme = theme === 'dark' ? 'light' : 'dark';
  els.themeButton.setAttribute('aria-pressed', String(theme === 'dark'));
  els.themeButton.setAttribute('aria-label', `Switch to ${nextTheme} mode`);
  els.themeButton.title = `Switch to ${nextTheme} mode`;
}

function toggleTheme() {
  const nextTheme = document.body.dataset.theme === 'dark' ? 'light' : 'dark';
  localStorage.setItem(THEME_STORAGE_KEY, nextTheme);
  applyTheme(nextTheme);
  initMermaid();
  route();
}

// ---------- tiny helpers ----------

function escapeHtml(str) {
  return String(str ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

function assetUrl(path) {
  const base = window.__ASSET_BASE_PATH__ || '';
  const cleanPath = String(path || '').replace(/\\/g, '/').replace(/^\/+/, '');
  return `${base}/${cleanPath}`;
}

// Inline markdown: **bold**, __bold__, *italic*, `code`, [text](url), ![alt](url).
function renderInline(text) {
  let html = escapeHtml(text);
  html = html.replace(/!\[([^\]]*)\]\(([^)\s]+)\)/g, '<img src="$2" alt="$1" loading="lazy">');
  html = html.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>');
  html = html.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
  html = html.replace(/__([^_]+)__/g, '<strong>$1</strong>');
  html = html.replace(/(^|[^*])\*([^*\n]+)\*(?!\*)/g, '$1<em>$2</em>');
  html = html.replace(/`([^`]+)`/g, '<code>$1</code>');
  return html;
}

function splitTableRow(line) {
  let t = line.trim();
  if (t.startsWith('|')) t = t.slice(1);
  if (t.endsWith('|')) t = t.slice(0, -1);
  return t.split('|').map(c => c.trim());
}

// Small GFM-flavored markdown renderer: headers, lists, tables, blockquotes,
// fenced code blocks (```mermaid fences become <pre class="mermaid"> for the
// CDN-loaded mermaid.js runtime to pick up), rules, and inline formatting.
function mdLite(src) {
  if (!src) return '';
  const lines = String(src).replace(/\r\n?/g, '\n').split('\n');
  const out = [];
  let paragraphBuf = [];
  let i = 0;

  function flushParagraph() {
    if (!paragraphBuf.length) return;
    out.push(`<p>${renderInline(paragraphBuf.join('\n')).replace(/\n/g, '<br>')}</p>`);
    paragraphBuf = [];
  }

  while (i < lines.length) {
    const line = lines[i];

    const fence = line.match(/^\s*```\s*([\w-]*)\s*$/);
    if (fence) {
      flushParagraph();
      const lang = fence[1] || '';
      const codeLines = [];
      i++;
      while (i < lines.length && !/^\s*```\s*$/.test(lines[i])) {
        codeLines.push(lines[i]);
        i++;
      }
      i++;
      const code = codeLines.join('\n');
      if (lang === 'mermaid') {
        out.push(`<pre class="mermaid">${escapeHtml(code)}</pre>`);
      } else {
        out.push(`<pre class="code-block"${lang ? ` data-lang="${escapeHtml(lang)}"` : ''}><code>${escapeHtml(code)}</code></pre>`);
      }
      continue;
    }

    if (/^\s*$/.test(line)) {
      flushParagraph();
      i++;
      continue;
    }

    const header = line.match(/^(#{1,6})\s+(.*)$/);
    if (header) {
      flushParagraph();
      const level = header[1].length;
      out.push(`<h${level}>${renderInline(header[2].trim())}</h${level}>`);
      i++;
      continue;
    }

    if (/^\s*([-*_])\s*(\1\s*){2,}$/.test(line)) {
      flushParagraph();
      out.push('<hr>');
      i++;
      continue;
    }

    if (/^\s*>\s?/.test(line)) {
      flushParagraph();
      const quoteLines = [];
      while (i < lines.length && /^\s*>\s?/.test(lines[i])) {
        quoteLines.push(lines[i].replace(/^\s*>\s?/, ''));
        i++;
      }
      out.push(`<blockquote>${mdLite(quoteLines.join('\n'))}</blockquote>`);
      continue;
    }

    if (/^\s*\|.*\|\s*$/.test(line) && lines[i + 1] && /^\s*\|?[\s:|-]+\|?\s*$/.test(lines[i + 1])) {
      flushParagraph();
      const headCells = splitTableRow(line);
      const aligns = splitTableRow(lines[i + 1]).map(cell => {
        if (/^:-+:$/.test(cell)) return 'center';
        if (/^-+:$/.test(cell)) return 'right';
        if (/^:-+$/.test(cell)) return 'left';
        return '';
      });
      i += 2;
      const bodyRows = [];
      while (i < lines.length && /^\s*\|.*\|\s*$/.test(lines[i])) {
        bodyRows.push(splitTableRow(lines[i]));
        i++;
      }
      const cellStyle = (idx) => (aligns[idx] ? ` style="text-align:${aligns[idx]}"` : '');
      const thead = `<thead><tr>${headCells.map((c, idx) => `<th${cellStyle(idx)}>${renderInline(c)}</th>`).join('')}</tr></thead>`;
      const tbody = `<tbody>${bodyRows.map(row => `<tr>${row.map((c, idx) => `<td${cellStyle(idx)}>${renderInline(c)}</td>`).join('')}</tr>`).join('')}</tbody>`;
      out.push(`<div class="table-wrap"><table>${thead}${tbody}</table></div>`);
      continue;
    }

    if (/^\s*[-*+]\s+/.test(line)) {
      flushParagraph();
      const items = [];
      while (i < lines.length && /^\s*[-*+]\s+/.test(lines[i])) {
        items.push(lines[i].replace(/^\s*[-*+]\s+/, ''));
        i++;
      }
      out.push(`<ul>${items.map(it => `<li>${renderInline(it)}</li>`).join('')}</ul>`);
      continue;
    }

    if (/^\s*\d+[.)]\s+/.test(line)) {
      flushParagraph();
      const items = [];
      while (i < lines.length && /^\s*\d+[.)]\s+/.test(lines[i])) {
        items.push(lines[i].replace(/^\s*\d+[.)]\s+/, ''));
        i++;
      }
      out.push(`<ol>${items.map(it => `<li>${renderInline(it)}</li>`).join('')}</ol>`);
      continue;
    }

    paragraphBuf.push(line.trim());
    i++;
  }
  flushParagraph();
  return out.join('');
}

// ---------- mermaid (CDN) ----------

function mermaidTheme() {
  return document.body.dataset.theme === 'dark' ? 'dark' : 'default';
}

function initMermaid() {
  if (!window.mermaid) return;
  window.mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme: mermaidTheme() });
}

function renderMermaidDiagrams() {
  if (!window.mermaid) return;
  const nodes = els.main.querySelectorAll('pre.mermaid');
  if (!nodes.length) return;
  nodes.forEach(node => node.removeAttribute('data-processed'));
  window.mermaid.run({ nodes }).catch(err => console.error('Mermaid render failed:', err));
}

function methodBadge(method) {
  const label = method === 'DELETE' ? 'DEL' : method;
  return `<span class="badge method-badge" data-method="${method}">${label}</span>`;
}

function statusClass(status) {
  const n = parseInt(status, 10);
  if (status === 'default' || Number.isNaN(n)) return 'default';
  if (n < 300) return '2xx';
  if (n < 400) return '3xx';
  if (n < 500) return '4xx';
  return '5xx';
}

function copyText(text, btn) {
  navigator.clipboard?.writeText(text).then(() => {
    if (!btn) return;
    const original = btn.textContent;
    btn.textContent = 'Copied!';
    setTimeout(() => { btn.textContent = original; }, 1200);
  });
}

function uid(prefix) {
  return `${prefix}-${Math.random().toString(36).slice(2, 9)}`;
}

function randomState() {
  return Array.from(crypto.getRandomValues(new Uint8Array(16)), b => b.toString(16).padStart(2, '0')).join('');
}

function toggleSidebar() {
  if (isSidebarOpen()) closeSidebar();
  else openSidebar();
}

/**
 * Returns whether the sidebar is currently acting as an overlay drawer.
 * Overlay behavior is only active below the desktop breakpoint.
 */
function isSidebarOverlayViewport() {
  return window.matchMedia('(max-width: 1100px)').matches;
}

/**
 * Returns whether the sidebar drawer is currently visible.
 * Desktop always reports open because the sidebar stays pinned there.
 */
function isSidebarOpen() {
  return els.sidebar.getAttribute('aria-hidden') !== 'true';
}

/**
 * Opens the sidebar and synchronizes the toggle button accessibility state.
 * Used by the mobile/tablet drawer behavior only.
 */
function openSidebar() {
  els.sidebar.setAttribute('aria-hidden', 'false');
  els.sidebarToggle.setAttribute('aria-expanded', 'true');
}

/**
 * Closes the sidebar and synchronizes the toggle button accessibility state.
 * No-op for desktop because the sidebar is meant to stay visible there.
 */
function closeSidebar() {
  if (!isSidebarOverlayViewport()) return;
  els.sidebar.setAttribute('aria-hidden', 'true');
  els.sidebarToggle.setAttribute('aria-expanded', 'false');
}

function syncSidebarForViewport() {
  const isDesktop = window.matchMedia('(min-width: 1101px)').matches;
  els.sidebar.setAttribute('aria-hidden', isDesktop ? 'false' : 'true');
  els.sidebarToggle.setAttribute('aria-expanded', isDesktop ? 'true' : 'false');
}

/**
 * Closes the overlay sidebar after a navigation selection.
 * Desktop keeps the pinned sidebar visible.
 */
function closeSidebarAfterSelection() {
  closeSidebar();
}

// ---------- sidebar ----------

function renderSidebar() {
  const items = [];
  items.push(`<ul class="sidebar-root">`);
  items.push(`<li><a href="#/introduction" class="sidebar-intro-link" data-op-id="introduction"><span>Introduction</span></a></li>`);

  // Tag `kind` (3.2) buckets top-level groups; tags without a `parent` render
  // at the root, tags with one are only shown nested under their parent.
  const topLevelGroups = doc.groups.filter(g => !g.parent);
  const childGroups = (parentName) => doc.groups.filter(g => g.parent === parentName);

  function renderGroup(group) {
    const detailsId = uid('grp');
    const children = childGroups(group.name);
    return `
      <li class="sidebar-group-wrap">
        <details id="${detailsId}" class="sidebar-group" open>
          <summary aria-controls="${detailsId}-content">
            <span class="sidebar-group-label">${escapeHtml(group.name)}${group.kind ? ` <span class="text-muted">(${escapeHtml(group.kind)})</span>` : ''}</span>
            <span class="sidebar-group-arrow">${chevronSvg()}</span>
          </summary>
          <ul id="${detailsId}-content" class="sidebar-group-items">
            ${group.items.map(op => `
              <li>
                <a href="#/${op.id}" class="sidebar-op-link" data-op-id="${op.id}">
                  ${methodBadge(op.method)}
                  <span>${escapeHtml(op.summary)}</span>
                </a>
              </li>
            `).join('')}
            ${children.map(child => renderGroup(child)).join('')}
          </ul>
        </details>
      </li>
    `;
  }

  for (const group of topLevelGroups) items.push(renderGroup(group));

  if (doc.webhooks && doc.webhooks.length) {
    const detailsId = uid('grp');
    items.push(`
      <li class="sidebar-group-wrap">
        <details id="${detailsId}" class="sidebar-group" open>
          <summary aria-controls="${detailsId}-content">
            <span class="sidebar-group-label">Webhooks</span>
            <span class="sidebar-group-arrow">${chevronSvg()}</span>
          </summary>
          <ul id="${detailsId}-content" class="sidebar-group-items">
            ${doc.webhooks.map(op => `
              <li>
                <a href="#/${op.id}" class="sidebar-op-link" data-op-id="${op.id}">
                  ${methodBadge(op.method)}
                  <span>${escapeHtml(op.summary)}</span>
                </a>
              </li>
            `).join('')}
          </ul>
        </details>
      </li>
    `);
  }
  items.push(`</ul>`);
  els.sidebarGroups.innerHTML = items.join('');
}

function setActiveSidebarItem(opId) {
  els.sidebarGroups.querySelectorAll('a[data-op-id]').forEach(a => {
    a.classList.toggle('active', a.dataset.opId === opId);
  });
}

// ---------- breadcrumb ----------

function renderBreadcrumb(parts) {
  const items = parts.map((p, i) => {
    const isLast = i === parts.length - 1;
    const label = i === 0 && p.icon ? `<span class="breadcrumb-home">${homeSvg()}</span>` : escapeHtml(p.label);
    const inner = isLast ? `<span aria-current="page">${label}</span>` : `<a href="${p.href || '#'}">${label}</a>`;
    const sep = isLast ? '' : `<li aria-hidden="true">${chevronSvg()}</li>`;
    return `<li>${inner}</li>${sep}`;
  });
  return `<nav class="breadcrumb" aria-label="Breadcrumb"><ol>${items.join('')}</ol></nav>`;
}

function chevronSvg() {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>`;
}

function homeSvg() {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M12 3.25 3 10.4v10.35h6.2v-6.2h5.6v6.2H21V10.4z"/></svg>`;
}

// ---------- param default values ----------

function defaultParamValue(p) {
  if (p.example !== undefined) return String(p.example);
  if (p.default !== undefined) return String(p.default);
  if (p.enum && p.enum.length) return String(p.enum[0]);
  if (p.type === 'integer' || p.type === 'number') return p.format === 'int64' ? '10' : '1';
  if (p.type === 'boolean') return 'true';
  if (p.required && p.in === 'path') return 'string';
  return '';
}

function getOpState(op) {
  if (tryState.has(op.id)) return tryState.get(op.id);

  const state = {
    baseUrl: doc.baseUrl,
    path: {},
    query: {},
    header: {},
    bodyText: '',
    mimeType: (op.consumes && op.consumes[0]) || 'application/json',
  };
  op.parameters.path.forEach(p => { state.path[p.name] = defaultParamValue(p); });
  op.parameters.query.forEach(p => { state.query[p.name] = p.required ? defaultParamValue(p) : ''; });
  op.parameters.header.forEach(p => { state.header[p.name] = p.required ? defaultParamValue(p) : ''; });
  if (op.produces?.length) state.header['Accept'] = op.produces.includes('application/json') ? 'application/json' : op.produces[0];
  if (op.parameters.body) {
    state.mimeType = op.parameters.body.contentType || state.mimeType;
    state.bodyText = typeof op.parameters.body.example === 'string'
      ? op.parameters.body.example
      : JSON.stringify(op.parameters.body.example, null, 2);
  }
  applyAuthToState(op, state);
  tryState.set(op.id, state);
  return state;
}

// Applies the sessionStorage-backed apikey/oauthkey values onto an
// operation's editable request state, for whichever security schemes that
// operation requires. Basic auth and apiKey-type schemes use 'apikey';
// oauth2-type schemes use 'oauthkey'.
function applyAuthToState(op, state) {
  const apikey = getStoredAuth('apikey');
  const oauthkey = getStoredAuth('oauthkey');
  for (const sec of op.security) {
    if (sec.type === 'apiKey') {
      if (sec.in === 'header') state.header[sec.name] = apikey;
      else if (sec.in === 'query') state.query[sec.name] = apikey;
    } else if (sec.type === 'basic') {
      state.header['Authorization'] = apikey ? `Basic ${apikey}` : '';
    } else if (sec.type === 'oauth2') {
      state.header['Authorization'] = oauthkey ? `Bearer ${oauthkey}` : '';
    }
  }
}

function updateAuthButtonVisual() {
  const authed = hasAnyStoredAuth();
  els.authBtn.classList.toggle('is-authorized', authed);
  els.authBtnLabel.textContent = authed ? 'Authorized' : 'Authorize';
}

function applyAuthToAllStates() {
  for (const [opId, state] of tryState.entries()) {
    const op = doc.findOperation(opId);
    if (op) applyAuthToState(op, state);
  }
  updateAuthButtonVisual();
  const op = doc.findOperation(currentOperationFromHash());
  if (op) renderTryIt(op);
}

function buildRequest(op, state) {
  let path = op.path;
  for (const [name, value] of Object.entries(state.path)) {
    path = path.replace(`{${name}}`, encodeURIComponent(value ?? ''));
  }
  const url = `${state.baseUrl}${path}`;
  const query = Object.entries(state.query).filter(([, v]) => v !== '').map(([k, v]) => [k, v]);
  const headers = Object.entries(state.header).filter(([, v]) => v !== '').map(([k, v]) => [k, v]);
  const hasBody = !!op.parameters.body && ['POST', 'PUT', 'PATCH'].includes(op.method);

  return {
    method: op.method,
    url,
    query,
    headers,
    bodyText: hasBody ? state.bodyText : '',
    mimeType: state.mimeType,
  };
}

// ---------- main doc pane ----------

function renderIntroduction() {
  const info = doc.info;
  els.main.innerHTML = `
    <div class="doc-pane">
      ${renderBreadcrumb([{ label: 'Home', href: '#/introduction', icon: true }, { label: 'Introduction' }])}
      <h1>${escapeHtml(info.title || 'API Documentation')}</h1>
      ${info.version ? `<span class="schema-pill">v${escapeHtml(info.version)}</span>` : ''}
      ${info.summary ? `<p class="param-desc">${escapeHtml(info.summary)}</p>` : ''}
      <div class="prose">${mdLite(info.description)}</div>
      <div class="intro-card">
        <header><h3>Base URL</h3></header>
        <section><code>${escapeHtml(doc.baseUrl)}</code></section>
      </div>
      <section class="doc-section">
        <h2>Groups</h2>
        <ul class="intro-group-list">
          ${doc.groups.map(g => `
            <li>
              <a href="#/${g.items[0]?.id}"><strong>${escapeHtml(g.name)}</strong></a>
              ${g.description ? `<span> &mdash; ${escapeHtml(g.description)}</span>` : ''}
              <span class="text-muted"> (${g.items.length} endpoint${g.items.length === 1 ? '' : 's'})</span>
            </li>
          `).join('')}
        </ul>
      </section>
    </div>
  `;
  els.tryit.innerHTML = '';
  els.tryit.hidden = true;
  renderMermaidDiagrams();
}

function renderParamGroup(title, params) {
  if (!params.length) return '';
  const detailsId = uid('params');
  return `
    <details class="param-group" open>
      <summary><span class="param-group-head"><span class="caret"></span>${title.toUpperCase()}</span></summary>
      <ul id="${detailsId}">
        ${params.map(p => `
          <li class="param-row">
            <div>
              <div class="param-row-head">
                <span class="param-name">${escapeHtml(p.name)}</span>
                <span class="param-type">${escapeHtml(p.type)}${p.format ? ` (${escapeHtml(p.format)})` : ''}</span>
              </div>
              ${p.description ? `<p class="param-desc">${escapeHtml(p.description)}</p>` : ''}
              ${p.enum ? `<p class="param-desc">enum: ${p.enum.map(e => `<code>${escapeHtml(e)}</code>`).join(', ')}</p>` : ''}
              ${p.example !== undefined ? `<p class="param-desc">example${p.exampleLabel ? ` (${escapeHtml(p.exampleLabel)})` : ''}: <code>${escapeHtml(p.example)}</code></p>` : ''}
            </div>
            ${p.required ? '<span class="required-chip">Required</span>' : ''}
          </li>
        `).join('')}
      </ul>
    </details>
  `;
}

function renderSchemaTree(node) {
  const rows = doc.schemaRows(node);
  if (!rows.length) return '<p class="text-muted">No schema available.</p>';
  return `
    <ul class="schema-tree">
      ${rows.map(r => `
        <li style="--depth:${(r.path.match(/\./g) || []).length}">
          <span class="param-name">${escapeHtml(r.path)}${r.required ? '<span class="required-star">*</span>' : ''}</span>
          <span class="param-type">${escapeHtml(r.type)}${r.format ? ` (${escapeHtml(r.format)})` : ''}</span>
          ${r.description ? `<p class="param-desc">${escapeHtml(r.description)}</p>` : ''}
        </li>
      `).join('')}
    </ul>
  `;
}

function renderResponses(op) {
  if (!op.responses.length) return '<p class="text-muted">No responses documented.</p>';

  const tabsId = uid('resp-tabs');
  const tabs = op.responses.map((r, i) => `
    <button type="button" class="tab-status" data-status="${statusClass(r.status)}" role="tab" id="${tabsId}-tab-${i}" aria-controls="${tabsId}-panel-${i}" aria-selected="${i === 0}" tabindex="${i === 0 ? 0 : -1}">
      ${escapeHtml(r.status)}
    </button>
  `).join('');

  const panels = op.responses.map((r, i) => {
    const schemaViewId = uid('view-schema');
    const exampleViewId = uid('view-example');
    const example = r.example;
    const exampleBody = typeof example === 'string' ? example : JSON.stringify(example, null, 2);
    return `
      <div role="tabpanel" id="${tabsId}-panel-${i}" aria-labelledby="${tabsId}-tab-${i}" tabindex="-1" ${i === 0 ? '' : 'hidden'}>
        ${r.summary ? `<p class="response-desc"><strong>${escapeHtml(r.summary)}</strong></p>` : ''}
        <p class="response-desc">${escapeHtml(r.description)}</p>
        ${r.headers.length ? renderParamGroup('Response Headers', r.headers) : ''}
        ${r.schema ? `
          <div class="response-body">
            <div class="response-body-head">
              <span class="badge" data-variant="outline">${escapeHtml(r.contentType || 'application/json')}</span>
              <div class="view-toggle" role="group">
                <button type="button" class="btn-sm" data-toggle-view data-show="${schemaViewId}" data-hide="${exampleViewId}" aria-pressed="true">Schema</button>
                <button type="button" class="btn-sm" data-toggle-view data-show="${exampleViewId}" data-hide="${schemaViewId}" aria-pressed="false">Example${r.exampleIsAuto ? ' (auto)' : ''}</button>
              </div>
            </div>
            <div id="${schemaViewId}">${renderSchemaTree(r.schema)}</div>
            <pre id="${exampleViewId}" hidden><code>${escapeHtml(exampleBody)}</code></pre>
          </div>
        ` : '<p class="text-muted">No response body.</p>'}
        ${r.links.length ? `
          <details class="param-group">
            <summary><span class="param-group-head"><span class="caret"></span>LINKS</span></summary>
            <ul>
              ${r.links.map(l => `
                <li class="param-row">
                  <div>
                    <div class="param-row-head"><span class="param-name">${escapeHtml(l.name)}</span></div>
                    ${l.description ? `<p class="param-desc">${escapeHtml(l.description)}</p>` : ''}
                    <p class="param-desc">${l.operationId ? `operationId: <code>${escapeHtml(l.operationId)}</code>` : `operationRef: <code>${escapeHtml(l.operationRef)}</code>`}</p>
                  </div>
                </li>
              `).join('')}
            </ul>
          </details>
        ` : ''}
      </div>
    `;
  }).join('');

  return `
    <div id="${tabsId}">
      <div class="response-tabs-head">
        <div class="response-tablist" role="tablist" aria-orientation="horizontal">${tabs}</div>
      </div>
      <div class="response-card-shell">${panels}</div>
    </div>
  `;
}

function renderOperation(op) {
  const group = doc.groups.find(g => g.name === op.tag);
  const groupLink = group ? `#/${group.items[0].id}` : '#';

  els.main.innerHTML = `
    <div class="doc-pane">
      ${renderBreadcrumb([{ label: 'Home', href: '#/introduction', icon: true }, { label: doc.info.title || 'API', href: '#/introduction' }, { label: op.tag, href: groupLink }, { label: op.summary }])}
      <h1>${escapeHtml(op.summary)}${op.deprecated ? ' <span class="schema-pill">Deprecated</span>' : ''}</h1>
      <div class="endpoint-box">
        ${methodBadge(op.method)}
        <code class="endpoint-url">${escapeHtml(doc.baseUrl + op.path)}</code>
      </div>
      ${op.description ? `<section class="doc-section"><div class="prose">${mdLite(op.description)}</div></section>` : ''}
      ${op.externalDocs?.url ? `<p class="param-desc"><a href="${escapeHtml(op.externalDocs.url)}" target="_blank" rel="noopener">${escapeHtml(op.externalDocs.description || 'External documentation')}</a></p>` : ''}

      <section class="doc-section">
        <h2>Request</h2>
        ${renderParamGroup('Path Parameters', op.parameters.path)}
        ${renderParamGroup('Query Parameters', op.parameters.query)}
        ${renderParamGroup('Header Parameters', op.parameters.header)}
        ${op.parameters.formData.length ? renderParamGroup('Form Data', op.parameters.formData) : ''}
        ${op.parameters.body ? `
          <details class="param-group" open>
            <summary><span class="param-group-head"><span class="caret"></span>BODY${op.parameters.body.required ? ' (required)' : ''}</span></summary>
            ${op.parameters.body.description ? `<p class="param-desc">${escapeHtml(op.parameters.body.description)}</p>` : ''}
            <p class="param-desc">content type: <code>${escapeHtml(op.parameters.body.contentType || 'application/json')}</code></p>
            ${op.parameters.body.encoding ? `
              <p class="param-desc">encoding:</p>
              <ul class="schema-tree">
                ${Object.entries(op.parameters.body.encoding).map(([prop, enc]) => `
                  <li>
                    <span class="param-name">${escapeHtml(prop)}</span>
                    <span class="param-type">${escapeHtml(enc.contentType || '')}${enc.style ? ` style=${escapeHtml(enc.style)}` : ''}</span>
                  </li>
                `).join('')}
              </ul>
            ` : ''}
            <p class="param-desc">example${op.parameters.body.exampleIsAuto ? ' (auto)' : op.parameters.body.exampleLabel ? ` (${escapeHtml(op.parameters.body.exampleLabel)})` : ''}:</p>
            <pre><code>${escapeHtml(typeof op.parameters.body.example === 'string' ? op.parameters.body.example : JSON.stringify(op.parameters.body.example, null, 2))}</code></pre>
            ${renderSchemaTree(op.parameters.body.schema)}
          </details>
        ` : ''}
      </section>

      <section class="doc-section">
        <div class="response-tabs-head">
          <h2>Responses</h2>
        </div>
        ${renderResponses(op)}
      </section>

      ${op.callbacks.length ? `
        <section class="doc-section">
          <h2>Callbacks</h2>
          <ul class="schema-tree">
            ${op.callbacks.map(cb => `
              <li>
                ${methodBadge(cb.method)}
                <span class="param-name">${escapeHtml(cb.name)}</span>
                <code class="param-type">${escapeHtml(cb.expression)}</code>
                ${cb.summary ? `<p class="param-desc">${escapeHtml(cb.summary)}</p>` : ''}
                ${cb.description ? `<p class="param-desc">${escapeHtml(cb.description)}</p>` : ''}
              </li>
            `).join('')}
          </ul>
        </section>
      ` : ''}
    </div>
  `;
  renderMermaidDiagrams();
}

function languageTone(langId) {
  if (langId === 'curl') return 'curl';
  if (langId === 'csharp') return 'csharp';
  if (langId === 'go') return 'go';
  if (langId === 'node') return 'node';
  if (langId === 'ruby') return 'ruby';
  if (langId === 'php') return 'php';
  return 'default';
}

function renderSnippetBlock(code) {
  const lines = String(code || '').split('\n');
  return `
    <pre class="snippet-block">
      <span class="snippet-gutter">${lines.map((_, index) => `<span>${index + 1}</span>`).join('')}</span>
      <span class="snippet-code"><code>${escapeHtml(code)}</code></span>
    </pre>
  `;
}

function renderResponsePlaceholder() {
  return `<div class="response-placeholder">Click the <code>Send API Request</code> button above and see the response here!</div>`;
}

// ---------- try-it panel ----------

function renderTryIt(op) {
  els.tryit.hidden = false;
  const state = getOpState(op);
  const serverOptions = (doc.servers || []).length ? doc.servers : [{ url: doc.baseUrl, description: '' }];
  const allParams = [
    ...op.parameters.path.map(p => ({ ...p, group: 'path' })),
    ...op.parameters.query.map(p => ({ ...p, group: 'query' })),
    ...op.parameters.header.map(p => ({ ...p, group: 'header' })),
  ];

  els.tryit.innerHTML = `
    <div class="tryit-stack">
      <section class="sdk-card">
        <div class="sdk-strip" role="tablist" aria-label="Code language">
          ${LANGUAGES.map(l => `
            <button type="button" class="lang-chip${l.id === currentLang ? ' active' : ''}" data-lang="${l.id}">
              <span class="lang-chip-icon lang-${languageTone(l.id)}">
                <span class="lang-chip-icon-image" style="background-image:url('${escapeHtml(assetUrl(l.icon))}')"></span>
              </span>
              <span class="lang-chip-label">${escapeHtml(l.label)}</span>
            </button>
          `).join('')}
        </div>
        <button type="button" class="sdk-current" data-lang="${currentLang}">${escapeHtml((LANGUAGES.find(l => l.id === currentLang) || LANGUAGES[0]).label)}</button>
        <div class="snippet-shell">${renderSnippetBlock('')}</div>
      </section>

      <section class="tryit-card">
        <header class="tryit-card-head">
          <h3>Request</h3>
          <button type="button" class="head-action" data-action="collapse-request">Collapse All</button>
        </header>
        <div class="tryit-card-body">
          <details open>
            <summary><span class="param-group-head"><span class="caret"></span>Base URL</span></summary>
            <div class="tryit-fields">
              <select class="select" data-param="baseUrl">
                ${serverOptions.map((server) => `
                  <option value="${escapeHtml(server.url)}" ${server.url === state.baseUrl ? 'selected' : ''}>
                    ${escapeHtml([server.name, server.description].filter(Boolean).join(' — ') || server.url)}${server.name || server.description ? ` (${escapeHtml(server.url)})` : ''}
                  </option>
                `).join('')}
              </select>
            </div>
          </details>

          ${allParams.length ? `
            <details open>
              <summary><span class="param-group-head"><span class="caret"></span>Parameters</span></summary>
              <div class="tryit-fields">
                ${allParams.map(p => `
                  <label class="tryit-field">
                    <span>${escapeHtml(p.name)} <span class="text-muted">— ${p.group}${p.required ? ' REQUIRED' : ''}</span></span>
                    ${p.enum ? `
                      <select class="select" data-param="${p.group}" data-name="${escapeHtml(p.name)}">
                        ${p.enum.map(v => `<option value="${escapeHtml(v)}" ${String(v) === state[p.group][p.name] ? 'selected' : ''}>${escapeHtml(v)}</option>`).join('')}
                      </select>
                    ` : `
                      <input type="text" class="input" data-param="${p.group}" data-name="${escapeHtml(p.name)}" value="${escapeHtml(state[p.group][p.name] ?? '')}" placeholder="${p.description ? escapeHtml(p.description) : p.required ? 'required' : 'optional'}">
                    `}
                  </label>
                `).join('')}
              </div>
            </details>
          ` : ''}

          ${op.parameters.body ? `
            <details open>
              <summary><span class="param-group-head"><span class="caret"></span>Body (${escapeHtml(state.mimeType)})</span></summary>
              <div class="tryit-fields">
                <textarea class="textarea" rows="8" data-param="body">${escapeHtml(state.bodyText)}</textarea>
              </div>
            </details>
          ` : ''}

          <button type="button" class="btn" data-action="send">Send API Request</button>
        </div>
      </section>

      <section class="tryit-card">
        <header class="tryit-card-head">
          <h3>Response</h3>
          <button type="button" class="response-clear" data-action="clear-response">Clear</button>
        </header>
        <div class="tryit-card-body" id="tryit-result">${renderResponsePlaceholder()}</div>
      </section>
    </div>
  `;

  updateSnippet(op);
}

async function updateSnippet(op) {
  const state = getOpState(op);
  const req = buildRequest(op, state);
  const lang = LANGUAGES.find(l => l.id === currentLang);
  const shell = els.tryit.querySelector('.snippet-shell');
  els.tryit.querySelector('.sdk-current').textContent = lang.label;
  try {
    shell.innerHTML = renderSnippetBlock(await generateSnippet(currentLang, req));
  } catch (err) {
    shell.innerHTML = renderSnippetBlock(`// Could not generate ${lang.label} snippet: ${err.message}`);
  }
}

async function sendRequest(op) {
  const state = getOpState(op);
  const req = buildRequest(op, state);
  const resultEl = document.getElementById('tryit-result');
  resultEl.innerHTML = `<div class="tryit-sending">Sending&hellip;</div>`;

  const fetchUrl = req.query.length ? `${req.url}?${new URLSearchParams(req.query).toString()}` : req.url;
  const fetchHeaders = Object.fromEntries(req.headers);
  if (req.bodyText) fetchHeaders['Content-Type'] = fetchHeaders['Content-Type'] || req.mimeType;

  const started = performance.now();
  try {
    const res = await fetch(fetchUrl, {
      method: req.method,
      headers: fetchHeaders,
      body: req.bodyText ? req.bodyText : undefined,
    });
    const elapsed = Math.round(performance.now() - started);
    const text = await res.text();
    let bodyText;
    try {
      bodyText = JSON.stringify(JSON.parse(text), null, 2);
    } catch {
      bodyText = text;
    }
    const headerLines = [...res.headers.entries()].map(([k, v]) => `${k}: ${v}`).join('\n');

    resultEl.innerHTML = `
      <div class="tryit-result-head">
        <span class="badge status-badge" data-status="${statusClass(res.status)}">${res.status}${res.statusText ? ` ${escapeHtml(res.statusText)}` : ''}</span>
        <span class="text-muted">${elapsed} ms</span>
      </div>
      <details class="param-group">
        <summary><span class="param-group-head"><span class="caret"></span>Response headers</span></summary>
        ${renderSnippetBlock(headerLines || '(none)')}
      </details>
      ${renderSnippetBlock(bodyText)}
    `;
  } catch (err) {
    resultEl.innerHTML = `
      <div class="tryit-result-head">
        <span class="schema-pill">Request failed</span>
      </div>
      <p class="param-desc">${escapeHtml(err.message)}. This is usually a CORS restriction or network error &mdash; check the browser console for details.</p>
    `;
  }
}

// ---------- authorize dialog ----------

/**
 * Renders one sessionStorage-backed auth input with its optional clean action.
 * Saved values stay visible but locked until the corresponding storage key is cleared.
 */
function renderStoredAuthField({ id, label, storageKey, placeholder, helpText }) {
  const value = getStoredAuth(storageKey);
  const isLocked = isStoredAuthLocked(storageKey);

  return `
    <div class="auth-field">
      <label class="tryit-field auth-field">
        <span class="auth-field-label">${label}</span>
        <div class="auth-input-row">
          <input
            type="text"
            class="input"
            id="${id}"
            value="${escapeHtml(value)}"
            placeholder="${escapeHtml(placeholder)}"
            ${isLocked ? 'disabled' : ''}
          >
          <button
            type="button"
            class="btn-outline btn-sm auth-clean-button ${isLocked ? '' : 'is-hidden'}"
            data-auth-clear="${storageKey}"
          >
            Clean
          </button>
        </div>
      </label>
      <p class="param-desc auth-field-help">${helpText}</p>
    </div>
  `;
}

function renderAuthGlobalFields() {
  return `
    <div class="auth-global-fields">
      ${renderStoredAuthField({
        id: 'auth-global-apikey',
        label: 'API Key',
        storageKey: 'apikey',
        placeholder: 'Enter API key',
        helpText: `Stored in this tab's session storage as <code>apikey</code>. Used for apiKey-type schemes and as the Basic auth credential (<code>Authorization: Basic &lt;API Key&gt;</code>).`,
      })}
      ${renderStoredAuthField({
        id: 'auth-global-oauthkey',
        label: 'OAuth Key (access token)',
        storageKey: 'oauthkey',
        placeholder: 'Bearer token (auto-filled by Authorize/Get token below)',
        helpText: `Stored in this tab's session storage as <code>oauthkey</code>. Used for oauth2-type schemes as <code>Authorization: Bearer &lt;OAuth Key&gt;</code>.`,
      })}
    </div>
  `;
}

function renderAuthDialogBody() {
  const defs = doc.securityDefinitions;
  const keys = Object.keys(defs);
  const globalFields = renderAuthGlobalFields();
  if (!keys.length) return globalFields + '<p class="text-muted auth-empty-state">No authorization schemes defined by this spec.</p>';

  const schemeCards = keys.map(key => {
    const def = defs[key];
    const current = authValues.get(key) || {};
    let fields = '';

    if (def.type === 'apiKey') {
      fields = `<p class="param-desc">Sent as ${def.in === 'query' ? 'query parameter' : 'header'} <code>${escapeHtml(def.name)}</code> using the <strong>API Key</strong> above.</p>`;
    } else if (def.type === 'basic') {
      fields = `<p class="param-desc">Sent as <code>Authorization: Basic &lt;API Key&gt;</code> using the <strong>API Key</strong> above.</p>`;
    } else if (def.type === 'mutualTLS') {
      fields = `<p class="param-desc">This scheme authenticates via a client TLS certificate negotiated at the transport layer &mdash; there is nothing to fill in here.</p>`;
    } else if (def.type === 'oauth2') {
      const scopes = Object.entries(def.scopes || {});
      const selectedScopes = current.scopes || Object.keys(def.scopes || {});
      const flow = def.flow;
      const usesPopup = flow === 'implicit' || flow === 'accessCode';
      const usesClientSecret = flow === 'accessCode' || flow === 'application' || flow === 'password';

      fields = `
        <p class="param-desc">Flow: <code>${escapeHtml(OAUTH_FLOW_LABELS[flow] || flow || 'unknown')}</code></p>
        ${scopes.length ? `
          <div class="tryit-field auth-field">
            <span class="auth-field-label">Scopes</span>
            <ul class="auth-scope-list">
              ${scopes.map(([s, d]) => `
                <li>
                  <label>
                    <input type="checkbox" data-oauth-scope-group="${escapeHtml(key)}" value="${escapeHtml(s)}" ${selectedScopes.includes(s) ? 'checked' : ''}>
                    <code>${escapeHtml(s)}</code> &mdash; ${escapeHtml(d)}
                  </label>
                </li>
              `).join('')}
            </ul>
          </div>
        ` : ''}
        <label class="tryit-field auth-field">
          <span class="auth-field-label">Client ID</span>
          <input type="text" class="input" data-auth-key="${escapeHtml(key)}" data-auth-field="clientId" value="${escapeHtml(current.clientId || '')}">
        </label>
        ${usesClientSecret ? `
          <label class="tryit-field auth-field">
            <span class="auth-field-label">Client secret</span>
            <input type="password" class="input" data-auth-key="${escapeHtml(key)}" data-auth-field="clientSecret" value="${escapeHtml(current.clientSecret || '')}">
          </label>
          <p class="param-desc auth-field-help">Client secret is sent from the browser &mdash; only use a test/dev OAuth client here.</p>
        ` : ''}
        ${flow === 'password' ? `
          <label class="tryit-field auth-field">
            <span class="auth-field-label">Username</span>
            <input type="text" class="input" data-auth-key="${escapeHtml(key)}" data-auth-field="username" value="${escapeHtml(current.username || '')}">
          </label>
          <label class="tryit-field auth-field">
            <span class="auth-field-label">Password</span>
            <input type="password" class="input" data-auth-key="${escapeHtml(key)}" data-auth-field="password" value="${escapeHtml(current.password || '')}">
          </label>
        ` : ''}
        ${flow ? `
          <div class="tryit-field auth-field auth-action-row">
            <button type="button" class="btn-outline btn-sm" data-oauth-authorize="${escapeHtml(key)}">${usesPopup ? 'Authorize' : 'Get token'}</button>
            <p class="param-desc auth-field-help" data-oauth-status="${escapeHtml(key)}"></p>
          </div>
        ` : ''}
        <p class="param-desc auth-field-help">Fills the <strong>OAuth Key</strong> field above on success${getStoredAuth('oauthkey') ? ` &mdash; currently set (ends in …${escapeHtml(getStoredAuth('oauthkey').slice(-4))})` : ' — not set yet'}.</p>
      `;
    } else {
      fields = `<p class="param-desc">Unsupported scheme type: ${escapeHtml(def.type)}</p>`;
    }

    return `
      <details class="param-group auth-scheme" open>
        <summary>
          <span>${escapeHtml(key)}</span>
          <span class="badge" data-variant="outline">${escapeHtml(def.type)}</span>
          ${def.deprecated ? '<span class="schema-pill">Deprecated</span>' : ''}
        </summary>
        ${def.description ? `<p class="param-desc">${escapeHtml(def.description)}</p>` : ''}
        ${def.oauth2MetadataUrl ? `<p class="param-desc">Metadata: <a href="${escapeHtml(def.oauth2MetadataUrl)}" target="_blank" rel="noopener">${escapeHtml(def.oauth2MetadataUrl)}</a></p>` : ''}
        ${fields}
      </details>
    `;
  }).join('');

  return globalFields + `<div class="auth-schemes">${schemeCards}</div>`;
}

function openAuthDialog() {
  els.authDialogBody.innerHTML = renderAuthDialogBody();
  els.authDialog.showModal();
}

/**
 * Clears one saved auth credential, rerenders the dialog field state, and
 * reapplies auth headers/query params across every Try It request draft.
 */
function clearStoredAuthField(storageKey) {
  setStoredAuth(storageKey, '');
  if (els.authDialog.open) {
    els.authDialogBody.innerHTML = renderAuthDialogBody();
  }
  applyAuthToAllStates();
}

/**
 * Persists the dialog's global auth values to session storage and refreshes
 * in-memory OAuth helper values before updating all Try It request drafts.
 */
function saveAuthDialog() {
  setStoredAuth('apikey', document.getElementById('auth-global-apikey').value.trim());
  setStoredAuth('oauthkey', document.getElementById('auth-global-oauthkey').value.trim());
  els.authDialogBody.querySelectorAll('[data-auth-key][data-auth-field]').forEach(input => {
    const key = input.dataset.authKey;
    const field = input.dataset.authField;
    const entry = authValues.get(key) || {};
    entry[field] = input.value;
    authValues.set(key, entry);
  });
  els.authDialog.close();
  applyAuthToAllStates();
}

function authFieldValue(key, field) {
  for (const el of els.authDialogBody.querySelectorAll('[data-auth-key][data-auth-field]')) {
    if (el.dataset.authKey === key && el.dataset.authField === field) return el.value.trim();
  }
  return '';
}

function authSelectedScopes(key) {
  const scopes = [];
  for (const el of els.authDialogBody.querySelectorAll('[data-oauth-scope-group]')) {
    if (el.dataset.oauthScopeGroup === key && el.checked) scopes.push(el.value);
  }
  return scopes;
}

function setOAuthStatus(key, message, isError) {
  for (const el of els.authDialogBody.querySelectorAll('[data-oauth-status]')) {
    if (el.dataset.oauthStatus === key) {
      el.textContent = message;
      el.style.color = isError ? '#dc2626' : '#15803d';
      return;
    }
  }
}

function applyOAuthToken(key, token, clientId, clientSecret) {
  setStoredAuth('oauthkey', token);
  const entry = authValues.get(key) || {};
  if (clientId) entry.clientId = clientId;
  if (clientSecret) entry.clientSecret = clientSecret;
  authValues.set(key, entry);
  applyAuthToAllStates();
  if (els.authDialog.open) els.authDialogBody.innerHTML = renderAuthDialogBody();
}

async function requestOAuthToken(key, tokenUrl, params) {
  if (!tokenUrl) {
    setOAuthStatus(key, 'No tokenUrl defined for this scheme.', true);
    return;
  }
  const body = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v) body.set(k, v);
  }
  try {
    const res = await fetch(resolveOAuthEndpointUrl(tokenUrl, currentSelectedBaseUrl()), {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded', Accept: 'application/json' },
      body: body.toString(),
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok || !data.access_token) {
      throw new Error(data.error_description || data.error || `HTTP ${res.status}`);
    }
    applyOAuthToken(key, data.access_token, params.client_id, params.client_secret);
    setOAuthStatus(key, 'Token acquired.', false);
  } catch (err) {
    setOAuthStatus(key, `Token request failed: ${err.message}. This is usually a CORS restriction on the auth server.`, true);
  }
}

function startOAuth2Flow(key) {
  const def = doc.securityDefinitions[key];
  const flow = def.flow;
  const clientId = authFieldValue(key, 'clientId');
  const clientSecret = authFieldValue(key, 'clientSecret');
  const scopes = authSelectedScopes(key);

  if (flow === 'implicit' || flow === 'accessCode') {
    if (!def.authorizationUrl) {
      setOAuthStatus(key, 'No authorizationUrl defined for this scheme.', true);
      return;
    }
    const state = randomState();
    const authUrl = new URL(resolveOAuthEndpointUrl(def.authorizationUrl, currentSelectedBaseUrl()));
    authUrl.searchParams.set('response_type', flow === 'implicit' ? 'token' : 'code');
    if (clientId) authUrl.searchParams.set('client_id', clientId);
    authUrl.searchParams.set('redirect_uri', OAUTH_REDIRECT_URI);
    if (scopes.length) authUrl.searchParams.set('scope', scopes.join(' '));
    authUrl.searchParams.set('state', state);

    const popup = window.open(authUrl.toString(), `oauth2-authorize-${key}`, 'width=560,height=680');
    if (!popup) {
      setOAuthStatus(key, 'Popup blocked — allow popups for this site and try again.', true);
      return;
    }
    pendingOAuth.set(state, { key, flow, clientId, clientSecret, popup });
    setOAuthStatus(key, 'Waiting for authorization in the popup…', false);

    const watcher = setInterval(() => {
      if (!popup.closed) return;
      clearInterval(watcher);
      if (pendingOAuth.delete(state)) setOAuthStatus(key, 'Authorization window was closed before completing.', true);
    }, 500);
    return;
  }

  if (flow === 'password') {
    setOAuthStatus(key, 'Requesting token…', false);
    requestOAuthToken(key, def.tokenUrl, {
      grant_type: 'password',
      username: authFieldValue(key, 'username'),
      password: authFieldValue(key, 'password'),
      client_id: clientId,
      client_secret: clientSecret,
      scope: scopes.join(' '),
    });
    return;
  }

  if (flow === 'application') {
    setOAuthStatus(key, 'Requesting token…', false);
    requestOAuthToken(key, def.tokenUrl, {
      grant_type: 'client_credentials',
      client_id: clientId,
      client_secret: clientSecret,
      scope: scopes.join(' '),
    });
    return;
  }

  setOAuthStatus(key, `Unsupported OAuth2 flow: ${flow}`, true);
}

function handleOAuth2Message(event) {
  if (event.origin !== window.location.origin) return;
  const data = event.data;
  if (!data || data.source !== 'oauth2-redirect') return;
  const pending = pendingOAuth.get(data.state);
  if (!pending) return;
  pendingOAuth.delete(data.state);

  const { key, flow, clientId, clientSecret } = pending;

  if (data.error) {
    setOAuthStatus(key, `Authorization failed: ${data.error}${data.errorDescription ? ' - ' + data.errorDescription : ''}`, true);
    return;
  }

  if (flow === 'implicit') {
    if (!data.accessToken) {
      setOAuthStatus(key, 'Authorization server did not return an access token.', true);
      return;
    }
    applyOAuthToken(key, data.accessToken, clientId, clientSecret);
    setOAuthStatus(key, 'Token acquired.', false);
    return;
  }

  if (flow === 'accessCode') {
    if (!data.code) {
      setOAuthStatus(key, 'Authorization server did not return a code.', true);
      return;
    }
    setOAuthStatus(key, 'Exchanging code for token…', false);
    requestOAuthToken(key, doc.securityDefinitions[key].tokenUrl, {
      grant_type: 'authorization_code',
      code: data.code,
      redirect_uri: OAUTH_REDIRECT_URI,
      client_id: clientId,
      client_secret: clientSecret,
    });
  }
}

// ---------- routing ----------

function currentOperationFromHash() {
  const id = (location.hash || '#/introduction').replace(/^#\//, '');
  if (id === 'introduction') return 'introduction';
  return doc.findOperation(id) ? id : 'introduction';
}

function route() {
  const id = currentOperationFromHash();
  setActiveSidebarItem(id);
  if (id === 'introduction') {
    renderIntroduction();
    return;
  }
  const op = doc.findOperation(id);
  renderOperation(op);
  renderTryIt(op);
}

// ---------- event wiring ----------

function wireEvents() {
  window.addEventListener('hashchange', route);

  els.sidebarToggle.addEventListener('click', () => {
    toggleSidebar();
  });

  els.themeButton?.addEventListener('click', toggleTheme);

  els.authBtn.addEventListener('click', openAuthDialog);

  els.authDialog.addEventListener('click', (e) => {
    if (e.target.closest('[data-action="auth-cancel"]')) els.authDialog.close();
    else if (e.target.closest('[data-action="auth-save"]')) saveAuthDialog();
    else {
      const clearBtn = e.target.closest('[data-auth-clear]');
      if (clearBtn) {
        clearStoredAuthField(clearBtn.dataset.authClear);
        return;
      }
      const oauthBtn = e.target.closest('[data-oauth-authorize]');
      if (oauthBtn) startOAuth2Flow(oauthBtn.dataset.oauthAuthorize);
    }
  });

  window.addEventListener('message', handleOAuth2Message);

  els.sidebarGroups.addEventListener('click', (e) => {
    const navLink = e.target.closest('a[data-op-id]');
    if (!navLink) return;
    closeSidebarAfterSelection();
  });

  document.addEventListener('click', (e) => {
    if (!isSidebarOverlayViewport() || !isSidebarOpen()) return;
    if (e.target.closest('#sidebar, #sidebar-toggle')) return;
    closeSidebar();
  });

  els.main.addEventListener('click', (e) => {
    const tab = e.target.closest('[role="tab"]');
    if (tab) {
      const root = tab.closest('[id]');
      root.querySelectorAll('[role="tab"]').forEach((item) => {
        item.setAttribute('aria-selected', String(item === tab));
        item.tabIndex = item === tab ? 0 : -1;
      });
      root.querySelectorAll('[role="tabpanel"]').forEach((panel) => {
        panel.hidden = panel.id !== tab.getAttribute('aria-controls');
      });
      return;
    }

    const btn = e.target.closest('[data-toggle-view]');
    if (!btn) return;
    const group = btn.closest('.view-toggle');
    group.querySelectorAll('[data-toggle-view]').forEach(b => b.setAttribute('aria-pressed', 'false'));
    btn.setAttribute('aria-pressed', 'true');
    document.getElementById(btn.dataset.show).hidden = false;
    document.getElementById(btn.dataset.hide).hidden = true;
  });

  els.tryit.addEventListener('click', (e) => {
    const op = doc.findOperation(currentOperationFromHash());
    if (!op) return;

    const langBtn = e.target.closest('[data-lang]');
    if (langBtn) {
      currentLang = langBtn.dataset.lang;
      els.tryit.querySelectorAll('.lang-chip').forEach(b => b.classList.toggle('active', b.dataset.lang === currentLang));
      updateSnippet(op);
      return;
    }

    if (e.target.closest('[data-action="copy-snippet"]')) {
      copyText(els.tryit.querySelector('.snippet-code code').textContent, e.target.closest('button'));
      return;
    }

    if (e.target.closest('[data-action="send"]')) {
      sendRequest(op);
      return;
    }

    if (e.target.closest('[data-action="clear-response"]')) {
      document.getElementById('tryit-result').innerHTML = renderResponsePlaceholder();
      return;
    }

    if (e.target.closest('[data-action="collapse-request"]')) {
      els.tryit.querySelectorAll('.tryit-card details').forEach((details, index) => {
        if (index < 3) details.open = false;
      });
    }
  });

  els.tryit.addEventListener('input', (e) => {
    const target = e.target.closest('[data-param]');
    if (!target) return;
    const op = doc.findOperation(currentOperationFromHash());
    if (!op) return;
    const state = getOpState(op);
    const paramGroup = target.dataset.param;

    if (paramGroup === 'baseUrl') state.baseUrl = target.value;
    else if (paramGroup === 'body') state.bodyText = target.value;
    else state[paramGroup][target.dataset.name] = target.value;

    updateSnippet(op);
  });
}

// ---------- boot ----------

async function main() {
  applyTheme(getPreferredTheme());
  initMermaid();
  syncSidebarForViewport();
  window.addEventListener('resize', syncSidebarForViewport);
  wireEvents();
  try {
    doc = await loadOpenApiDoc(SPEC_URL);
  } catch (err) {
    els.main.innerHTML = `<div class="empty-state">Failed to load API spec from ${escapeHtml(SPEC_URL)}: ${escapeHtml(err.message)}</div>`;
    return;
  }
  els.specVersion.textContent = doc.info.version ? `v${doc.info.version}` : '';
  els.authBtn.hidden = Object.keys(doc.securityDefinitions).length === 0;
  updateAuthButtonVisual();
  renderSidebar();
  route();
}

main();
