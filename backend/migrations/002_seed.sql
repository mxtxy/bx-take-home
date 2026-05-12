INSERT INTO users (id, email, password_hash, display_name, role, created_at, updated_at) VALUES
  (1, 'manager1@brix.test', '$2a$10$dbqYuIXKpwIb0heW8bQkoe9JyC2E5Ih/1n.TJkAXT2UjanW5TxVQW', 'Sarah Manager', 'manager', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000'),
  (2, 'manager2@brix.test', '$2a$10$dbqYuIXKpwIb0heW8bQkoe9JyC2E5Ih/1n.TJkAXT2UjanW5TxVQW', 'Alex Manager', 'manager', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000'),
  (3, 'technician1@brix.test', '$2a$10$dbqYuIXKpwIb0heW8bQkoe9JyC2E5Ih/1n.TJkAXT2UjanW5TxVQW', 'Tom Technician', 'technician', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000'),
  (4, 'technician2@brix.test', '$2a$10$dbqYuIXKpwIb0heW8bQkoe9JyC2E5Ih/1n.TJkAXT2UjanW5TxVQW', 'Priya Technician', 'technician', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000')
ON DUPLICATE KEY UPDATE
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

INSERT INTO quotes (id, customer_name, description, status, created_at, updated_at) VALUES
  (1, 'Acme Plumbing', 'Replace leaking kitchen tap', 'unscheduled', '2026-05-11 00:00:00.000000', '2026-05-11 00:00:00.000000'),
  (2, 'Northside Dental', 'Repair reception lighting', 'unscheduled', '2026-05-11 00:01:00.000000', '2026-05-11 00:01:00.000000'),
  (3, 'Green Grocer', 'Service refrigeration unit', 'unscheduled', '2026-05-11 00:02:00.000000', '2026-05-11 00:02:00.000000'),
  (4, 'City Gym', 'Inspect hot water system', 'unscheduled', '2026-05-11 00:03:00.000000', '2026-05-11 00:03:00.000000'),
  (5, 'Harbor Cafe', 'Repair dishwasher drainage', 'unscheduled', '2026-05-11 00:04:00.000000', '2026-05-11 00:04:00.000000')
ON DUPLICATE KEY UPDATE
  customer_name = VALUES(customer_name),
  description = VALUES(description),
  updated_at = VALUES(updated_at);
