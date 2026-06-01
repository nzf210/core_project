# Database Migration Registry

This document provides a comprehensive overview of all database migrations in the WCH Platform project, covering migrations 000001 through 000024.

---

## Summary Table

| Migration | Name | Purpose | Tables Created/Modified |
|-----------|------|---------|--------------------------|
| 000001 | init_schema | Core authentication and tenant infrastructure | tenants, users, refresh_tokens, ai_usage_logs |
| 000002 | accounting_schema | Double-entry accounting system | chart_of_accounts, journal_entries, journal_lines |
| 000003 | chatbot_schema | Chat session management | chat_sessions, chat_messages |
| 000004 | campaign_schema | Political campaign management | candidates, campaigns, provinces, regencies, districts, villages, tps, volunteers, volunteer_assignments, voters, voter_interactions, events, event_attendees, surveys, survey_questions, survey_responses |
| 000005 | crypto_schema | Crypto trading bot infrastructure | exchange_api_keys, bots, bot_orders, bot_pnl_snapshots |
| 000006 | crypto_bot_real_testing | Grid trading enhancements | bots (modified), bot_grid_lines |
| 000007 | crypto_features | Crypto notification system | crypto_notifications |
| 000008 | user_phone_number | User phone number support | users (modified) |
| 000009 | password_resets | Password reset functionality | password_resets |
| 000010 | tenant_whatsapp_settings | WhatsApp/Fonnte integration | tenants (modified) |
| 000011 | business_types | Business type catalog and tenant onboarding | business_types, tenant_module_config, usage_quotas |
| 000012 | phone_verified_and_email_nullable | Authentication flexibility | users (modified) |
| 000013 | tenant_logo | Tenant branding | tenants (modified) |
| 000014 | faq_and_forwarders | Tenant FAQ and WhatsApp forwarders | tenant_faqs, tenant_forwarders |
| 000015 | add_category_and_stock_to_products | Inventory management | products (modified) |
| 000016 | add_additional_photos_to_products | Product image gallery | products (modified) |
| 000017 | pos_transactions | Point of Sale transactions | pos_transactions |
| 000018 | tenant_xendit_settings | Xendit payment gateway | tenants (modified) |
| 000019 | tenant_domains | Custom domain support | tenants (modified) |
| 000020 | tenant_contacts | Customer contact management | tenant_contacts |
| 000021 | add_tenant_qris_and_report_settings | QRIS and reporting features | tenants (modified) |
| 000022 | add_hotel_business | Hotel/Penginapan business type | business_types (modified) |
| 000023 | add_coupons_table | Coupon/discount system | coupons, tenant_coupons |
| 000024 | tenant_notification_settings | Multi-channel notification preferences | tenant_notification_settings |

---

## Detailed Migration Documentation

---

### Migration 000001: init_schema

**Purpose:** Establishes the core database infrastructure including tenant isolation, user authentication, and AI usage tracking.

**Tables Created:**

#### tenants
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique tenant identifier |
| name | VARCHAR(255) | NOT NULL | Tenant/business name |
| plan | VARCHAR(50) | NOT NULL, DEFAULT 'free' | Subscription plan (free, lite, etc.) |
| api_quota | INT | NOT NULL, DEFAULT 0 | API request quota |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### users
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Unique user identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| username | VARCHAR(100) | NOT NULL, UNIQUE | Username |
| email | VARCHAR(255) | NOT NULL, UNIQUE | Email address |
| password_hash | VARCHAR(255) | NOT NULL | Bcrypt hashed password |
| role | VARCHAR(50) | NOT NULL, DEFAULT 'user' | User role |
| mfa_secret | VARCHAR(255) | NULL | MFA TOTP secret |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### refresh_tokens
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Token identifier |
| user_id | UUID | NOT NULL, FK -> users(id) | Token owner |
| token_hash | VARCHAR(255) | NOT NULL, UNIQUE | Hashed refresh token |
| expires_at | TIMESTAMPTZ | NOT NULL | Token expiration |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### ai_usage_logs
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Log entry identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| model | VARCHAR(100) | NOT NULL | AI model name |
| tokens_in | INT | NOT NULL, DEFAULT 0 | Input tokens |
| tokens_out | INT | NOT NULL, DEFAULT 0 | Output tokens |
| cost_usd | DECIMAL(10,4) | NOT NULL, DEFAULT 0.0000 | Cost in USD |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

**Relationships:**
- users.tenant_id -> tenants.id (CASCADE)
- refresh_tokens.user_id -> users.id (CASCADE)
- ai_usage_logs.tenant_id -> tenants.id (CASCADE)

**Indexes:**
- idx_users_tenant_id ON users(tenant_id)
- idx_refresh_tokens_user_id ON refresh_tokens(user_id)
- idx_ai_usage_logs_tenant_id ON ai_usage_logs(tenant_id)

---

### Migration 000002: accounting_schema

**Purpose:** Implements double-entry accounting system for financial tracking within tenants.

**Tables Created:**

