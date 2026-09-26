-- seed_recipes.sql
-- Demo resep: Es Kopi Susu mengurangi bahan baku saat terjual.
-- Idempoten: aman dijalankan berulang (ON CONFLICT DO NOTHING).
-- Jalankan setelah 014/015: make migrate && make seed-recipes

BEGIN;

-- 1. Bahan baku + menu demo di store Central - Kemang (lattepos-central).
--    product_type memisahkan bahan baku dari barang jual.
WITH store AS (
    SELECT s.id AS store_id, s.organization_id
    FROM stores s
    JOIN organizations o ON o.id = s.organization_id
    WHERE o.slug = 'lattepos-central' AND s.code = 'KMG-01'
)
INSERT INTO products (store_id, organization_id, name, sku, description, product_type, price, stock, unit, is_active)
SELECT store.store_id, store.organization_id, v.name, v.sku, v.description, v.product_type, v.price, v.stock, v.unit, TRUE
FROM store
JOIN (VALUES
    ('BB-SUSU-250', 'Susu UHT',           'Bahan baku susu segar',   'RAW_MATERIAL', 0,     20000, 'ml'),
    ('BB-ESP-001',  'Espresso',           'Shot espresso base',      'RAW_MATERIAL', 0,     10000, 'ml'),
    ('BB-GAR-001',  'Gula Aren Syrup',    'Sirup gula aren',         'RAW_MATERIAL', 0,     8000,  'ml'),
    ('BB-ES-001',   'Es Batu',            'Es kristal',              'RAW_MATERIAL', 0,     15000, 'gram'),
    ('PK-CUP-001',  'Cup 16 oz',          'Gelas plastik + tutup',   'PACKAGING',    0,     500,   'pcs'),
    ('MN-KOPI-001', 'Es Kopi Susu',       'Es kopi susu gula aren',  'MENU',         22000, 0,     'pcs')
) AS v(sku, name, description, product_type, price, stock, unit)
    ON v.sku IS NOT NULL
ON CONFLICT (store_id, sku) DO NOTHING;

-- 2. Resep Es Kopi Susu (versi 1, aktif).
--    yield_quantity 1 = satu resep menghasilkan satu gelas.
INSERT INTO recipes (organization_id, store_id, product_id, name, version, yield_quantity, is_active, notes)
SELECT p.organization_id, p.store_id, p.id, 'Es Kopi Susu Gula Aren', 1, 1, TRUE, 'Resep standar barista'
FROM products p
JOIN organizations o ON o.id = p.organization_id
WHERE o.slug = 'lattepos-central' AND p.sku = 'MN-KOPI-001'
ON CONFLICT (product_id, version) DO NOTHING;

-- 3. Baris bahan: susu 250ml, espresso 20ml, gula aren 15ml (+5% susut),
--    cup 1 pcs, es batu 100 gram. Tiap bahan menunjuk product bahan baku.
INSERT INTO recipe_items (recipe_id, organization_id, store_id, ingredient_product_id, quantity, unit, wastage_percentage, sequence, notes)
SELECT r.id, r.organization_id, r.store_id, ing.id, v.quantity, v.unit, v.wastage, v.seq, v.notes
FROM recipes r
JOIN products menu ON menu.id = r.product_id AND menu.sku = 'MN-KOPI-001'
JOIN (VALUES
    ('BB-SUSU-250', 250::numeric, 'ml',   0::numeric, 1, 'Susu UHT dingin'),
    ('BB-ESP-001',  20::numeric,  'ml',   0::numeric, 2, 'Dua shot espresso'),
    ('BB-GAR-001',  15::numeric,  'ml',   5::numeric, 3, 'Sirup gula aren'),
    ('PK-CUP-001',  1::numeric,   'pcs',  0::numeric, 4, 'Cup 16 oz'),
    ('BB-ES-001',   100::numeric, 'gram', 0::numeric, 5, 'Es batu')
) AS v(sku, quantity, unit, wastage, seq, notes) ON TRUE
JOIN products ing ON ing.sku = v.sku AND ing.store_id = r.store_id
WHERE r.version = 1
ON CONFLICT (recipe_id, ingredient_product_id) DO NOTHING;

COMMIT;
