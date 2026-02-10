ALTER TABLE products
  ADD COLUMN image_url VARCHAR(1024),
  ADD COLUMN images JSONB NOT NULL DEFAULT '[]';

COMMENT ON COLUMN products.image_url IS 'Primary product image URL (thumbnail)';
COMMENT ON COLUMN products.images IS 'Array of {url, alt, position} image objects';

-- Update seed data with placeholder images
UPDATE products SET
  image_url = 'https://placehold.co/600x600/e2e8f0/64748b?text=' || REPLACE(LEFT(name, 20), ' ', '+'),
  images = '[]'
WHERE image_url IS NULL;