#### chart_of_accounts
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Account identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| code | VARCHAR(50) | NOT NULL | Account code |
| name | VARCHAR(255) | NOT NULL | Account name |
| type | VARCHAR(50) | NOT NULL | Account type (asset, liability, equity, revenue, expense) |
| parent_id | UUID | FK -> chart_of_accounts(id) | Parent account for hierarchy |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

**Key Constraint:** UNIQUE(tenant_id, code)

#### journal_entries
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Entry identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| date | DATE | NOT NULL | Entry date |
| description | TEXT | NOT NULL | Entry description |
| reference | VARCHAR(100) | NULL | External reference |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### journal_lines
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Line identifier |
| entry_id | UUID | NOT NULL, FK -> journal_entries(id) | Parent journal entry |
| account_id | UUID | NOT NULL, FK -> chart_of_accounts(id) | Target account |
| debit | BIGINT | NOT NULL, DEFAULT 0 | Debit amount (in cents) |
| credit | BIGINT | NOT NULL, DEFAULT 0 | Credit amount (in cents) |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

**Relationships:**
- chart_of_accounts.tenant_id -> tenants.id (CASCADE)
- chart_of_accounts.parent_id -> chart_of_accounts.id (self-referential)
- journal_entries.tenant_id -> tenants.id (CASCADE)
- journal_lines.entry_id -> journal_entries.id (CASCADE)
- journal_lines.account_id -> chart_of_accounts(id)

**Indexes:**
- idx_coa_tenant_id ON chart_of_accounts(tenant_id)
- idx_journal_entries_tenant_date ON journal_entries(tenant_id, date)
- idx_journal_lines_entry_id ON journal_lines(entry_id)
- idx_journal_lines_account_id ON journal_lines(account_id)

---

### Migration 000003: chatbot_schema

**Purpose:** Enables chat session management for AI chatbot functionality.

**Tables Created:**

#### chat_sessions
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Session identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| title | VARCHAR(255) | NULL | Session title |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### chat_messages
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Message identifier |
| session_id | UUID | NOT NULL, FK -> chat_sessions(id) | Parent session |
| role | VARCHAR(50) | NOT NULL | Message role (user, assistant, system) |
| content | TEXT | NOT NULL | Message content |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |

**Relationships:**
- chat_sessions.tenant_id -> tenants.id (CASCADE)
- chat_messages.session_id -> chat_sessions.id (CASCADE)

**Indexes:**
- idx_chat_sessions_tenant ON chat_sessions(tenant_id)
- idx_chat_messages_session ON chat_messages(session_id)

---

### Migration 000004: campaign_schema

**Purpose:** Comprehensive political campaign management system with voter tracking, volunteer coordination, and survey capabilities.

**Tables Created:**

#### candidates
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Candidate identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| name | VARCHAR(255) | NOT NULL | Candidate name |
| status | VARCHAR(50) | NOT NULL, DEFAULT 'draft' | Candidate status |
| verification_document | VARCHAR(255) | NULL | Verification document path |
| is_verified | BOOLEAN | DEFAULT FALSE | Verification status |
| suspended | BOOLEAN | DEFAULT FALSE | Suspension status |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### campaigns
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Campaign identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| candidate_id | UUID | NOT NULL, FK -> candidates(id) | Associated candidate |
| name | VARCHAR(255) | NOT NULL | Campaign name |
| status | VARCHAR(50) | NOT NULL, DEFAULT 'draft' | Campaign status (draft, active, completed) |
| target_voters | INT | NOT NULL, DEFAULT 0 | Target voter count |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### Geographic Hierarchy (Indonesia)

##### provinces
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Province identifier |
| name | VARCHAR(255) | NOT NULL | Province name |

##### regencies
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Regency/city identifier |
| province_id | UUID | NOT NULL, FK -> provinces(id) | Parent province |
| name | VARCHAR(255) | NOT NULL | Regency name |

##### districts
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | District identifier |
| regency_id | UUID | NOT NULL, FK -> regencies(id) | Parent regency |
| name | VARCHAR(255) | NOT NULL | District name |

##### villages
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Village identifier |
| district_id | UUID | NOT NULL, FK -> districts(id) | Parent district |
| name | VARCHAR(255) | NOT NULL | Village name |

##### tps (Tempat Pemungutan Suara)
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | TPS identifier |
| village_id | UUID | NOT NULL, FK -> villages(id) | Parent village |
| name | VARCHAR(100) | NOT NULL | TPS name/number |
| target_voters | INT | NOT NULL, DEFAULT 0 | Target voters for this TPS |

#### volunteers
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Volunteer identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| user_id | UUID | FK -> users(id) | Associated user (nullable) |
| name | VARCHAR(255) | NOT NULL | Volunteer name |
| phone | VARCHAR(50) | NULL | Phone number |
| rank | INT | DEFAULT 0 | Volunteer rank/level |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### volunteer_assignments
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Assignment identifier |
| volunteer_id | UUID | NOT NULL, FK -> volunteers(id) | Assigned volunteer |
| village_id | UUID | FK -> villages(id) | Assigned village |
| tps_id | UUID | FK -> tps(id) | Assigned TPS |
| assigned_at | TIMESTAMPTZ | DEFAULT NOW() | Assignment timestamp |

