-- Areas Table (long-lived buckets; never complete)
CREATE TABLE IF NOT EXISTS areas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT,
    name TEXT NOT NULL,
    sort_key TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Headings Table (sections inside one project; hold only tasks)
CREATE TABLE IF NOT EXISTS headings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT,
    project_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    sort_key TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Repeaters Table (rule + template behind repeating tasks; occurrences are tasks)
CREATE TABLE IF NOT EXISTS repeaters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT,
    kind TEXT NOT NULL,
    interval INTEGER NOT NULL DEFAULT 1,
    weekdays TEXT NOT NULL DEFAULT '',
    month_day INTEGER NOT NULL DEFAULT 0,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    project_id INTEGER,
    area_id INTEGER,
    heading_id INTEGER,
    priority INTEGER NOT NULL DEFAULT 0,
    evening INTEGER NOT NULL DEFAULT 0,
    deadline_offset INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Projects Table
CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tasks Table
CREATE TABLE IF NOT EXISTS tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'open',
    priority INTEGER DEFAULT 0,
    when_kind TEXT DEFAULT 'anytime',
    when_date TEXT,
    deadline TEXT,
    evening INTEGER DEFAULT 0,
    repeater_id INTEGER,
    occurrence_date TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
);

-- Tags Table
CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    color TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Task Tags Join Table
CREATE TABLE IF NOT EXISTS task_tags (
    task_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (task_id, tag_id),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- Task Time Logs Table
CREATE TABLE IF NOT EXISTS task_time_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    description TEXT,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Habits Table
CREATE TABLE IF NOT EXISTS habits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    frequency TEXT DEFAULT 'daily',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Habit Logs Table
CREATE TABLE IF NOT EXISTS habit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    habit_id INTEGER NOT NULL,
    completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (habit_id) REFERENCES habits(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_habit_logs_habit ON habit_logs(habit_id);
CREATE INDEX IF NOT EXISTS idx_task_time_logs_task ON task_time_logs(task_id);

-- Scheduled Timers Table
CREATE TABLE IF NOT EXISTS scheduled_timers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER,
    title TEXT NOT NULL,
    trigger_time DATETIME NOT NULL,
    status TEXT DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

-- Task History Table
CREATE TABLE IF NOT EXISTS task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    details TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_scheduled_timers_status ON scheduled_timers(status);
CREATE INDEX IF NOT EXISTS idx_task_history_task ON task_history(task_id);


-- Sync groundwork (ADR 0010): per-field stamps, written with every mutation.
CREATE TABLE IF NOT EXISTS field_versions (
    entity TEXT NOT NULL,
    uuid TEXT NOT NULL,
    field TEXT NOT NULL,
    hlc TEXT NOT NULL,
    device_id TEXT NOT NULL,
    PRIMARY KEY (entity, uuid, field)
);

-- Device-local: this device's id and hybrid logical clock. Never synced.
CREATE TABLE IF NOT EXISTS sync_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    device_id TEXT NOT NULL,
    hlc_ms INTEGER NOT NULL DEFAULT 0,
    hlc_counter INTEGER NOT NULL DEFAULT 0
);
