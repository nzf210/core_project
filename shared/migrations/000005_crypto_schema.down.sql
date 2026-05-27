-- Down migration for crypto schema
DROP TABLE IF EXISTS bot_pnl_snapshots CASCADE;
DROP TABLE IF EXISTS bot_orders CASCADE;
DROP TABLE IF EXISTS bots CASCADE;
DROP TABLE IF EXISTS exchange_api_keys CASCADE;
