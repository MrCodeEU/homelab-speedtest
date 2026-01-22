INSERT INTO devices (name, hostname, ip, ssh_user, ssh_port) VALUES ('Node 1', 'node1', 'node1', 'root', 22), ('Node 2', 'node2', 'node2', 'root', 22);
INSERT INTO schedules (type, cron, enabled) VALUES ('ping', '*/1 * * * *', 1), ('speed', '*/5 * * * *', 1);
