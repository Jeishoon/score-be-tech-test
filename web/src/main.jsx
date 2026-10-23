import React, { useEffect, useMemo, useState } from 'react';
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
  const [filter, setFilter] = useState('');
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [filters, setFilters] = useState({ region: '', postcode: '', currency: '', period: '', estimate: '' });
  const [cases, setCases] = useState([]);
  const [restoring, setRestoring] = useState(true);
  const [savedIds, setSavedIds] = useState(readSavedIds);
  const [loadingId, setLoadingId] = useState('');
  const [error, setError] = useState('');
  const [selected, setSelected] = useState(null);
  const [confirmRemoval, setConfirmRemoval] = useState('');
  const [detailLoading, setDetailLoading] = useState(false);

  const filtered = useMemo(() => {
    const needle = filter.trim().toLowerCase();
    return cases.filter((item) => {
      const matchesSearch = !needle || [item.case_id, item.region, item.postcode].some((value) => String(value || '').toLowerCase().includes(needle));
      const total = Object.entries(item.outgoings).filter(([key, value]) => key !== 'outgoings_additional_details' && typeof value === 'number').reduce((sum, [, value]) => sum + value, 0);
      const matchesEstimate = !filters.estimate || (filters.estimate === 'under-1000' && total < 1000) || (filters.estimate === '1000-2000' && total >= 1000 && total < 2000) || (filters.estimate === '2000-3000' && total >= 2000 && total < 3000) || (filters.estimate === 'over-3000' && total >= 3000);
      return matchesSearch && (!filters.region || item.region === filters.region) && (!filters.postcode || item.postcode === filters.postcode) && (!filters.currency || item.currency === filters.currency) && (!filters.period || item.period === filters.period) && matchesEstimate;
    });
  }, [cases, filter, filters]);

  const filterOptions = (key) => [...new Set(cases.map((item) => item[key]).filter(Boolean))].sort();
  const activeFilterCount = Object.values(filters).filter(Boolean).length;
  const clearFilters = () => { setFilter(''); setFilters({ region: '', postcode: '', currency: '', period: '', estimate: '' }); };

  useEffect(() => {
    let cancelled = false;
    const restore = async () => {
      const results = await Promise.all(savedIds.map(async (id) => {
        try {
          const response = await fetch(`${API_BASE_URL}/cases/${encodeURIComponent(id)}/outgoings`, { headers: { Accept: 'application/json' } });
          if (!response.ok) return null;
          const payload = await response.json();
          return payload.case_id && payload.outgoings ? payload : null;
        } catch { return null; }
      }));
      if (!cancelled) {
        const restored = results.filter(Boolean);
        setCases(restored);
        const validIds = restored.map((item) => item.case_id);
        if (validIds.length !== savedIds.length) {
          setSavedIds(validIds);
          localStorage.setItem(STORAGE_KEY, JSON.stringify(validIds));
        }
        setRestoring(false);
      }
    };
    restore();
    return () => { cancelled = true; };
  }, []);

  const persistIds = (ids) => {
    setSavedIds(ids);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(ids));
  };

  const fetchCase = async (rawId) => {
    const caseId = rawId.trim();
    if (!caseId) { setError('Enter a case ID to look up.'); return; }
    if (savedIds.includes(caseId)) { setError('This case is already in your list.'); return; }
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
  const hasQuery = filter.trim().length > 0;

  return <main className="shell">
    <header className="topbar"><a className="brand" href="#top" aria-label="Ledger home"><span className="brand-mark"><span/></span><span>ledger<span className="brand-dot">.</span></span></a></header>
    <section className="intro" id="top"><div className="intro-main"><div className="breadcrumb"><span>Cases</span><Icon name="chevron"/><strong>Outgoings</strong></div><p className="overline">HOUSEHOLD EXPENDITURE</p><h1>Case outgoings<span className="period">/</span></h1><p className="intro-copy">Look up a case you know to view its regional monthly spending estimate.</p></div></section>
    <section className="workspace" aria-label="Case lookup and presentation">
      <form className="lookup" onSubmit={(event) => { event.preventDefault(); fetchCase(query); }}><label className="lookup-label" htmlFor="case-id">ADD A CASE BY ID</label><div className="lookup-row"><div className="lookup-input"><Icon name="search"/><input id="case-id" value={query} onChange={(event) => { setQuery(event.target.value); setError(''); }} placeholder="Paste a case ID" autoComplete="off"/><span className="input-hint">UUID</span></div><button className="add-button" type="submit" disabled={Boolean(loadingId)}>{loadingId ? <><span className="button-pulse"/>Looking up</> : <>Look up case <Icon name="arrow"/></>}</button></div>{error && <div className="error-banner" role="alert">{error}</div>}</form>
      <div className="toolbar"><div className="result-label">YOUR CASES <span className="result-count">{String(filtered.length).padStart(2, '0')}</span></div><div className="toolbar-controls"><button className={`filter-toggle ${filtersOpen ? 'active' : ''}`} type="button" onClick={() => setFiltersOpen((open) => !open)}><Icon name="search"/> Filters{activeFilterCount > 0 && <span>{activeFilterCount}</span>}</button><div className="segmented" role="group" aria-label="Presentation style"><button className={mode === 'table' ? 'active' : ''} onClick={() => setMode('table')}><Icon name="table"/><span>Table</span></button><button className={mode === 'cards' ? 'active' : ''} onClick={() => setMode('cards')}><span className="grid-icon"><i/><i/><i/><i/></span><span>Cards</span></button></div></div></div>
      {filtersOpen && <div className="filter-panel"><label className="search filter-search"><Icon name="search"/><input value={filter} onChange={(event) => setFilter(event.target.value)} placeholder="Search ID, region, postcode" aria-label="Search cases"/></label>
        {[['region', 'Region'], ['postcode', 'Postcode'], ['currency', 'Currency'], ['period', 'Period']].map(([key, label]) => <label className="filter-field" key={key}><span>{label}</span><select value={filters[key]} onChange={(event) => setFilters((current) => ({ ...current, [key]: event.target.value }))}><option value="">All {label === 'Currency' ? 'currencies' : `${label.toLowerCase()}s`}</option>{filterOptions(key).map((value) => <option key={value} value={value}>{value}</option>)}</select></label>)}
        <label className="filter-field"><span>Est. monthly</span><select value={filters.estimate} onChange={(event) => setFilters((current) => ({ ...current, estimate: event.target.value }))}><option value="">Any amount</option><option value="under-1000">Under £1,000</option><option value="1000-2000">£1,000–£1,999</option><option value="2000-3000">£2,000–£2,999</option><option value="over-3000">£3,000 and over</option></select></label>
        {activeFilterCount > 0 || filter ? <button className="clear-filters" type="button" onClick={clearFilters}>Clear filters</button> : null}</div>}
      {restoring ? <div className="restore-state"><div className="skeleton"><div/><div/><div/><div/></div></div> : filtered.length === 0 ? <div className="empty"><div className="empty-symbol"><Icon name={hasQuery ? 'search' : 'table'}/></div><h2>{cases.length === 0 ? 'No cases added yet' : 'No matching cases'}</h2><p>{cases.length === 0 ? 'Enter a case ID above to fetch its estimate from the API.' : 'Try another case ID, region, or postcode.'}</p>{cases.length > 0 && <button onClick={() => setFilter('')}>Clear filter</button>}</div> : mode === 'table' ? <div className="table-wrap"><table><thead><tr><th>CASE ID</th><th>REGION</th><th>POSTCODE</th><th>CURRENCY</th><th className="align-right">EST. MONTHLY</th><th><span className="sr-only">Actions</span></th></tr></thead><tbody>{filtered.map((item, index) => { const total = Object.entries(item.outgoings).filter(([key, value]) => key !== 'outgoings_additional_details' && typeof value === 'number').reduce((sum, [, value]) => sum + value, 0); return <tr key={item.case_id} style={{ '--row': index }}><td><span className="case-code" title={item.case_id}>{shortId(item.case_id)}</span></td><td>{item.region}</td><td><span className="postcode">{item.postcode}</span></td><td>{item.currency}</td><td className="align-right amount">{money(total, item.currency)}</td><td className="action-cell"><button className="row-action" onClick={() => openCase(item)} aria-label={`View outgoings for ${item.case_id}`} title="View outgoings"><Icon name="arrow"/></button><button className="remove-action" onClick={() => setConfirmRemoval(item.case_id)} aria-label={`Remove ${item.case_id}`} title="Remove case"><Icon name="remove"/></button></td></tr>; })}</tbody></table><div className="table-foot"><span>{filtered.length} case{filtered.length === 1 ? '' : 's'}</span></div></div> : <div className="case-list">{filtered.map((item, index) => { const total = Object.entries(item.outgoings).filter(([key, value]) => key !== 'outgoings_additional_details' && typeof value === 'number').reduce((sum, [, value]) => sum + value, 0); return <article className="case-card" key={item.case_id} style={{ '--row': index }}><button className="card-open" onClick={() => openCase(item)} aria-label={`Open outgoings for ${item.case_id}`}><div className="card-top"><span className="card-index">CASE {String(index + 1).padStart(2, '0')}</span><span className="card-action">OUTGOINGS <Icon name="arrow"/></span></div><div className="card-id" title={item.case_id}>{shortId(item.case_id)}</div><div className="card-region">{item.region}</div><div className="card-bottom"><div><span>POSTCODE</span><strong>{item.postcode}</strong></div><div><span>PERIOD</span><strong>{item.period}</strong></div><div><span>CURRENCY</span><strong>{item.currency}</strong></div><div className="card-total"><span>MONTHLY ESTIMATE</span><strong>{money(total, item.currency)}</strong></div></div></button><button className="card-remove" onClick={() => setConfirmRemoval(item.case_id)} aria-label={`Remove ${item.case_id}`}><Icon name="remove"/> Remove</button></article>; })}</div>}
    </section>
    {selected && <div className="overlay" onMouseDown={(event) => { if (event.target === event.currentTarget) setSelected(null); }}><section className="drawer" role="dialog" aria-modal="true" aria-label={`Outgoings for ${selected.case_id}`}><div className="drawer-header"><div><span className="drawer-kicker">Case Detail:</span><strong className="full-case-id" title={selected.case_id}>{selected.case_id}</strong></div><button className="close-button" onClick={() => setSelected(null)} aria-label="Close details"><Icon name="close"/></button></div><div className="drawer-body">{detailLoading ? <div className="skeleton"><div/><div/><div/><div/><div/></div> : <div className="detail"><div className="detail-head"><div><p className="eyebrow">{selected.region} · {selected.postcode}</p><h2>{selected.period} outgoings</h2></div><div className="detail-total"><span>Estimated total</span><strong>{money(selectedTotal, selected.currency)}</strong></div></div><div className="outgoing-grid">{outgoingRows.map(([key, value]) => <div className="outgoing-row" key={key}><span>{outgoingLabel(key)}</span><strong>{money(value, selected.currency)}</strong></div>)}</div>{selected.outgoings.outgoings_additional_details && <p className="additional-details">{selected.outgoings.outgoings_additional_details}</p>}<div className="source-line"><span className="source-mark"/>{selected.source || 'Source details returned by case API'}{selected.uk_fallback_codes?.length > 0 && ` · UK average used for ${selected.uk_fallback_codes.length} categories`}</div></div>}</div><div className="drawer-footer"><button className="remove-case-button" onClick={() => setConfirmRemoval(selected.case_id)}><Icon name="remove"/> Remove case</button></div></section></div>}
    {confirmRemoval && <div className="confirm-overlay" role="presentation"><section className="confirm-dialog" role="alertdialog" aria-modal="true" aria-labelledby="confirm-title" aria-describedby="confirm-case-id"><p className="eyebrow">REMOVE CASE</p><h2 id="confirm-title">Are you sure you want to remove this case?</h2><p id="confirm-case-id" className="confirm-case-id">{confirmRemoval}</p><div className="confirm-actions"><button className="cancel-button" onClick={() => setConfirmRemoval('')}>No</button><button className="confirm-remove-button" onClick={() => { removeCase(confirmRemoval); setConfirmRemoval(''); }}>Yes, remove case</button></div></section></div>}
  </main>;
}

createRoot(document.getElementById('root')).render(<App/>);