#### voters
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Voter identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| village_id | UUID | FK -> villages(id) | Village location |
| tps_id | UUID | FK -> tps(id) | TPS location |
| nik_encrypted | VARCHAR(255) | NOT NULL | Encrypted NIK (National ID) |
| name_encrypted | VARCHAR(255) | NOT NULL | Encrypted name |
| address_encrypted | VARCHAR(255) | NULL | Encrypted address |
| phone | VARCHAR(50) | NULL | Phone number |
| status | VARCHAR(50) | DEFAULT 'uncontacted' | Support status (uncontacted, strong_supporter, weak_supporter, opponent) |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### voter_interactions
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Interaction identifier |
| voter_id | UUID | NOT NULL, FK -> voters(id) | Interacted voter |
| volunteer_id | UUID | FK -> volunteers(id) | Interacting volunteer |
| interaction_type | VARCHAR(50) | NOT NULL | Type of interaction |
| notes | TEXT | NULL | Interaction notes |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |

#### events
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Event identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| campaign_id | UUID | NOT NULL, FK -> campaigns(id) | Associated campaign |
| name | VARCHAR(255) | NOT NULL | Event name |
| event_date | TIMESTAMPTZ | NOT NULL | Event date/time |
| location | VARCHAR(255) | NULL | Event location |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### event_attendees
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Attendee identifier |
| event_id | UUID | NOT NULL, FK -> events(id) | Associated event |
| voter_id | UUID | FK -> voters(id) | Attending voter |
| scanned_at | TIMESTAMPTZ | DEFAULT NOW() | Check-in timestamp |

#### surveys
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Survey identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| campaign_id | UUID | NOT NULL, FK -> campaigns(id) | Associated campaign |
| name | VARCHAR(255) | NOT NULL | Survey name |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### survey_questions
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Question identifier |
| survey_id | UUID | NOT NULL, FK -> surveys(id) | Parent survey |
| question_text | TEXT | NOT NULL | Question content |
| question_type | VARCHAR(50) | NOT NULL | Question type (choice, text) |
| options | JSONB | NULL | Options for choice questions |

#### survey_responses
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Response identifier |
| survey_id | UUID | NOT NULL, FK -> surveys(id) | Associated survey |
| voter_id | UUID | FK -> voters(id) | Responding voter |
| volunteer_id | UUID | FK -> volunteers(id) | Collecting volunteer |
| answers | JSONB | NULL | Response answers |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |

**Relationships:**
- campaigns.tenant_id -> tenants.id (CASCADE)
- campaigns.candidate_id -> candidates.id (CASCADE)
- regencies.province_id -> provinces.id (CASCADE)
- districts.regency_id -> regencies.id (CASCADE)
- villages.district_id -> districts.id (CASCADE)
- tps.village_id -> villages.id (CASCADE)
- volunteers.tenant_id -> tenants.id (CASCADE)
- volunteers.user_id -> users.id (SET NULL)
- volunteer_assignments.volunteer_id -> volunteers.id (CASCADE)
- volunteer_assignments.village_id -> villages(id) (SET NULL)
- volunteer_assignments.tps_id -> tps(id) (SET NULL)
- voters.tenant_id -> tenants.id (CASCADE)
- voters.village_id -> villages(id) (SET NULL)
- voters.tps_id -> tps(id) (SET NULL)
- voter_interactions.voter_id -> voters.id (CASCADE)
- voter_interactions.volunteer_id -> volunteers(id) (SET NULL)
- events.tenant_id -> tenants.id (CASCADE)
- events.campaign_id -> campaigns.id (CASCADE)
- event_attendees.event_id -> events.id (CASCADE)
- event_attendees.voter_id -> voters(id) (SET NULL)
- surveys.tenant_id -> tenants.id (CASCADE)
- surveys.campaign_id -> campaigns.id (CASCADE)
- survey_questions.survey_id -> surveys.id (CASCADE)
- survey_responses.survey_id -> surveys.id (CASCADE)
- survey_responses.voter_id -> voters(id) (SET NULL)
- survey_responses.volunteer_id -> volunteers(id) (SET NULL)

**Indexes:**
- idx_candidates_tenant_id ON candidates(tenant_id)
- idx_campaigns_tenant_id ON campaigns(tenant_id)
- idx_volunteers_tenant_id ON volunteers(tenant_id)
- idx_voters_tenant_id ON voters(tenant_id)
- idx_events_tenant_id ON events(tenant_id)

---

### Migration 000005: crypto_schema

**Purpose:** Crypto trading bot infrastructure with encrypted API key storage and multiple bot types.

**Tables Created:**

