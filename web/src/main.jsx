import React, { useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import './styles.css';

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080').replace(/\/$/, '');
const STORAGE_KEY = 'case-outgoings.case-ids';
const money = (amount, currency = 'GBP') => new Intl.NumberFormat('en-GB', { style: 'currency', currency }).format(amount);
const shortId = (id) => `…${id.slice(-8)}`;
const outgoingLabel = (key) => key.replace(/^outgoings_/, '').replaceAll('_', ' ').replace(/\b\w/g, (char) => char.toUpperCase());

function Icon({ name }) {
  const common = { width: 17, height: 17, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 1.7, strokeLinecap: 'round', strokeLinejoin: 'round', 'aria-hidden': true };
  if (name === 'search') return <svg {...common}><circle cx="10.8" cy="10.8" r="6.8"/><path d="m16 16 4.4 4.4"/></svg>;
  if (name === 'arrow') return <svg {...common}><path d="M5 12h14m-6-6 6 6-6 6"/></svg>;
  if (name === 'close') return <svg {...common}><path d="m6 6 12 12M18 6 6 18"/></svg>;
  if (name === 'chevron') return <svg {...common}><path d="m9 18 6-6-6-6"/></svg>;
  if (name === 'remove') return <svg {...common}><path d="M5 12h14"/></svg>;
  return <svg {...common}><path d="M4 7h16M4 12h16M4 17h16"/></svg>;
}

function readSavedIds() {
  try {
    const ids = JSON.parse(localStorage.getItem(STORAGE_KEY) || '[]');
    return Array.isArray(ids) ? ids.filter((id) => typeof id === 'string') : [];
  } catch { return []; }
}

function App() {
  const [mode, setMode] = useState('table');
  const [query, setQuery] = useState('');
  const [cases, setCases] = useState([]);
  const [savedIds, setSavedIds] = useState(readSavedIds);
  const [loadingId, setLoadingId] = useState('');
  const [error, setError] = useState('');
  const [selected, setSelected] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return cases;
    return cases.filter((item) => [item.case_id, item.region, item.postcode].some((value) => String(value || '').toLowerCase().includes(needle)));
  }, [cases, query]);

  const persistIds = (ids) => {
    setSavedIds(ids);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(ids));
  };

  const fetchCase = async (rawId) => {
    const caseId = rawId.trim();
    if (!caseId) { setError('Enter a case ID to look up.'); return; }
    if (cases.some((item) => item.case_id === caseId)) { setError('This case is already in your list.'); return; }
    setLoadingId(caseId);
    setError('');
    try {
      const response = await fetch(`${API_BASE_URL}/cases/${encodeURIComponent(caseId)}/outgoings`, { headers: { Accept: 'application/json' } });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        if (response.status === 404) throw new Error('No case was found for that ID. Check the ID and try again.');
        if (response.status === 422) throw new Error(payload.error || 'A region could not be resolved for this case.');
        throw new Error(payload.error || `The API returned an error (${response.status}).`);
      }
      if (!payload.case_id || !payload.outgoings || typeof payload.outgoings !== 'object') throw new Error('The API response did not include case and outgoing details.');
      setCases((current) => [...current, payload]);
      persistIds([...new Set([...savedIds, payload.case_id])]);
      setQuery('');
    } catch (caught) {
      setError(caught instanceof TypeError ? `Could not reach the API at ${API_BASE_URL}. Check that the server is running and CORS is enabled.` : caught.message || 'The case could not be loaded.');
    } finally { setLoadingId(''); }
  };

  const removeCase = (caseId) => {
    setCases((current) => current.filter((item) => item.case_id !== caseId));
    persistIds(savedIds.filter((id) => id !== caseId));
    if (selected?.case_id === caseId) setSelected(null);
  };

  const openCase = (item) => { setSelected(item); setDetailLoading(true); window.setTimeout(() => setDetailLoading(false), 260); };
  const outgoingRows = selected ? Object.entries(selected.outgoings).filter(([key, value]) => key !== 'outgoings_additional_details' && typeof value === 'number') : [];
  const selectedTotal = outgoingRows.reduce((sum, [, value]) => sum + value, 0);
  const hasQuery = query.trim().length > 0;

  return <main className="shell">
    <header className="topbar"><a className="brand" href="#top" aria-label="Ledger home"><span className="brand-mark"><span/></span><span>ledger<span className="brand-dot">.</span></span></a><div className="top-meta"><span className="env-dot"/>CASE ESTIMATES <span className="meta-divider"/> API CONNECTED</div><button className="avatar" aria-label="Account">JG</button></header>
    <section className="intro" id="top"><div className="intro-main"><div className="breadcrumb"><span>Workspace</span><Icon name="chevron"/><span>Cases</span><Icon name="chevron"/><strong>Outgoings</strong></div><p className="overline">HOUSEHOLD EXPENDITURE</p><h1>Case outgoings<span className="period">/</span></h1><p className="intro-copy">Look up a case you know to view its regional monthly spending estimate.</p></div><div className="intro-aside"><span className="aside-label">SOURCE</span><strong>Live API response</strong><span>One lookup per case ID</span><span className="aside-rule"/><div className="aside-stat"><span>CASES ADDED</span><strong>{String(cases.length).padStart(2, '0')}</strong></div></div></section>
    <section className="workspace" aria-label="Case lookup and presentation">
      <form className="lookup" onSubmit={(event) => { event.preventDefault(); fetchCase(query); }}><label className="lookup-label" htmlFor="case-id">ADD A CASE BY ID</label><div className="lookup-row"><div className="lookup-input"><Icon name="search"/><input id="case-id" value={query} onChange={(event) => { setQuery(event.target.value); setError(''); }} placeholder="Paste a case ID" autoComplete="off"/><span className="input-hint">UUID</span></div><button className="add-button" type="submit" disabled={Boolean(loadingId)}>{loadingId ? <><span className="button-pulse"/>Looking up</> : <>Look up case <Icon name="arrow"/></>}</button></div><p className="lookup-help">A case appears only after the API returns a successful estimate.</p>{error && <div className="error-banner" role="alert">{error}</div>}</form>
      <div className="toolbar"><div className="result-label"><span className="live-dot"/>YOUR CASES <span className="result-count">{String(filtered.length).padStart(2, '0')}</span></div><div className="toolbar-controls">{cases.length > 0 && <label className="search"><Icon name="search"/><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Filter added cases" aria-label="Filter added cases"/></label>}<div className="segmented" role="group" aria-label="Presentation style"><button className={mode === 'table' ? 'active' : ''} onClick={() => setMode('table')}><Icon name="table"/><span>Table</span></button><button className={mode === 'cards' ? 'active' : ''} onClick={() => setMode('cards')}><span className="grid-icon"><i/><i/><i/><i/></span><span>Cards</span></button></div></div></div>
      {filtered.length === 0 ? <div className="empty"><div className="empty-symbol"><Icon name={hasQuery ? 'search' : 'table'}/></div><h2>{cases.length === 0 ? 'No cases added yet' : 'No matching cases'}</h2><p>{cases.length === 0 ? 'Enter a case ID above to fetch its estimate from the API.' : 'Try another case ID, region, or postcode.'}</p>{cases.length > 0 && <button onClick={() => setQuery('')}>Clear filter</button>}</div> : mode === 'table' ? <div className="table-wrap"><table><thead><tr><th>CASE ID</th><th>REGION</th><th>POSTCODE</th><th>PERIOD</th><th>CURRENCY</th><th className="align-right">EST. MONTHLY</th><th><span className="sr-only">Actions</span></th></tr></thead><tbody>{filtered.map((item, index) => { const total = Object.entries(item.outgoings).filter(([key, value]) => key !== 'outgoings_additional_details' && typeof value === 'number').reduce((sum, [, value]) => sum + value, 0); return <tr key={item.case_id} style={{ '--row': index }}><td><span className="case-code" title={item.case_id}>{shortId(item.case_id)}</span></td><td>{item.region}</td><td><span className="postcode">{item.postcode}</span></td><td><span className="period-tag">{item.period}</span></td><td>{item.currency}</td><td className="align-right amount">{money(total, item.currency)}</td><td className="action-cell"><button className="row-action" onClick={() => openCase(item)} aria-label={`View outgoings for ${item.case_id}`} title="View outgoings"><Icon name="arrow"/></button><button className="remove-action" onClick={() => removeCase(item.case_id)} aria-label={`Remove ${item.case_id}`} title="Remove case"><Icon name="remove"/></button></td></tr>; })}</tbody></table><div className="table-foot"><span>Showing <strong>{filtered.length}</strong> added case{filtered.length === 1 ? '' : 's'}</span><span>Data returned by <strong>case API</strong></span></div></div> : <div className="case-list">{filtered.map((item, index) => { const total = Object.entries(item.outgoings).filter(([key, value]) => key !== 'outgoings_additional_details' && typeof value === 'number').reduce((sum, [, value]) => sum + value, 0); return <article className="case-card" key={item.case_id} style={{ '--row': index }}><button className="card-open" onClick={() => openCase(item)} aria-label={`Open outgoings for ${item.case_id}`}><div className="card-top"><span className="card-index">CASE {String(index + 1).padStart(2, '0')}</span><span className="card-action">OUTGOINGS <Icon name="arrow"/></span></div><div className="card-id" title={item.case_id}>{shortId(item.case_id)}</div><div className="card-region">{item.region}</div><div className="card-bottom"><div><span>POSTCODE</span><strong>{item.postcode}</strong></div><div><span>PERIOD</span><strong>{item.period}</strong></div><div><span>CURRENCY</span><strong>{item.currency}</strong></div><div className="card-total"><span>MONTHLY ESTIMATE</span><strong>{money(total, item.currency)}</strong></div></div></button><button className="card-remove" onClick={() => removeCase(item.case_id)} aria-label={`Remove ${item.case_id}`}><Icon name="remove"/> Remove</button></article>; })}</div>}
      <div className="view-note"><span className="note-line"/><span>{mode === 'table' ? 'Scan and compare the cases you have added.' : 'Open each case as a distinct record.'}</span><span className="api-note">LIVE RESPONSE DATA</span></div>
    </section>
    <footer className="footer"><span>CASE DATA PRESENTATION <span className="meta-divider"/> {API_BASE_URL}</span><span>CASE IDS SAVED IN THIS BROWSER</span></footer>
    {selected && <div className="overlay" onMouseDown={(event) => { if (event.target === event.currentTarget) setSelected(null); }}><section className="drawer" role="dialog" aria-modal="true" aria-label={`Outgoings for ${selected.case_id}`}><div className="drawer-header"><div><span className="drawer-kicker">API CASE DETAIL</span><strong title={selected.case_id}>{shortId(selected.case_id)}</strong></div><button className="close-button" onClick={() => setSelected(null)} aria-label="Close details"><Icon name="close"/></button></div><div className="drawer-body">{detailLoading ? <div className="skeleton"><div/><div/><div/><div/><div/></div> : <div className="detail"><div className="detail-head"><div><p className="eyebrow">{selected.region} · {selected.postcode}</p><h2>{selected.period} outgoings</h2></div><div className="detail-total"><span>Estimated total</span><strong>{money(selectedTotal, selected.currency)}</strong></div></div><div className="outgoing-grid">{outgoingRows.map(([key, value]) => <div className="outgoing-row" key={key}><span>{outgoingLabel(key)}</span><strong>{money(value, selected.currency)}</strong></div>)}</div>{selected.outgoings.outgoings_additional_details && <p className="additional-details">{selected.outgoings.outgoings_additional_details}</p>}<div className="source-line"><span className="source-mark"/>{selected.source || 'Source details returned by case API'}{selected.uk_fallback_codes?.length > 0 && ` · UK average used for ${selected.uk_fallback_codes.length} categories`}</div></div>}</div><div className="drawer-footer"><span><span className="live-dot"/>ESTIMATE FROM API</span><button onClick={() => removeCase(selected.case_id)}>Remove case</button></div></section></div>}
  </main>;
}

createRoot(document.getElementById('root')).render(<App/>);
