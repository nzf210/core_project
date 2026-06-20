-- F055: Revert superadmin phone_number update
UPDATE users SET phone_number = NULL
WHERE role = 'superadmin' AND phone_number = '08210113344';