const fs = require('fs');
const path = '/opt/zhangyi/internal/api/asset_transaction_handlers.go';
let content = fs.readFileSync(path, 'utf-8');
content = content.replace(
  'db.Raw(, bookID).Scan(&transactions)',
  'db.Raw("SELECT t.* FROM asset_transactions t JOIN asset_cards a ON t.card_id = a.id WHERE a.book_id = ? ORDER BY t.created_at DESC LIMIT 200", bookID).Scan(&transactions)'
);
fs.writeFileSync(path, content);
console.log('OK');

