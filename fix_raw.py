#!/usr/bin/env python3
path = '/opt/zhangyi/internal/api/asset_transaction_handlers.go'

with open(path, 'r') as f:
    content = f.read()

# Fix: the db.Raw("...` got corrupted by shell heredoc
# We need to properly write the Raw SQL call
old_raw = '''	db.Raw(`, bookID).Scan(&transactions)'''
new_raw = '''	db.Raw(`SELECT t.* FROM asset_transactions t JOIN asset_cards a ON t.card_id = a.id WHERE a.book_id = ? ORDER BY t.created_at DESC LIMIT 200`, bookID).Scan(&transactions)'''

content = content.replace(old_raw, new_raw)

with open(path, 'w') as f:
    f.write(content)

print('OK: fixed corrupted Raw call')

