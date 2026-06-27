SELECT '=== 用户 ===';
SELECT id, username, role FROM users;
SELECT '=== 所有表数据量 ===';
SELECT 'accounts' AS tbl, COUNT(*) AS cnt FROM accounts
UNION ALL SELECT 'vouchers', COUNT(*) FROM vouchers
UNION ALL SELECT 'voucher_items', COUNT(*) FROM voucher_items
UNION ALL SELECT 'account_balances', COUNT(*) FROM account_balances
UNION ALL SELECT 'opening_balances', COUNT(*) FROM opening_balances
UNION ALL SELECT 'operation_logs', COUNT(*) FROM operation_logs;