#### exchange_api_keys
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Key identifier |
| user_id | UUID | NOT NULL, FK -> users(id) | Owner user |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| exchange | VARCHAR(50) | NOT NULL | Exchange name (binance, tokocrypto, indodax) |
| label | VARCHAR(100) | NOT NULL | User-defined label |
| encrypted_api_key | TEXT | NOT NULL | AES-256-GCM encrypted API key |
| encrypted_api_secret | TEXT | NOT NULL | AES-256-GCM encrypted API secret |
| is_active | BOOLEAN | NOT NULL, DEFAULT true | Active status |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### bots
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Bot identifier |
| user_id | UUID | NOT NULL, FK -> users(id) | Bot owner |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| api_key_id | UUID | FK -> exchange_api_keys(id) | Associated API key |
| name | VARCHAR(100) | NOT NULL | Bot name |
| bot_type | VARCHAR(20) | NOT NULL | Bot type (dca, grid, signal) |
| pair | VARCHAR(20) | NOT NULL | Trading pair (e.g., BTCUSDT) |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'stopped' | Status (running, paused, stopped) |
| is_paper_trading | BOOLEAN | NOT NULL, DEFAULT true | Paper trading mode |
| dca_interval | VARCHAR(20) | NULL | DCA interval (hourly, daily, weekly, monthly) |
| dca_amount_per_order | BIGINT | DEFAULT 0 | DCA amount in USDT cents |
| grid_lower_price | BIGINT | DEFAULT 0 | Grid lower price bound in USDT cents |
| grid_upper_price | BIGINT | DEFAULT 0 | Grid upper price bound in USDT cents |
| grid_count | INT | DEFAULT 0 | Number of grid levels |
| grid_investment | BIGINT | DEFAULT 0 | Total grid investment in USDT cents |
| total_invested | BIGINT | NOT NULL, DEFAULT 0 | Total invested in USDT cents |
| total_profit | BIGINT | NOT NULL, DEFAULT 0 | Realized profit in USDT cents |
| total_trades | INT | NOT NULL, DEFAULT 0 | Total completed trades |
| last_executed_at | TIMESTAMPTZ | NULL | Last trade execution time |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### bot_orders
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Order identifier |
| bot_id | UUID | NOT NULL, FK -> bots(id) | Parent bot |
| side | VARCHAR(4) | NOT NULL | Order side (buy, sell) |
| price | BIGINT | NOT NULL | Price in USDT cents |
| quantity | BIGINT | NOT NULL | Quantity in satoshi-level (x10^8) |
| total | BIGINT | NOT NULL | Total cost in USDT cents |
| fee | BIGINT | NOT NULL, DEFAULT 0 | Fee in USDT cents |
| exchange_order_id | VARCHAR(100) | NULL | Exchange order ID |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'pending' | Order status (pending, filled, failed, cancelled) |
| is_paper | BOOLEAN | NOT NULL, DEFAULT true | Simulated trade flag |
| error_message | TEXT | NULL | Error details if failed |
| executed_at | TIMESTAMPTZ | NULL | Execution timestamp |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |

#### bot_pnl_snapshots
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Snapshot identifier |
| bot_id | UUID | NOT NULL, FK -> bots(id) | Associated bot |
| total_invested | BIGINT | NOT NULL, DEFAULT 0 | Cumulative invested |
| current_value | BIGINT | NOT NULL, DEFAULT 0 | Current portfolio value |
| realized_pnl | BIGINT | NOT NULL, DEFAULT 0 | Realized profit/loss |
| unrealized_pnl | BIGINT | NOT NULL, DEFAULT 0 | Unrealized profit/loss |
| snapshot_at | TIMESTAMPTZ | DEFAULT NOW() | Snapshot timestamp |

**Relationships:**
- exchange_api_keys.user_id -> users(id) (CASCADE)
- exchange_api_keys.tenant_id -> tenants(id) (CASCADE)
- bots.user_id -> users(id) (CASCADE)
- bots.tenant_id -> tenants(id) (CASCADE)
- bots.api_key_id -> exchange_api_keys(id) (SET NULL)
- bot_orders.bot_id -> bots(id) (CASCADE)
- bot_pnl_snapshots.bot_id -> bots(id) (CASCADE)

**Indexes:**
- idx_exchange_api_keys_user_id ON exchange_api_keys(user_id)
- idx_exchange_api_keys_tenant_id ON exchange_api_keys(tenant_id)
- idx_bots_user_id ON bots(user_id)
- idx_bots_tenant_id ON bots(tenant_id)
- idx_bots_status ON bots(status)
- idx_bot_orders_bot_id ON bot_orders(bot_id)
- idx_bot_orders_status ON bot_orders(status)
- idx_bot_orders_executed_at ON bot_orders(executed_at)
- idx_bot_pnl_snapshots_bot_id ON bot_pnl_snapshots(bot_id)
- idx_bot_pnl_snapshots_snapshot_at ON bot_pnl_snapshots(snapshot_at)

---

### Migration 000006: crypto_bot_real_testing

**Purpose:** Adds grid trading capabilities to the crypto bot system.

**Table Modifications:**

#### bots (Modified)
Added column: `has_open_position` BOOLEAN DEFAULT FALSE

#### bot_grid_lines (New)
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Grid line identifier |
| bot_id | UUID | NOT NULL, FK -> bots(id) | Parent bot |
| price | BIGINT | NOT NULL | Grid price level in USDT cents |
| side | VARCHAR(10) | NOT NULL | Order side (buy, sell) |
| status | VARCHAR(20) | NOT NULL | Line status (pending, active, filled) |
| exchange_order_id | VARCHAR(100) | NULL | Exchange order ID |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

**Relationships:**
- bot_grid_lines.bot_id -> bots(id) (CASCADE)

