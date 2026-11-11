-- Frozen legacy fixture: the schema a v0.1.2 install has on disk (no uuid
-- columns) plus representative rows. Never edit it to track new schema; add a
-- new fixture file for each later legacy shape instead. Ids are deliberately
-- non-contiguous (deleted rows) so tests prove integer handles survive.

CREATE TABLE projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'todo',
    priority INTEGER DEFAULT 0,
    due_date DATETIME,
    planned_date DATE,
    recurring TEXT DEFAULT 'none',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL
);
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    color TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE task_tags (
    task_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (task_id, tag_id),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);
CREATE TABLE task_time_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    description TEXT,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);
CREATE TABLE habits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    frequency TEXT DEFAULT 'daily',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE habit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    habit_id INTEGER NOT NULL,
    completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (habit_id) REFERENCES habits(id) ON DELETE CASCADE
);
CREATE TABLE scheduled_timers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER,
    title TEXT NOT NULL,
    trigger_time DATETIME NOT NULL,
    status TEXT DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);
CREATE TABLE task_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    action TEXT NOT NULL,
    details TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

INSERT INTO projects (id, name, description) VALUES (1, 'Anytime', 'Default project for tasks'), (3, 'Website', 'Marketing site');

INSERT INTO tasks (id, project_id, title, description, status, priority, recurring) VALUES
  (1, 1, 'Write report', 'quarterly', 'todo', 1, 'none'),
  (2, 3, 'Fix header', '', 'done', 2, 'none'),
  (5, 3, 'Ship landing page', 'launch', 'todo', 3, 'weekly'),
  (6, 1, 'Review PR', '', 'InProgress', 1, 'none'),
  (7, NULL, 'Call the bank', '', 'todo', 0, 'none');

INSERT INTO tags (id, name, color) VALUES (1, 'work', '#0000ff'), (4, 'urgent', '#ff0000');
INSERT INTO task_tags (task_id, tag_id) VALUES (1, 1), (5, 1), (5, 4);

INSERT INTO task_time_logs (id, task_id, start_time, end_time) VALUES
  (1, 2, '2026-01-02 09:00:00', '2026-01-02 09:30:00'),
  (2, 5, '2026-01-03 10:00:00', '2026-01-03 10:01:30'),
  (3, 6, '2026-01-04 11:00:00', NULL);

INSERT INTO habits (id, name, description, frequency) VALUES (1, 'Read', '', 'daily'), (3, 'Stretch', 'morning', 'daily');
INSERT INTO habit_logs (id, habit_id, completed_at) VALUES (1, 1, '2026-01-02 08:00:00'), (2, 1, '2026-01-03 08:00:00'), (3, 3, '2026-01-03 08:05:00');

INSERT INTO scheduled_timers (id, task_id, title, trigger_time, status) VALUES
  (1, NULL, 'Stand up', '2026-01-02 09:00:00', 'triggered'),
  (2, 5, 'Check launch', '2099-01-01 00:00:00', 'pending');

INSERT INTO task_history (id, task_id, action, details, timestamp) VALUES
  (1, 1, 'created', 'Task created: ''Write report''', '2026-01-01 08:00:00'),
  (2, 2, 'status_changed', 'Status changed from ''todo'' to ''done''', '2026-01-02 09:31:00');
