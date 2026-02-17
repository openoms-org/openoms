-- Carbon footprint tracking per shipment
ALTER TABLE shipments ADD COLUMN carbon_kg DECIMAL(8,3);
ALTER TABLE shipments ADD COLUMN distance_km DECIMAL(10,1);
ALTER TABLE shipments ADD COLUMN carbon_method TEXT DEFAULT 'estimate';
