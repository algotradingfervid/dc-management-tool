ALTER TABLE users ADD COLUMN last_project_id INTEGER REFERENCES projects(id);
