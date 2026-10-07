# Implementation Plan: F075 UMKM Cash Flow Forecast & Early Warning

**Date:** 2026-10-08

**Feature ID:** F075

**Target Module:** UMKM (`apps/umkm/accounting`, `frontend/umkm-web`)

**Status:** Waiting for SPEC Approval

## Goal

Provide a tenant-scoped 30-day daily forecast of cash balance based on existing journal history, plus an in-app early warning for any projected non-positive balance. This complements the existing historical cash-flow report without writing accounting data.

## Architecture and boundaries

- Reuse the cash-flow report's tenant scope, journal inclusion rules, asset cash accounts (`100`, `101`), and current-cash calculation.
- Add a small backend handler/query file and a pure forecast-calculation file; avoid making `main.go` or existing report handlers larger beyond the file-size limit.
- Aggregate a maximum of 90 complete calendar days. Use ordinary least squares with an intercept, linear time index, and weekday indicators; require 56 calendar days.
- Add the forecast view to the existing UMKM financial report UI; do not add a database migration, external notifications, LLM use, or automatic transactions.
- This implementation plan is a proposal and is blocked until the detailed SPEC is explicitly approved.

## Step-by-step plan

### Step 1: Backend query and forecast model

- Query only the authenticated tenant's journal lines for cash accounts 100/101 using parameterized SQL and bounded dates.
- Aggregate daily debit-minus-credit movements, filling missing calendar dates with zero.
- Return an insufficient-data status when fewer than 56 calendar days are available.
- Fit deterministic linear trend and weekday effects, project 30 days, and cumulatively compute closing cash balances from current cash.
- Detect the first projected closing balance at or below zero.

### Step 2: API and unit tests

- Register `GET /api/umkm/reports/cash-flow/forecast` with existing authentication, tenant, and response conventions.
- Return the standard API envelope with status, cut-off, history length, current cash, ordered forecast, risk state, and method metadata.
- Add unit tests for trend, weekday pattern, determinism, zero-movement dates, insufficient history, risk detection, and date boundaries.
- Add tenant-isolation coverage using the repository's established handler/data test patterns.

### Step 3: Frontend report view

- Add a typed client method for the endpoint.
- Present current cash and a daily 30-day projected balance chart/table in the financial reports area.
- Render explicit loading, error, insufficient-data, and at-risk states.
- Display the forecast cut-off, model summary, and limitations disclaimer; reuse the existing IDR formatter.

### Step 4: Verification and documentation

- Run focused Go tests and frontend type/build checks for changed code.
- Run the applicable broader `make check` if focused verification passes and runtime permits.
- Verify tenant isolation, file-size limits, and consistency with the historical cash-flow balance.
- Update the F075 implementation status only after implementation and verification; refresh the graph with `graphify update .`.

## Out of scope

- Holiday adjustments, upcoming bills/receivables, scenario planning, user-configurable thresholds, external alerts, LLM-generated advice, and automatic journal entries.
- Any modification to archived Crypto.
