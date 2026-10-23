import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from './main.jsx';

const idA = '56763bf7-d046-4a54-a4a1-0728c724a092';
const idB = '11111111-1111-4111-8111-111111111111';
const responseA = {
  case_id: idA,
  region: 'East Midlands',
  postcode: 'NG1 2ET',
  period: 'monthly',
  currency: 'GBP',
  source: 'ONS Family Spending 2019',
  outgoings: { outgoings_food: 265.2, outgoings_water: 37.27, outgoings_additional_details: 'ONS estimate' },
};
const responseB = {
  case_id: idB,
  region: 'London',
  postcode: 'SW1A 2AA',
  period: 'monthly',
  currency: 'GBP',
  source: 'ONS Family Spending 2019',
  outgoings: { outgoings_food: 410.5, outgoings_water: 42.1 },
};
const apiResponse = (data, status = 200) => ({ ok: status >= 200 && status < 300, status, json: async () => data });

beforeEach(() => {
  vi.stubGlobal('fetch', vi.fn());
});

describe('case outgoings preview', () => {
  it('starts with no cases and makes no API requests without saved IDs', async () => {
    render(<App />);
    expect(await screen.findByRole('heading', { name: 'No cases added yet' })).toBeInTheDocument();
    expect(fetch).not.toHaveBeenCalled();
  });

  it('adds a case only from a successful API response and displays its returned fields', async () => {
    fetch.mockResolvedValueOnce(apiResponse(responseA));
    const user = userEvent.setup();
    render(<App />);

    await user.type(screen.getByPlaceholderText('Paste a case ID'), idA);
    await user.click(screen.getByRole('button', { name: /look up case/i }));

    expect(await screen.findByText('East Midlands')).toBeInTheDocument();
    expect(screen.getByText('NG1 2ET')).toBeInTheDocument();
    expect(screen.getByText('GBP')).toBeInTheDocument();
    expect(screen.getByText('£302.47')).toBeInTheDocument();
    expect(JSON.parse(localStorage.getItem('case-outgoings.case-ids'))).toEqual([idA]);
    expect(fetch).toHaveBeenCalledWith(`http://localhost:8080/cases/${idA}/outgoings`, { headers: { Accept: 'application/json' } });
  });

  it('shows a not-found error and does not persist an unknown case', async () => {
    fetch.mockResolvedValueOnce(apiResponse({ error: 'case not found' }, 404));
    const user = userEvent.setup();
    render(<App />);

    await user.type(screen.getByPlaceholderText('Paste a case ID'), idB);
    await user.click(screen.getByRole('button', { name: /look up case/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('No case was found');
    expect(localStorage.getItem('case-outgoings.case-ids')).toBeNull();
    expect(screen.queryByText('London')).not.toBeInTheDocument();
  });

  it('restores browser-saved IDs by refetching them from the API', async () => {
    localStorage.setItem('case-outgoings.case-ids', JSON.stringify([idA]));
    fetch.mockResolvedValueOnce(apiResponse(responseA));

    render(<App />);

    expect(await screen.findByText('East Midlands')).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith(`http://localhost:8080/cases/${idA}/outgoings`, { headers: { Accept: 'application/json' } });
  });

  it('supports searching and dropdown filters across API-loaded cases', async () => {
    fetch.mockResolvedValueOnce(apiResponse(responseA)).mockResolvedValueOnce(apiResponse(responseB));
    const user = userEvent.setup();
    render(<App />);

    await user.type(screen.getByPlaceholderText('Paste a case ID'), idA);
    await user.click(screen.getByRole('button', { name: /look up case/i }));
    await screen.findByText('East Midlands');
    await user.type(screen.getByPlaceholderText('Paste a case ID'), idB);
    await user.click(screen.getByRole('button', { name: /look up case/i }));
    await screen.findByText('London');
    await user.click(screen.getByRole('button', { name: /filters/i }));
    await user.selectOptions(screen.getByLabelText('Region'), 'London');

    expect(screen.getByRole('row', { name: /London/ })).toBeInTheDocument();
    expect(screen.queryByRole('row', { name: /East Midlands/ })).not.toBeInTheDocument();
    await user.type(screen.getByRole('textbox', { name: 'Search cases' }), 'SW1A');
    expect(screen.getByRole('row', { name: /London/ })).toBeInTheDocument();
    expect(screen.queryByRole('row', { name: /East Midlands/ })).not.toBeInTheDocument();
  });

  it('requires confirmation before removing a case and updates browser storage only on yes', async () => {
    fetch.mockResolvedValueOnce(apiResponse(responseA));
    const user = userEvent.setup();
    render(<App />);
    await user.type(screen.getByPlaceholderText('Paste a case ID'), idA);
    await user.click(screen.getByRole('button', { name: /look up case/i }));
    await screen.findByText('East Midlands');

    await user.click(screen.getByRole('button', { name: `Remove ${idA}` }));
    expect(screen.getByRole('alertdialog')).toHaveTextContent(idA);
    await user.click(screen.getByRole('button', { name: 'No' }));
    expect(screen.getByText('East Midlands')).toBeInTheDocument();
    expect(JSON.parse(localStorage.getItem('case-outgoings.case-ids'))).toEqual([idA]);

    await user.click(screen.getByRole('button', { name: `Remove ${idA}` }));
    await user.click(screen.getByRole('button', { name: 'Yes, remove case' }));
    expect(await screen.findByRole('heading', { name: 'No cases added yet' })).toBeInTheDocument();
    expect(localStorage.getItem('case-outgoings.case-ids')).toBe('[]');
  });

  it('opens API case details and supports the cards presentation', async () => {
    fetch.mockResolvedValueOnce(apiResponse(responseA));
    const user = userEvent.setup();
    render(<App />);
    await user.type(screen.getByPlaceholderText('Paste a case ID'), idA);
    await user.click(screen.getByRole('button', { name: /look up case/i }));
    await screen.findByText('East Midlands');
    await user.click(screen.getByRole('button', { name: 'Cards' }));
    expect(screen.getByText('NG1 2ET')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: `Open outgoings for ${idA}` }));
    expect(await screen.findByRole('dialog', { name: `Outgoings for ${idA}` })).toBeInTheDocument();
    expect(await screen.findByText('Food')).toBeInTheDocument();
    expect(screen.getByText('£265.20')).toBeInTheDocument();
  });
});
