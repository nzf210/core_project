# F075: UMKM Cash Flow Forecast & Early Warning

**Spec Status:** 🔍 In Review

**Implementation:** ⏸ Not Started

**Last Updated:** 2026-10-08
**Target:** UMKM (`apps/umkm/accounting`, `frontend/umkm-web`)

## Objective

Help an UMKM owner anticipate a possible cash shortfall using a transparent estimate derived from the tenant's own accounting history. Existing cash-flow reports show actuals; F075 adds a bounded forward view without generating or changing accounting transactions.

## User story

As an UMKM owner, I want to see estimated daily cash balances for the next 30 days and an explicit warning if the estimate reaches zero or below, so I can prepare for a possible shortfall.

## Scope

- Provide a 30-calendar-day daily forecast for the authenticated tenant.
- Use up to the latest 90 complete calendar days of cash movements and require at least 56 calendar days of available accounting history.
- Estimate daily net cash with a linear trend and a seven-day weekday seasonal factor.
- Display current cash, daily projected closing balances, history/cut-off dates, method, and an estimate disclaimer.
- Display an in-app warning if any projected closing balance is zero or negative.
- Do not send WhatsApp, email, or other external alerts; do not create transactions; do not use an LLM; do not persist forecast results.
- Do not adjust for public holidays, planned payments, or manually entered future commitments in this version.

## Data and calculation

1. Use the same tenant scoping, journal inclusion rules, and cash-account definition as the existing cash-flow report. Cash accounts are asset accounts with codes `100` and `101`.
2. Let the latest complete accounting date be the forecast cut-off. Aggregate each historical calendar date's net cash movement as the sum of `debit - credit` for those accounts. Fill dates without movements with zero. Use at most the 90 calendar days ending at the cut-off.
3. Require at least 56 calendar days between the first eligible cash journal date and the cut-off. If fewer are available, return `insufficient_data`; do not return a valid forecast.
4. Fit an ordinary least-squares model to the daily net movement series with an intercept, a linear day index, and six weekday indicator terms (Monday is the baseline). The model therefore represents a linear trend plus a seven-day seasonal pattern. Do not include holiday adjustments.
5. Predict the next 30 daily net movements using the fitted model. Starting from the current cash balance at the cut-off, add each predicted daily movement cumulatively to produce daily projected closing balances.
6. Return an at-risk result if any projected closing balance is less than or equal to zero. Do not imply certainty; disclose that unrecorded future transactions and holidays are not modeled.
7. Keep monetary values in integer rupiah minor units (`int64` sen) internally and use the existing UMKM currency formatter in the UI.

## API draft

`GET /api/umkm/reports/cash-flow/forecast`

The response should include:
- `status`: `ready` or `insufficient_data`
- `as_of`: last complete accounting date included
- `history_days`: number of calendar days used
- `forecast_days`: 30 when status is `ready`
- `current_cash`: cash balance at the cut-off
- `daily_forecast`: ordered date and projected closing balance points (empty when insufficient)
- `risk`: whether any projected closing balance is zero or negative
- `risk_date`: first date at or below zero, or null
- `method`: user-readable description of linear trend + weekday seasonality

The exact existing API envelope should be preserved. The endpoint must use the standard auth and tenant middleware.

## UI draft

Add a forecast view to the existing financial reports area. Show a daily chart/table of projected closing cash, current balance, date range, method and data sufficiency. Highlight the first non-positive balance date. For insufficient data, show the required-history explanation instead of an empty chart. Include a visible estimation disclaimer.

## Acceptance criteria

- [ ] AC-1: The tenant-scoped forecast endpoint returns current cash and 30 ordered daily projected closing balances when at least 56 days of history are available.
- [ ] AC-2: Historical movement uses asset cash accounts `100` and `101`, and daily movement is debit minus credit; missing dates are included as zero movement.
- [ ] AC-3: The forecast uses a linear trend and weekday seasonality, is deterministic for the same input series, and does not call an external AI service.
- [ ] AC-4: Fewer than 56 calendar days of history returns `insufficient_data`, with no valid daily forecast.
- [ ] AC-5: `risk` is true exactly when at least one projected closing balance is less than or equal to zero; `risk_date` is the earliest matching date.
- [ ] AC-6: Tenant A's request cannot read or incorporate Tenant B's journal lines.
- [ ] AC-7: The UI explains its history cut-off and estimation limits and displays insufficient data and risk states accessibly.
- [ ] AC-8: Unit tests cover model trend, weekday effects, determinism, zero-movement dates, insufficient history, non-positive balance detection, date boundaries, and tenant isolation.
- [ ] AC-9: No migration or persisted forecast table is introduced.

## Planned files after approval

- `apps/umkm/accounting/cash_flow_forecast_handlers.go` — endpoint and bounded tenant-scoped data retrieval.
- `apps/umkm/accounting/cash_flow_forecast.go` — deterministic model and typed results.
- `apps/umkm/accounting/cash_flow_forecast_test.go` — model and result tests.
- `apps/umkm/accounting/main.go` — route registration only.
- `frontend/umkm-web/src/api.ts` — typed forecast client.
- `frontend/umkm-web/src/components/Reports.vue` — forecast visualization and warning state.

## Dependencies and risks

- Reuse the existing cash-flow report semantics and ensure the current-cash starting point reconciles with the accounting report.
- The linear/weekday model is an estimate, not a financial guarantee. The UI must identify the cut-off and disclose omitted holidays and unrecorded future events.
- All backend files must remain within the project's 450-line limit; the Vue component must remain within 500 lines.
- No code changes may begin until the user explicitly approves this SPEC.