**Indexes:**
- idx_bot_grid_lines_bot_id ON bot_grid_lines(bot_id)

---

### Migration 000007: crypto_features

**Purpose:** Adds notification system for crypto trading events.

**Tables Created:**

#### crypto_notifications
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Notification identifier |
| tenant_id | VARCHAR(50) | NOT NULL | Tenant identifier |
| user_id | VARCHAR(50) | NOT NULL | User identifier |
| title | VARCHAR(100) | NOT NULL | Notification title |
| message | TEXT | NOT NULL | Notification content |
| type | VARCHAR(50) | NOT NULL | Notification type |
| is_read | BOOLEAN | NOT NULL, DEFAULT FALSE | Read status |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT NOW() | Creation timestamp |

**Indexes:**
- idx_crypto_notif_user ON crypto_notifications(tenant_id, user_id, created_at DESC)

---

### Migration 000008: user_phone_number

**Purpose:** Adds phone number field to users for SMS-based authentication.

**Table Modifications:**

#### users (Modified)
Added column: `phone_number` VARCHAR(20) UNIQUE

---

### Migration 000009: password_resets

**Purpose:** Implements password reset functionality with token-based verification.

**Tables Created:**

#### password_resets
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Reset token identifier |
| email | VARCHAR(255) | NOT NULL | User email |
| token | VARCHAR(255) | NOT NULL, UNIQUE | Reset token |
| expires_at | TIMESTAMPTZ | NOT NULL | Token expiration |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |

**Indexes:**
- idx_password_resets_email ON password_resets(email)
- idx_password_resets_token ON password_resets(token)

---

### Migration 000010: tenant_whatsapp_settings

**Purpose:** Enables WhatsApp integration via Fonnte for tenant notifications.

**Table Modifications:**

#### tenants (Modified)
Added columns:
- `fonnte_token` VARCHAR(255) - Fonnte API token
- `wa_number` VARCHAR(50) - WhatsApp number

---

### Migration 000011: business_types

**Purpose:** Implements business type catalog with tenant onboarding system and usage quotas.

**Tables Created:**

#### business_types
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | VARCHAR(50) | PRIMARY KEY | Business type identifier |
| name | VARCHAR(100) | NOT NULL | Display name |
| description | TEXT | NULL | Type description |
| icon | VARCHAR(50) | NULL | Icon identifier |
| default_modules | JSONB | NOT NULL, DEFAULT '[]' | Default enabled modules |
| default_dashboard_widgets | JSONB | NOT NULL, DEFAULT '[]' | Default dashboard widgets |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |

**Business Types Seeded:**
- umum (General)
- warung (Warung/Toko Kelontong)
- laundry (Laundry)
- industri_kreatif (Industri Kreatif)
- toko_online (Toko Online/E-Commerce)
- restoran (Restoran/F&B)
- jasa (Jasa/Service)

#### tenant_module_config
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Config identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| module_key | VARCHAR(100) | NOT NULL | Module identifier |
| enabled | BOOLEAN | DEFAULT TRUE | Module enabled status |
| config | JSONB | DEFAULT '{}' | Module configuration |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

**Key Constraint:** UNIQUE(tenant_id, module_key)

#### usage_quotas
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | PRIMARY KEY, FK -> tenants(id) | Tenant ownership |
| plan_tier | VARCHAR(20) | NOT NULL, DEFAULT 'lite' | Subscription tier |
| transactions_used | INT | DEFAULT 0 | Used transactions count |
| transactions_limit | INT | DEFAULT 100 | Transaction limit |
| users_used | INT | DEFAULT 1 | Used user count |
| users_limit | INT | DEFAULT 1 | User limit |
| ai_requests_used | INT | DEFAULT 0 | Used AI requests |
| ai_requests_limit | INT | DEFAULT 5 | AI request limit |
| bots_used | INT | DEFAULT 0 | Used bot count |
| bots_limit | INT | DEFAULT 0 | Bot limit |
| period_start | TIMESTAMPTZ | DEFAULT NOW() | Billing period start |
| period_end | TIMESTAMPTZ | DEFAULT NOW() + INTERVAL '30 days' | Billing period end |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

**Table Modifications:**

#### tenants (Modified)
Added columns:
- `business_type` VARCHAR(50) REFERENCES business_types(id) DEFAULT 'umum'
- `onboarding_completed` BOOLEAN DEFAULT FALSE
- `business_name` VARCHAR(255)
- `business_address` TEXT

**Relationships:**
- tenant_module_config.tenant_id -> tenants(id) (CASCADE)
- usage_quotas.tenant_id -> tenants(id) (CASCADE)

**Indexes:**
- idx_tenant_module_config_tenant ON tenant_module_config(tenant_id)

---

### Migration 000012: phone_verified_and_email_nullable

**Purpose:** Improves authentication flexibility by allowing phone verification and optional email.

**Table Modifications:**

#### users (Modified)
- Added column: `is_phone_verified` BOOLEAN NOT NULL DEFAULT false
- Changed: `email` now allows NULL
- Changed: Dropped unique constraint on email, replaced with partial unique index `users_email_unique_when_set` WHERE email IS NOT NULL AND email != ''

