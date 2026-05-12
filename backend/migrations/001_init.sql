CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  display_name VARCHAR(120) NOT NULL,
  role ENUM('manager', 'technician') NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
    ON UPDATE CURRENT_TIMESTAMP(6)
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

CREATE TABLE IF NOT EXISTS quotes (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_name VARCHAR(120) NOT NULL,
  description TEXT NOT NULL,
  status ENUM('unscheduled', 'scheduled') NOT NULL DEFAULT 'unscheduled',
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
    ON UPDATE CURRENT_TIMESTAMP(6),
  INDEX idx_quotes_status_created (status, created_at, id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS jobs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
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
  INDEX idx_jobs_technician_window (technician_id, starts_at, ends_at),
  INDEX idx_jobs_manager_status (manager_id, status, starts_at, id),
  INDEX idx_jobs_technician_status (technician_id, status, starts_at, id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS notifications (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  recipient_user_id BIGINT NOT NULL,
  actor_user_id BIGINT NULL,
  job_id BIGINT NULL,
  type ENUM('job_assigned', 'job_updated', 'job_completed') NOT NULL,
  message VARCHAR(500) NOT NULL,
  read_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  CONSTRAINT fk_notifications_recipient_user
    FOREIGN KEY (recipient_user_id) REFERENCES users(id),
  CONSTRAINT fk_notifications_actor_user
    FOREIGN KEY (actor_user_id) REFERENCES users(id),
  CONSTRAINT fk_notifications_job
    FOREIGN KEY (job_id) REFERENCES jobs(id),
  INDEX idx_notifications_recipient_created (recipient_user_id, created_at DESC, id DESC),
  INDEX idx_notifications_recipient_unread (recipient_user_id, read_at, created_at DESC, id DESC)
) ENGINE=InnoDB;
