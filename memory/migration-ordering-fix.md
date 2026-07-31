---
name: migration-ordering-fix
description: Fix for migration 000070/000071/000084 ordering and dependency issues
metadata:
  type: project
---

**Root cause:**

1. **Migration 000070** (`wa_credential_verification`) attempted to `ALTER TABLE wa_cloud_api_credentials` to add verification columns. But `wa_cloud_api_credentials` table was only created in migration **000075**. Since migrations run in ascending order, migration 70 would fail on clean databases.

2. **Migration 000071** had two files with the same version number:
   - `000071_campaign_rag_documents` (CREATE TABLE campaign_documents)
   - `000071_rename_cents_to_rupiah` (ALTER TABLE on non-existent tables)

   The rename migration referenced tables `plan_prices`, `subscriptions`, `addon_purchases`, and `affiliate_payouts` that were never created by any migration.

**Fixes applied:**

1. Deleted `000070_wa_credential_verification.{up,down}.sql` — merged its logic into `000075_wa_cloud_api_credentials.up.sql` which creates the table with all columns from the start

2. Deleted `000071_rename_cents_to_rupiah.{up,down}.sql` — references tables that don't exist (plan_prices, subscriptions, addon_purchases, affiliate_payouts)

3. Added missing `.down.sql` files for migrations 58, 74, 76, 78, 79

4. Updated `CLAUDE.md` to reference the correct migration number (75) for verification columns

**Verification:**
- No duplicate migration versions
- All ALTER TABLE statements reference existing tables
- All migrations have corresponding up and down files
- No Go code references migration 70 directly
- Docker build copies migrations correctly via `COPY . .`

**How to apply:**
These changes only affect fresh database setups. For existing databases:
- If migration 70 was never applied (because it failed), the merged migration 75 will create the table with all columns from the start
- If migration 75 was already applied, it will use `CREATE TABLE IF NOT EXISTS` and `IF NOT EXISTS` on indexes — no error

**Related files modified:**
- `shared/migrations/000075_wa_cloud_api_credentials.{up,down}.sql` (merged columns)
- `CLAUDE.md` (updated reference)
- Added: `000058`, `000074`, `000076`, `000078`, `000079` down files