---

### Migration 000013: tenant_logo

**Purpose:** Adds branding support with tenant logo URL.

**Table Modifications:**

#### tenants (Modified)
Added column: `logo_url` VARCHAR(255)

---

### Migration 000014: faq_and_forwarders

**Purpose:** Enables FAQ management and WhatsApp message forwarding for tenants.

**Tables Created:**

#### tenant_faqs
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | FAQ identifier |
| tenant_id | UUID | FK -> tenants(id) | Tenant ownership |
| question | TEXT | NOT NULL | FAQ question |
| answer | TEXT | NOT NULL | FAQ answer |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT NOW() | Creation timestamp |

#### tenant_forwarders
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Forwarder identifier |
| tenant_id | UUID | FK -> tenants(id) | Tenant ownership |
| phone_number | VARCHAR(50) | NOT NULL | Forward-to phone number |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT NOW() | Creation timestamp |

**Relationships:**
- tenant_faqs.tenant_id -> tenants(id) (CASCADE)
- tenant_forwarders.tenant_id -> tenants(id) (CASCADE)

---

### Migration 000015: add_category_and_stock_to_products

**Purpose:** Adds inventory management capabilities to products.

**Table Modifications:**

#### products (Modified)
Added columns:
- `category` VARCHAR(100) DEFAULT 'Umum'
- `stock_quantity` INT DEFAULT 0

---

### Migration 000016: add_additional_photos_to_products

**Purpose:** Extends product catalog with multiple product images.

**Table Modifications:**

#### products (Modified)
Added column: `additional_photos` JSONB DEFAULT '[]'::jsonb

---

### Migration 000017: pos_transactions

**Purpose:** Implements Point of Sale transaction recording system.

**Tables Created:**

#### pos_transactions
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Transaction identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| reference | VARCHAR(50) | UNIQUE, NOT NULL | Transaction reference number |
| total_amount | NUMERIC(15,2) | NOT NULL | Transaction total |
| payment_method | VARCHAR(20) | NOT NULL | Payment method used |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'pending' | Transaction status |
| items_json | JSONB | NOT NULL | Line items as JSON |
| created_at | TIMESTAMP WITH TIME ZONE | DEFAULT CURRENT_TIMESTAMP | Creation timestamp |
| updated_at | TIMESTAMP WITH TIME ZONE | DEFAULT CURRENT_TIMESTAMP | Last update timestamp |

**Relationships:**
- pos_transactions.tenant_id -> tenants(id) (CASCADE)

**Indexes:**
- idx_pos_trx_tenant ON pos_transactions(tenant_id)
- idx_pos_trx_status ON pos_transactions(status)
- idx_pos_trx_reference ON pos_transactions(reference)

---

### Migration 000018: tenant_xendit_settings

**Purpose:** Enables Xendit payment gateway integration.

**Table Modifications:**

#### tenants (Modified)
- Removed: `qris_data` column
- Added: `xendit_api_key` VARCHAR(255)
- Added: `xendit_webhook_token` VARCHAR(255)

---

### Migration 000019: tenant_domains

**Purpose:** Adds custom domain and subdomain support for tenant websites.

**Table Modifications:**

#### tenants (Modified)
Added columns:
- `custom_domain` VARCHAR(255) UNIQUE
- `subdomain` VARCHAR(255) UNIQUE

---

### Migration 000020: tenant_contacts

**Purpose:** Manages customer contacts for tenant messaging systems.

**Tables Created:**

#### tenant_contacts
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Contact identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| phone_number | VARCHAR(50) | NOT NULL | Contact phone number |
| name | VARCHAR(255) | DEFAULT 'Pelanggan Baru' | Contact name |
| category | VARCHAR(50) | DEFAULT 'general' | Contact category |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

**Key Constraint:** UNIQUE(tenant_id, phone_number)

**Relationships:**
- tenant_contacts.tenant_id -> tenants(id) (CASCADE)

**Indexes:**
- idx_tenant_contacts_tenant_id ON tenant_contacts(tenant_id)

---

### Migration 000021: add_tenant_qris_and_report_settings

**Purpose:** Enables QRIS payment and scheduled reporting features.

**Table Modifications:**

#### tenants (Modified)
Added columns:
- `qris_enabled` BOOLEAN DEFAULT false
- `report_enabled` BOOLEAN DEFAULT false
- `report_time` VARCHAR(10) DEFAULT '07:00'

---

### Migration 000022: add_hotel_business

**Purpose:** Adds Hotel/Penginapan business type to the business type catalog.

**Table Modifications:**

#### business_types (Modified)
Added business type:
- hotel (Hotel/Penginapan) - Manajemen kamar, reservasi, dan layanan penginapan
- Modules: transactions, customers, reports, pos, room_management, booking
- Widgets: room_occupancy, daily_revenue, booking_status, income_summary, guest_demographics

---

### Migration 000023: add_coupons_table

**Purpose:** Implements coupon and discount system for tenant subscriptions.

**Tables Created:**

