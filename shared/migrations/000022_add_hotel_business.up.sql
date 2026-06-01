INSERT INTO business_types (id, name, description, icon, default_modules, default_dashboard_widgets) VALUES
('hotel', 'Hotel / Penginapan', 'Manajemen kamar, reservasi, dan layanan penginapan', 'bed',
 '["transactions", "customers", "reports", "pos", "room_management", "booking"]',
 '["room_occupancy", "daily_revenue", "booking_status", "income_summary", "guest_demographics"]')
ON CONFLICT (id) DO NOTHING;
