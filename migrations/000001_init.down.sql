DROP INDEX IF EXISTS idx_search_history_query_type;
DROP INDEX IF EXISTS idx_search_history_created_at;
DROP INDEX IF EXISTS idx_search_history_user_id;
DROP TABLE IF EXISTS search_history;

DROP INDEX IF EXISTS idx_transactions_created_at;
DROP INDEX IF EXISTS idx_transactions_user_id;
DROP TABLE IF EXISTS transactions;

DROP TABLE IF EXISTS users;