#### coupons
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Coupon identifier |
| code | VARCHAR(50) | UNIQUE, NOT NULL | Coupon code |
| discount_type | VARCHAR(20) | NOT NULL | Type: percentage, fixed, free_months |
| discount_value | INT | NOT NULL | Discount amount |
| duration_months | INT | DEFAULT 1 | Subscription duration |
| max_uses | INT | DEFAULT 0 | Maximum uses (0 = unlimited) |
| uses_count | INT | DEFAULT 0 | Current usage count |
| expires_at | TIMESTAMPTZ | NULL | Expiration date |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |

#### tenant_coupons
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | FK -> tenants(id) | Tenant that used coupon |
| coupon_id | UUID | FK -> coupons(id) | Applied coupon |
| applied_at | TIMESTAMPTZ | DEFAULT NOW() | Application timestamp |

**Primary Key:** (tenant_id, coupon_id)

**Relationships:**
- tenant_coupons.tenant_id -> tenants(id) (CASCADE)
- tenant_coupons.coupon_id -> coupons(id) (CASCADE)

**Data Changes:**
- Default promo coupon seeded: 'PROMO-LITE-90' (3 free months)
- usage_quotas.plan_tier default changed from 'free' to 'lite'
- All existing 'free' tenants migrated to 'lite'

---

### Migration 000024: tenant_notification_settings

**Purpose:** Configurable multi-channel notification preferences per tenant.

**Tables Created:**

#### tenant_notification_settings
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| tenant_id | UUID | PRIMARY KEY, FK -> tenants(id) | Tenant ownership |
| notify_email | BOOLEAN | DEFAULT TRUE | Email notifications enabled |
| notify_wa | BOOLEAN | DEFAULT TRUE | WhatsApp notifications enabled |
| notify_telegram | BOOLEAN | DEFAULT FALSE | Telegram notifications enabled |
| telegram_chat_id | VARCHAR(50) | NULL | Telegram chat ID |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

**Relationships:**
- tenant_notification_settings.tenant_id -> tenants(id) (CASCADE)

---

## Legacy Migrations

### migration.sql

**Purpose:** Adds parent-child user relationship, extends tasks and voters tables for campaign features.

**Changes:**
- `users`: Added `parent_id` UUID REFERENCES users(id) ON DELETE SET NULL
- `tasks`: Added `created_by`, `assigned_to`, `verification_type`, `proof_image`, `gps_location`, `is_verified` columns
- `voters`: Added `potential_level` and `competitor_support` columns

### seed.sql

**Purpose:** Seeds default candidates and campaigns for existing tenants.

---

## Campaign Features Part 2 (20260522)

**Purpose:** Adds task management, notification center, and RBAC system.

**Tables Created:**

#### tasks
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Task identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| campaign_id | UUID | NOT NULL, FK -> campaigns(id) | Associated campaign |
| title | VARCHAR(255) | NOT NULL | Task title |
| description | TEXT | NULL | Task description |
| status | VARCHAR(50) | NOT NULL, DEFAULT 'todo' | Task status (todo, in_progress, done) |
| deadline | TIMESTAMPTZ | NULL | Task deadline |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |
| updated_at | TIMESTAMPTZ | DEFAULT NOW() | Last update timestamp |

#### task_assignments
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Assignment identifier |
| task_id | UUID | NOT NULL, FK -> tasks(id) | Assigned task |
| volunteer_id | UUID | FK -> volunteers(id) | Assigned volunteer |
| assigned_at | TIMESTAMPTZ | DEFAULT NOW() | Assignment timestamp |

#### notifications
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Notification identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| title | VARCHAR(255) | NOT NULL | Notification title |
| message | TEXT | NOT NULL | Notification content |
| type | VARCHAR(50) | NOT NULL, DEFAULT 'in_app' | Notification type (in_app, email, broadcast) |
| status | VARCHAR(50) | NOT NULL, DEFAULT 'unread' | Read status (unread, read) |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |

#### roles
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Role identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| name | VARCHAR(50) | NOT NULL | Role name |
| description | VARCHAR(255) | NULL | Role description |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |

#### role_permissions
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Permission identifier |
| role_id | UUID | NOT NULL, FK -> roles(id) | Parent role |
| resource | VARCHAR(100) | NOT NULL | Resource name |
| action | VARCHAR(50) | NOT NULL | Action (read, write, delete) |

#### audit_logs
| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY | Log identifier |
| tenant_id | UUID | NOT NULL, FK -> tenants(id) | Tenant ownership |
| user_id | UUID | FK -> users(id) | Acting user |
| action | VARCHAR(100) | NOT NULL | Action performed |
| resource | VARCHAR(100) | NULL | Resource affected |
| details | JSONB | NULL | Additional details |
| created_at | TIMESTAMPTZ | DEFAULT NOW() | Creation timestamp |

**Relationships:**
- tasks.tenant_id -> tenants(id) (CASCADE)
- tasks.campaign_id -> campaigns(id) (CASCADE)
- task_assignments.task_id -> tasks(id) (CASCADE)
- task_assignments.volunteer_id -> volunteers(id) (SET NULL)
- notifications.tenant_id -> tenants(id) (CASCADE)
- roles.tenant_id -> tenants(id) (CASCADE)
- role_permissions.role_id -> roles(id) (CASCADE)
- audit_logs.tenant_id -> tenants(id) (CASCADE)
- audit_logs.user_id -> users(id) (SET NULL)

