const { execFileSync } = require('child_process');

const CONTAINER = 'openoms-postgres-1';

function getOrderFromDb(orderId) {
  const sql = `
SELECT json_build_object(
  'id', id,
  'tenant_id', tenant_id,
  'customer_name', customer_name,
  'customer_email', customer_email,
  'customer_phone', customer_phone,
  'status', status,
  'total_amount', total_amount,
  'currency', currency,
  'items', items,
  'payment_status', payment_status
)
FROM public.orders
WHERE id = '${orderId}'
LIMIT 1;
`;

  const output = execFileSync(
    'docker',
    [
      'exec',
      CONTAINER,
      'psql',
      '-U',
      'openoms',
      '-d',
      'openoms',
      '-t',
      '-A',
      '-c',
      sql,
    ],
    {
      encoding: 'utf8',
    }
  ).trim();

  if (!output) {
    throw new Error(`Order ${orderId} was not found in PostgreSQL`);
  }

  return JSON.parse(output);
}

module.exports = {
  getOrderFromDb,
};
