CREATE TABLE IF NOT EXISTS organizations (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(120) NOT NULL,
  slug VARCHAR(80) NOT NULL UNIQUE,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  organization_id BIGINT NOT NULL,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  display_name VARCHAR(120) NOT NULL,
  role ENUM('manager', 'technician') NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
    ON UPDATE CURRENT_TIMESTAMP(6),
  CONSTRAINT fk_users_organization
    FOREIGN KEY (organization_id) REFERENCES organizations(id),
  INDEX idx_users_organization_role (organization_id, role, id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS sessions (
  id CHAR(36) PRIMARY KEY,
  organization_id BIGINT NOT NULL,
  user_id BIGINT NOT NULL,
  expires_at DATETIME(6) NOT NULL,
  revoked_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  last_seen_at DATETIME(6) NULL,
  CONSTRAINT fk_sessions_organization
    FOREIGN KEY (organization_id) REFERENCES organizations(id),
  CONSTRAINT fk_sessions_user
    FOREIGN KEY (user_id) REFERENCES users(id),
  INDEX idx_sessions_user_active (user_id, revoked_at, expires_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS managers (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL UNIQUE,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  CONSTRAINT fk_managers_user
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS technicians (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL UNIQUE,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  CONSTRAINT fk_technicians_user
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS technician_availability_rules (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  organization_id BIGINT NOT NULL,
  technician_id BIGINT NOT NULL,
  weekday TINYINT NOT NULL,
  starts_at TIME NOT NULL,
  ends_at TIME NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  CONSTRAINT fk_availability_organization
    FOREIGN KEY (organization_id) REFERENCES organizations(id),
  CONSTRAINT fk_availability_technician
    FOREIGN KEY (technician_id) REFERENCES technicians(id),
  CONSTRAINT chk_availability_weekday
    CHECK (weekday BETWEEN 0 AND 6),
  CONSTRAINT chk_availability_window_positive
    CHECK (ends_at > starts_at),
  UNIQUE KEY uq_availability_technician_weekday_start (technician_id, weekday, starts_at),
  INDEX idx_availability_lookup (organization_id, technician_id, weekday, starts_at, ends_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS quotes (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  organization_id BIGINT NOT NULL,
  customer_name VARCHAR(120) NOT NULL,
  description TEXT NOT NULL,
  status ENUM('unscheduled', 'scheduled') NOT NULL DEFAULT 'unscheduled',
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
    ON UPDATE CURRENT_TIMESTAMP(6),
  CONSTRAINT fk_quotes_organization
    FOREIGN KEY (organization_id) REFERENCES organizations(id),
  INDEX idx_quotes_org_status_created (organization_id, status, created_at, id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS jobs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  organization_id BIGINT NOT NULL,
  quote_id BIGINT NOT NULL UNIQUE,
  technician_id BIGINT NOT NULL,
  manager_id BIGINT NOT NULL,
  starts_at DATETIME(6) NOT NULL,
  ends_at DATETIME(6) NOT NULL,
  status ENUM('scheduled', 'completed') NOT NULL DEFAULT 'scheduled',
  completed_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
    ON UPDATE CURRENT_TIMESTAMP(6),
  CONSTRAINT fk_jobs_organization
    FOREIGN KEY (organization_id) REFERENCES organizations(id),
  CONSTRAINT fk_jobs_quote
    FOREIGN KEY (quote_id) REFERENCES quotes(id),
  CONSTRAINT fk_jobs_technician
    FOREIGN KEY (technician_id) REFERENCES technicians(id),
  CONSTRAINT fk_jobs_manager
    FOREIGN KEY (manager_id) REFERENCES managers(id),
  CONSTRAINT chk_job_window_positive
    CHECK (ends_at > starts_at),
  CONSTRAINT chk_job_window_two_hours
    CHECK (TIMESTAMPDIFF(MINUTE, starts_at, ends_at) = 120),
  INDEX idx_jobs_org_technician_window (organization_id, technician_id, starts_at, ends_at),
  INDEX idx_jobs_org_manager_status (organization_id, manager_id, status, starts_at, id),
  INDEX idx_jobs_org_technician_status (organization_id, technician_id, status, starts_at, id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS notifications (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  organization_id BIGINT NOT NULL,
  recipient_user_id BIGINT NOT NULL,
  actor_user_id BIGINT NULL,
  job_id BIGINT NULL,
  type ENUM('job_assigned', 'job_updated', 'job_completed') NOT NULL,
  message VARCHAR(500) NOT NULL,
  read_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  CONSTRAINT fk_notifications_organization
    FOREIGN KEY (organization_id) REFERENCES organizations(id),
  CONSTRAINT fk_notifications_recipient_user
    FOREIGN KEY (recipient_user_id) REFERENCES users(id),
  CONSTRAINT fk_notifications_actor_user
    FOREIGN KEY (actor_user_id) REFERENCES users(id),
  CONSTRAINT fk_notifications_job
    FOREIGN KEY (job_id) REFERENCES jobs(id),
  INDEX idx_notifications_recipient_created (recipient_user_id, created_at DESC, id DESC),
  INDEX idx_notifications_recipient_unread (recipient_user_id, read_at, created_at DESC, id DESC),
  INDEX idx_notifications_org_recipient (organization_id, recipient_user_id, id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS schedule_audit_logs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  organization_id BIGINT NOT NULL,
  actor_user_id BIGINT NOT NULL,
  action ENUM('job_assigned', 'job_rescheduled', 'job_completed') NOT NULL,
  job_id BIGINT NOT NULL,
  quote_id BIGINT NOT NULL,
  previous_technician_id BIGINT NULL,
  new_technician_id BIGINT NULL,
  previous_starts_at DATETIME(6) NULL,
  previous_ends_at DATETIME(6) NULL,
  new_starts_at DATETIME(6) NULL,
  new_ends_at DATETIME(6) NULL,
  new_completed_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  CONSTRAINT fk_audit_organization
    FOREIGN KEY (organization_id) REFERENCES organizations(id),
  CONSTRAINT fk_audit_actor_user
    FOREIGN KEY (actor_user_id) REFERENCES users(id),
  CONSTRAINT fk_audit_job
    FOREIGN KEY (job_id) REFERENCES jobs(id),
  CONSTRAINT fk_audit_quote
    FOREIGN KEY (quote_id) REFERENCES quotes(id),
  CONSTRAINT fk_audit_previous_technician
    FOREIGN KEY (previous_technician_id) REFERENCES technicians(id),
  CONSTRAINT fk_audit_new_technician
    FOREIGN KEY (new_technician_id) REFERENCES technicians(id),
  INDEX idx_audit_org_job_created (organization_id, job_id, created_at, id)
) ENGINE=InnoDB;