---

## Schema Summary

### Entity-Relationship Overview

```
tenants (1) ──┬── (*) users
              ├── (*) refresh_tokens
              ├── (*) ai_usage_logs
              ├── (*) chart_of_accounts ── (*) journal_entries ── (*) journal_lines
              ├── (*) chat_sessions ── (*) chat_messages
              ├── (*) exchange_api_keys ── (*) bots
              │                          ├─ (*) bot_orders
              │                          ├─ (*) bot_pnl_snapshots
              │                          └─ (*) bot_grid_lines
              ├── (*) candidates ── (*) campaigns
              │                     ├─ (*) events ── (*) event_attendees
              │                     ├─ (*) surveys ── (*) survey_questions
              │                     │                        └─ (*) survey_responses
              │                     ├─ (*) tasks ── (*) task_assignments
              │                     └─ (*) voters ── (*) voter_interactions
              ├── (*) provinces ── (*) regencies ── (*) districts ── (*) villages
              │                                                      └─ (*) tps
              ├── (*) volunteers ── (*) volunteer_assignments
              ├── (*) business_types
              ├── (*) tenant_module_config
              ├── (*) usage_quotas
              ├── (*) tenant_faqs
              ├── (*) tenant_forwarders
              ├── (*) tenant_contacts
              ├── (*) pos_transactions
              ├── (*) crypto_notifications
              ├── (*) tenant_notification_settings
              ├── (*) roles ── (*) role_permissions
              ├── (*) notifications
              └── (*) audit_logs
```

### Tables by Domain

**Authentication & Users:**
- users
- refresh_tokens
- password_resets
- ai_usage_logs

**Multi-Tenancy:**
- tenants
- business_types
- tenant_module_config
- usage_quotas

**Chat:**
- chat_sessions
- chat_messages

**Accounting:**
- chart_of_accounts
- journal_entries
- journal_lines

**Crypto Trading:**
- exchange_api_keys
- bots
- bot_orders
- bot_pnl_snapshots
- bot_grid_lines
- crypto_notifications

**Campaign Management:**
- candidates
- campaigns
- provinces
- regencies
- districts
- villages
- tps
- volunteers
- volunteer_assignments
- voters
- voter_interactions
- events
- event_attendees
- surveys
- survey_questions
- survey_responses
- tasks
- task_assignments

**E-Commerce/Point of Sale:**
- products (external, referenced)
- pos_transactions

**Communication:**
- tenant_faqs
- tenant_forwarders
- tenant_contacts
- tenant_notification_settings

**Access Control & Audit:**
- roles
- role_permissions
- audit_logs
- notifications

---

## Indexes Summary

| Table | Index | Columns |
|-------|-------|---------|
| users | idx_users_tenant_id | tenant_id |
| refresh_tokens | idx_refresh_tokens_user_id | user_id |
| ai_usage_logs | idx_ai_usage_logs_tenant_id | tenant_id |
| chart_of_accounts | idx_coa_tenant_id | tenant_id |
| journal_entries | idx_journal_entries_tenant_date | tenant_id, date |
| journal_lines | idx_journal_lines_entry_id | entry_id |
| journal_lines | idx_journal_lines_account_id | account_id |
| chat_sessions | idx_chat_sessions_tenant | tenant_id |
| chat_messages | idx_chat_messages_session | session_id |
| candidates | idx_candidates_tenant_id | tenant_id |
| campaigns | idx_campaigns_tenant_id | tenant_id |
| volunteers | idx_volunteers_tenant_id | tenant_id |
| voters | idx_voters_tenant_id | tenant_id |
| events | idx_events_tenant_id | tenant_id |
| exchange_api_keys | idx_exchange_api_keys_user_id | user_id |
| exchange_api_keys | idx_exchange_api_keys_tenant_id | tenant_id |
| bots | idx_bots_user_id | user_id |
| bots | idx_bots_tenant_id | tenant_id |
| bots | idx_bots_status | status |
| bot_orders | idx_bot_orders_bot_id | bot_id |
| bot_orders | idx_bot_orders_status | status |
| bot_orders | idx_bot_orders_executed_at | executed_at |
| bot_pnl_snapshots | idx_bot_pnl_snapshots_bot_id | bot_id |
| bot_pnl_snapshots | idx_bot_pnl_snapshots_snapshot_at | snapshot_at |
| bot_grid_lines | idx_bot_grid_lines_bot_id | bot_id |
| crypto_notifications | idx_crypto_notif_user | tenant_id, user_id, created_at DESC |
| password_resets | idx_password_resets_email | email |
| password_resets | idx_password_resets_token | token |
| tenant_module_config | idx_tenant_module_config_tenant | tenant_id |
| pos_transactions | idx_pos_trx_tenant | tenant_id |
| pos_transactions | idx_pos_trx_status | status |
| pos_transactions | idx_pos_trx_reference | reference |
| tenant_contacts | idx_tenant_contacts_tenant_id | tenant_id |

---

*Generated: May 2026*
*Project: WCH Platform*
