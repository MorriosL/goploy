ALTER TABLE monitor ADD success_script text;
ALTER TABLE monitor ADD fail_script text;
ALTER TABLE monitor ADD success_server_id INT ( 10 ) UNSIGNED NOT NULL DEFAULT 0;
ALTER TABLE monitor ADD fail_server_id INT ( 10 ) UNSIGNED NOT NULL DEFAULT 0;
ALTER TABLE project ADD tag VARCHAR ( 6382 ) NOT NULL DEFAULT '';

INSERT IGNORE INTO `permission` (`id`, `pid`, `name`, `sort`, `description`) VALUES (78, 23, 'SFTPEditFile', 0, '');