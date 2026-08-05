INSERT INTO app_settings (key, value_json) VALUES
    ('smtp.enabled', 'false'),
    ('smtp.host', '""'),
    ('smtp.port', '587'),
    ('smtp.username', '""'),
    ('smtp.password', '""'),
    ('smtp.from_email', '""'),
    ('smtp.tls_mode', '"starttls"'),
    ('smtp.allow_private_networks', 'false')
ON CONFLICT(key) DO NOTHING;
