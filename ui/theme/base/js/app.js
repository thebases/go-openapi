// loadOpenApiDoc (openapi.js) and LANGUAGES/generateSnippet (snippet.js) are
// loaded as globals via plain <script> tags before this file, in that order.

const SPEC_URL = 'petstore.json';

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
};

/** doc: normalized OpenAPI model. tryState: Map<opId, editable request state>. */
let doc = null;
const tryState = new Map();
let currentLang = 'curl';

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

function getStoredAuth(name) {
  return sessionStorage.getItem(name) || '';
}

function setStoredAuth(name, value) {
  if (value) sessionStorage.setItem(name, value);
  else sessionStorage.removeItem(name);
}

function hasAnyStoredAuth() {
  return !!(getStoredAuth('apikey') || getStoredAuth('oauthkey'));
}

// ---------- tiny helpers ----------

function escapeHtml(str) {
  return String(str ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

// Minimal markdown: **bold**, `code`, [text](url), blank-line paragraphs.
function mdLite(src) {
  if (!src) return '';
  const paragraphs = src.split(/\n\s*\n/).map(p => {
    let html = escapeHtml(p.trim());
    html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
    html = html.replace(/`([^`]+)`/g, '<code>$1</code>');
    html = html.replace(/\[([^\]]+)\]\((https?:\/\/[^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>');
    return `<p>${html.replace(/\n/g, '<br>')}</p>`;
  });
  return paragraphs.join('');
}

function methodBadge(method) {
  return `<span class="badge method-badge" data-method="${method}">${method}</span>`;
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
  const isHidden = els.sidebar.getAttribute('aria-hidden') === 'true';
  els.sidebar.setAttribute('aria-hidden', isHidden ? 'false' : 'true');
  els.sidebarToggle.setAttribute('aria-expanded', isHidden ? 'true' : 'false');
}

function syncSidebarForViewport() {
  const isDesktop = window.matchMedia('(min-width: 1101px)').matches;
  els.sidebar.setAttribute('aria-hidden', isDesktop ? 'false' : 'true');
  els.sidebarToggle.setAttribute('aria-expanded', isDesktop ? 'true' : 'false');
}

// ---------- sidebar ----------

function renderSidebar() {
  const items = [];
  items.push(`<ul>`);
  items.push(`<li><a href="#/introduction" data-op-id="introduction"><span>Introduction</span></a></li>`);

  for (const group of doc.groups) {
    const detailsId = uid('grp');
    items.push(`
      <li>
        <details id="${detailsId}" open>
          <summary aria-controls="${detailsId}-content">
            <span>${escapeHtml(group.name)}</span>
          </summary>
          <ul id="${detailsId}-content">
            ${group.items.map(op => `
              <li>
                <a href="#/${op.id}" data-op-id="${op.id}">
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
    const inner = isLast ? `<span aria-current="page">${escapeHtml(p.label)}</span>` : `<a href="${p.href || '#'}">${escapeHtml(p.label)}</a>`;
    const sep = isLast ? '' : `<li aria-hidden="true">${chevronSvg()}</li>`;
    return `<li>${inner}</li>${sep}`;
  });
  return `<nav class="breadcrumb" aria-label="Breadcrumb"><ol>${items.join('')}</ol></nav>`;
}

function chevronSvg() {
  return `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>`;
}

// ---------- param default values ----------

function defaultParamValue(p) {
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
    state.bodyText = JSON.stringify(doc.generateExample(op.parameters.body.schema), null, 2);
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
    ${renderBreadcrumb([{ label: 'Home', href: '#/introduction' }, { label: 'Introduction' }])}
    <h1>${escapeHtml(info.title || 'API Documentation')}</h1>
    ${info.version ? `<span class="badge" data-variant="secondary">v${escapeHtml(info.version)}</span>` : ''}
    <div class="prose">${mdLite(info.description)}</div>
    <div class="card intro-card">
      <header><h3>Base URL</h3></header>
      <section><code>${escapeHtml(doc.baseUrl)}</code></section>
    </div>
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
  `;
  els.tryit.innerHTML = '';
  els.tryit.hidden = true;
}

function renderParamGroup(title, params) {
  if (!params.length) return '';
  const detailsId = uid('params');
  return `
    <details class="param-group" open>
      <summary>${title.toUpperCase()}</summary>
      <ul id="${detailsId}">
        ${params.map(p => `
          <li class="param-row">
            <div class="param-row-head">
              <span class="param-name">${escapeHtml(p.name)}${p.required ? '<span class="required-star">*</span>' : ''}</span>
              <span class="param-type">${escapeHtml(p.type)}${p.format ? ` (${escapeHtml(p.format)})` : ''}</span>
            </div>
            ${p.description ? `<p class="param-desc">${escapeHtml(p.description)}</p>` : ''}
            ${p.enum ? `<p class="param-desc">enum: ${p.enum.map(e => `<code>${escapeHtml(e)}</code>`).join(', ')}</p>` : ''}
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
    <button type="button" role="tab" id="${tabsId}-tab-${i}" aria-controls="${tabsId}-panel-${i}" aria-selected="${i === 0}" tabindex="${i === 0 ? 0 : -1}">
      <span class="badge status-badge" data-status="${statusClass(r.status)}">${escapeHtml(r.status)}</span>
    </button>
  `).join('');

  const panels = op.responses.map((r, i) => {
    const schemaViewId = uid('view-schema');
    const exampleViewId = uid('view-example');
    const example = r.schema ? doc.generateExample(r.schema) : null;
    return `
      <div role="tabpanel" id="${tabsId}-panel-${i}" aria-labelledby="${tabsId}-tab-${i}" tabindex="-1" ${i === 0 ? '' : 'hidden'}>
        <p class="response-desc">${escapeHtml(r.description)}</p>
        ${r.schema ? `
          <div class="response-body">
            <div class="response-body-head">
              <span class="badge" data-variant="outline">application/json</span>
              <div class="view-toggle" role="group">
                <button type="button" class="btn-sm" data-toggle-view data-show="${schemaViewId}" data-hide="${exampleViewId}" aria-pressed="true">Schema</button>
                <button type="button" class="btn-sm" data-toggle-view data-show="${exampleViewId}" data-hide="${schemaViewId}" aria-pressed="false">Example (auto)</button>
              </div>
            </div>
            <div id="${schemaViewId}">${renderSchemaTree(r.schema)}</div>
            <pre id="${exampleViewId}" hidden><code>${escapeHtml(JSON.stringify(example, null, 2))}</code></pre>
          </div>
        ` : '<p class="text-muted">No response body.</p>'}
      </div>
    `;
  }).join('');

  return `
    <div class="tabs" id="${tabsId}">
      <nav role="tablist" aria-orientation="horizontal">${tabs}</nav>
      ${panels}
    </div>
  `;
}

function renderOperation(op) {
  const group = doc.groups.find(g => g.name === op.tag);
  const groupLink = group ? `#/${group.items[0].id}` : '#';

  els.main.innerHTML = `
    ${renderBreadcrumb([{ label: 'Home', href: '#/introduction' }, { label: doc.info.title || 'API', href: '#/introduction' }, { label: op.tag, href: groupLink }, { label: op.summary }])}
    <h1>${escapeHtml(op.summary)}${op.deprecated ? ' <span class="badge" data-variant="destructive">Deprecated</span>' : ''}</h1>
    <div class="endpoint-url-row">
      ${methodBadge(op.method)}
      <code class="endpoint-url">${escapeHtml(doc.baseUrl + op.path)}</code>
    </div>
    ${op.description ? `<div class="prose">${mdLite(op.description)}</div>` : ''}

    <h2>Request</h2>
    ${renderParamGroup('Path Parameters', op.parameters.path)}
    ${renderParamGroup('Query Parameters', op.parameters.query)}
    ${renderParamGroup('Header Parameters', op.parameters.header)}
    ${op.parameters.formData.length ? renderParamGroup('Form Data', op.parameters.formData) : ''}
    ${op.parameters.body ? `
      <details class="param-group" open>
        <summary>BODY${op.parameters.body.required ? ' (required)' : ''}</summary>
        ${op.parameters.body.description ? `<p class="param-desc">${escapeHtml(op.parameters.body.description)}</p>` : ''}
        ${renderSchemaTree(op.parameters.body.schema)}
      </details>
    ` : ''}

    <h2>Responses</h2>
    ${renderResponses(op)}
  `;
}

// ---------- try-it panel ----------

function renderTryIt(op) {
  els.tryit.hidden = false;
  const state = getOpState(op);
  const allParams = [
    ...op.parameters.path.map(p => ({ ...p, group: 'path' })),
    ...op.parameters.query.map(p => ({ ...p, group: 'query' })),
    ...op.parameters.header.map(p => ({ ...p, group: 'header' })),
  ];

  els.tryit.innerHTML = `
    <div class="tryit-lang-strip" role="tablist" aria-label="Code language">
      ${LANGUAGES.map(l => `<button type="button" class="lang-btn${l.id === currentLang ? ' active' : ''}" data-lang="${l.id}">${escapeHtml(l.label)}</button>`).join('')}
    </div>

    <div class="tryit-snippet-card">
      <div class="tryit-snippet-head">
        <span class="text-muted" id="tryit-lang-label"></span>
        <button type="button" class="btn-sm" data-action="copy-snippet">Copy</button>
      </div>
      <pre class="snippet-block"><code id="tryit-snippet"></code></pre>
    </div>

    <div class="card tryit-request-card">
      <header class="tryit-request-head">
        <h3>Request</h3>
      </header>
      <section>
        <details open>
          <summary>Base URL</summary>
          <input type="text" class="input" data-param="baseUrl" value="${escapeHtml(state.baseUrl)}">
        </details>

        ${allParams.length ? `
          <details open>
            <summary>Parameters</summary>
            ${allParams.map(p => `
              <label class="tryit-field">
                <span>${escapeHtml(p.name)}${p.required ? '<span class="required-star">*</span>' : ''} <span class="text-muted">&mdash; ${p.group}</span></span>
                ${p.enum ? `
                  <select class="select" data-param="${p.group}" data-name="${escapeHtml(p.name)}">
                    ${p.enum.map(v => `<option value="${escapeHtml(v)}" ${String(v) === state[p.group][p.name] ? 'selected' : ''}>${escapeHtml(v)}</option>`).join('')}
                  </select>
                ` : `
                  <input type="text" class="input" data-param="${p.group}" data-name="${escapeHtml(p.name)}" value="${escapeHtml(state[p.group][p.name] ?? '')}" placeholder="${p.required ? 'required' : 'optional'}">
                `}
              </label>
            `).join('')}
          </details>
        ` : ''}

        ${op.parameters.body ? `
          <details open>
            <summary>Body (${escapeHtml(state.mimeType)})</summary>
            <textarea class="textarea" rows="8" data-param="body">${escapeHtml(state.bodyText)}</textarea>
          </details>
        ` : ''}
      </section>
      <footer>
        <button type="button" class="btn" data-action="send">Send API Request</button>
      </footer>
    </div>

    <div id="tryit-result"></div>
  `;

  updateSnippet(op);
}

async function updateSnippet(op) {
  const state = getOpState(op);
  const req = buildRequest(op, state);
  const lang = LANGUAGES.find(l => l.id === currentLang);
  document.getElementById('tryit-lang-label').textContent = lang.label;
  const codeEl = document.getElementById('tryit-snippet');
  try {
    codeEl.textContent = await generateSnippet(currentLang, req);
  } catch (err) {
    codeEl.textContent = `// Could not generate ${lang.label} snippet: ${err.message}`;
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
    let bodyHtml;
    try {
      bodyHtml = escapeHtml(JSON.stringify(JSON.parse(text), null, 2));
    } catch {
      bodyHtml = escapeHtml(text);
    }
    const headerLines = [...res.headers.entries()].map(([k, v]) => `${k}: ${v}`).join('\n');

    resultEl.innerHTML = `
      <div class="tryit-result-head">
        <span class="badge status-badge" data-status="${statusClass(res.status)}">${res.status}${res.statusText ? ` ${escapeHtml(res.statusText)}` : ''}</span>
        <span class="text-muted">${elapsed} ms</span>
      </div>
      <details class="param-group">
        <summary>Response headers</summary>
        <pre class="snippet-block"><code>${escapeHtml(headerLines) || '(none)'}</code></pre>
      </details>
      <pre class="snippet-block"><code>${bodyHtml}</code></pre>
    `;
  } catch (err) {
    resultEl.innerHTML = `
      <div class="tryit-result-head">
        <span class="badge" data-variant="destructive">Request failed</span>
      </div>
      <p class="param-desc">${escapeHtml(err.message)}. This is usually a CORS restriction or network error &mdash; check the browser console for details.</p>
    `;
  }
}

// ---------- authorize dialog ----------

function renderAuthGlobalFields() {
  return `
    <div class="auth-global-fields">
      <label class="tryit-field">
        <span>API Key</span>
        <input type="text" class="input" id="auth-global-apikey" value="${escapeHtml(getStoredAuth('apikey'))}" placeholder="Enter API key">
      </label>
      <p class="param-desc">Stored in this tab's session storage as <code>apikey</code>. Used for apiKey-type schemes and as the Basic auth credential (<code>Authorization: Basic &lt;API Key&gt;</code>).</p>
      <label class="tryit-field">
        <span>OAuth Key (access token)</span>
        <input type="text" class="input" id="auth-global-oauthkey" value="${escapeHtml(getStoredAuth('oauthkey'))}" placeholder="Bearer token (auto-filled by Authorize/Get token below)">
      </label>
      <p class="param-desc">Stored in this tab's session storage as <code>oauthkey</code>. Used for oauth2-type schemes as <code>Authorization: Bearer &lt;OAuth Key&gt;</code>.</p>
    </div>
  `;
}

function renderAuthDialogBody() {
  const defs = doc.securityDefinitions;
  const keys = Object.keys(defs);
  const globalFields = renderAuthGlobalFields();
  if (!keys.length) return globalFields + '<p class="text-muted">No authorization schemes defined by this spec.</p>';

  const schemeCards = keys.map(key => {
    const def = defs[key];
    const current = authValues.get(key) || {};
    let fields = '';

    if (def.type === 'apiKey') {
      fields = `<p class="param-desc">Sent as ${def.in === 'query' ? 'query parameter' : 'header'} <code>${escapeHtml(def.name)}</code> using the <strong>API Key</strong> above.</p>`;
    } else if (def.type === 'basic') {
      fields = `<p class="param-desc">Sent as <code>Authorization: Basic &lt;API Key&gt;</code> using the <strong>API Key</strong> above.</p>`;
    } else if (def.type === 'oauth2') {
      const scopes = Object.entries(def.scopes || {});
      const selectedScopes = current.scopes || Object.keys(def.scopes || {});
      const flow = def.flow;
      const usesPopup = flow === 'implicit' || flow === 'accessCode';
      const usesClientSecret = flow === 'accessCode' || flow === 'application' || flow === 'password';

      fields = `
        <p class="param-desc">Flow: <code>${escapeHtml(OAUTH_FLOW_LABELS[flow] || flow || 'unknown')}</code></p>
        ${scopes.length ? `
          <div class="tryit-field">
            <span>Scopes</span>
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
        <label class="tryit-field">
          <span>Client ID</span>
          <input type="text" class="input" data-auth-key="${escapeHtml(key)}" data-auth-field="clientId" value="${escapeHtml(current.clientId || '')}">
        </label>
        ${usesClientSecret ? `
          <label class="tryit-field">
            <span>Client secret</span>
            <input type="password" class="input" data-auth-key="${escapeHtml(key)}" data-auth-field="clientSecret" value="${escapeHtml(current.clientSecret || '')}">
          </label>
          <p class="param-desc">Client secret is sent from the browser &mdash; only use a test/dev OAuth client here.</p>
        ` : ''}
        ${flow === 'password' ? `
          <label class="tryit-field">
            <span>Username</span>
            <input type="text" class="input" data-auth-key="${escapeHtml(key)}" data-auth-field="username" value="${escapeHtml(current.username || '')}">
          </label>
          <label class="tryit-field">
            <span>Password</span>
            <input type="password" class="input" data-auth-key="${escapeHtml(key)}" data-auth-field="password" value="${escapeHtml(current.password || '')}">
          </label>
        ` : ''}
        ${flow ? `
          <div class="tryit-field">
            <button type="button" class="btn-outline btn-sm" data-oauth-authorize="${escapeHtml(key)}">${usesPopup ? 'Authorize' : 'Get token'}</button>
            <p class="param-desc" data-oauth-status="${escapeHtml(key)}"></p>
          </div>
        ` : ''}
        <p class="param-desc">Fills the <strong>OAuth Key</strong> field above on success${getStoredAuth('oauthkey') ? ` &mdash; currently set (ends in …${escapeHtml(getStoredAuth('oauthkey').slice(-4))})` : ' — not set yet'}.</p>
      `;
    } else {
      fields = `<p class="param-desc">Unsupported scheme type: ${escapeHtml(def.type)}</p>`;
    }

    return `
      <details class="param-group auth-scheme" open>
        <summary>
          <span>${escapeHtml(key)}</span>
          <span class="badge" data-variant="outline">${escapeHtml(def.type)}</span>
        </summary>
        ${def.description ? `<p class="param-desc">${escapeHtml(def.description)}</p>` : ''}
        ${fields}
      </details>
    `;
  }).join('');

  return globalFields + schemeCards;
}

function openAuthDialog() {
  els.authDialogBody.innerHTML = renderAuthDialogBody();
  els.authDialog.showModal();
}

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
    const res = await fetch(tokenUrl, {
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
    const authUrl = new URL(def.authorizationUrl);
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

  els.authBtn.addEventListener('click', openAuthDialog);

  els.authDialog.addEventListener('click', (e) => {
    if (e.target.closest('[data-action="auth-cancel"]')) els.authDialog.close();
    else if (e.target.closest('[data-action="auth-save"]')) saveAuthDialog();
    else {
      const oauthBtn = e.target.closest('[data-oauth-authorize]');
      if (oauthBtn) startOAuth2Flow(oauthBtn.dataset.oauthAuthorize);
    }
  });

  window.addEventListener('message', handleOAuth2Message);

  els.main.addEventListener('click', (e) => {
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
      els.tryit.querySelectorAll('.lang-btn').forEach(b => b.classList.toggle('active', b.dataset.lang === currentLang));
      updateSnippet(op);
      return;
    }

    if (e.target.closest('[data-action="copy-snippet"]')) {
      copyText(document.getElementById('tryit-snippet').textContent, e.target.closest('button'));
      return;
    }

    if (e.target.closest('[data-action="send"]')) {
      sendRequest(op);
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


