-- F055: Fix existing superadmin user missing phone_number
UPDATE users SET phone_number = '08210113344'
WHERE role = 'superadmin' AND phone_number IS NULL;