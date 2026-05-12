INSERT INTO organizations (id, name, slug, created_at) VALUES
  (1, 'Brix Services', 'brix', '2026-05-11 00:00:00.000000')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  slug = VALUES(slug);

INSERT INTO users (id, organization_id, email, password_hash, display_name, role, created_at, updated_at) VALUES
  (1, 1, 'manager1@brix.test', '$2a$10$dbqYuIXKpwIb0heW8bQkoe9JyC2E5Ih/1n.TJkAXT2UjanW5TxVQW', 'Sarah Manager', 'manager', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000'),
  (2, 1, 'manager2@brix.test', '$2a$10$dbqYuIXKpwIb0heW8bQkoe9JyC2E5Ih/1n.TJkAXT2UjanW5TxVQW', 'Alex Manager', 'manager', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000'),
  (3, 1, 'technician1@brix.test', '$2a$10$dbqYuIXKpwIb0heW8bQkoe9JyC2E5Ih/1n.TJkAXT2UjanW5TxVQW', 'Tom Technician', 'technician', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000'),
  (4, 1, 'technician2@brix.test', '$2a$10$dbqYuIXKpwIb0heW8bQkoe9JyC2E5Ih/1n.TJkAXT2UjanW5TxVQW', 'Priya Technician', 'technician', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000')
ON DUPLICATE KEY UPDATE
  organization_id = VALUES(organization_id),
  email = VALUES(email),
  password_hash = VALUES(password_hash),
  display_name = VALUES(display_name),
  role = VALUES(role),
  updated_at = VALUES(updated_at);

INSERT INTO managers (id, user_id, created_at) VALUES
  (1, 1, '2026-05-11 00:00:00.000000'),
  (2, 2, '2026-05-11 00:00:00.000000')
ON DUPLICATE KEY UPDATE user_id = VALUES(user_id);

INSERT INTO technicians (id, user_id, created_at) VALUES
  (1, 3, '2026-05-11 00:00:00.000000'),
  (2, 4, '2026-05-11 00:00:00.000000')
ON DUPLICATE KEY UPDATE user_id = VALUES(user_id);

INSERT INTO technician_availability_rules (organization_id, technician_id, weekday, starts_at, ends_at) VALUES
  (1, 1, 0, '08:00:00', '18:00:00'),
  (1, 1, 1, '08:00:00', '18:00:00'),
  (1, 1, 2, '08:00:00', '18:00:00'),
  (1, 1, 3, '08:00:00', '18:00:00'),
  (1, 1, 4, '08:00:00', '18:00:00'),
  (1, 2, 0, '08:00:00', '18:00:00'),
  (1, 2, 1, '08:00:00', '18:00:00'),
  (1, 2, 2, '08:00:00', '18:00:00'),
  (1, 2, 3, '08:00:00', '18:00:00'),
  (1, 2, 4, '08:00:00', '18:00:00')
ON DUPLICATE KEY UPDATE
  organization_id = VALUES(organization_id),
  starts_at = VALUES(starts_at),
  ends_at = VALUES(ends_at);

INSERT INTO quotes (id, organization_id, customer_name, description, status, created_at, updated_at) VALUES
  (1, 1, 'Acme Plumbing', 'Replace leaking kitchen tap', 'unscheduled', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000'),
  (2, 1, 'Northside Dental', 'Repair reception lighting', 'unscheduled', '2026-05-11 00:01:00.000000', '2026-05-11 00:01:00.000000'),
  (3, 1, 'Green Grocer', 'Service refrigeration unit', 'unscheduled', '2026-05-11 00:02:00.000000', '2026-05-11 00:02:00.000000'),
  (4, 1, 'City Gym', 'Inspect hot water system', 'unscheduled', '2026-05-11 00:03:00.000000', '2026-05-11 00:03:00.000000'),
  (5, 1, 'Harbor Cafe', 'Repair dishwasher drainage', 'unscheduled', '2026-05-11 00:04:00.000000', '2026-05-11 00:04:00.000000')
ON DUPLICATE KEY UPDATE
  organization_id = VALUES(organization_id),
  customer_name = VALUES(customer_name),
  description = VALUES(description),
  updated_at = VALUES(updated_at);
