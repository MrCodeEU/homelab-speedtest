.PHONY: local-test clean-test

local-test:
	@echo "Setting up local test environment..."
	@mkdir -p test-env/keys
	@if [ ! -f test-env/keys/id_rsa ]; then \
		ssh-keygen -t rsa -b 4096 -f test-env/keys/id_rsa -N "" -C "test@homelab-speedtest"; \
	fi
	@cp test-env/keys/id_rsa.pub test-env/keys/authorized_keys
	@echo "Creating seed data..."
	@echo "INSERT INTO devices (name, hostname, ip, ssh_user, ssh_port) VALUES ('Node 1', 'node1', 'node1', 'root', 22), ('Node 2', 'node2', 'node2', 'root', 22);" > test-env/seed.sql
	@echo "INSERT INTO schedules (type, cron, enabled) VALUES ('ping', '*/1 * * * *', 1), ('speed', '*/5 * * * *', 1);" >> test-env/seed.sql
	@echo "Building and starting containers..."
	@docker compose -f docker-compose.test.yml up --build -d
	@echo "Waiting for database initialization..."
	@max_retries=10; \
	count=0; \
	while [ $$count -lt $$max_retries ]; do \
		if docker exec homelab-speedtest-server [ -f data/speedtest.db ]; then \
			echo "Database file found!"; \
			break; \
		fi; \
		echo "Waiting for database file... ($$((count+1))/$$max_retries)"; \
		sleep 2; \
		count=$$((count+1)); \
	done
	@echo "Seeding database..."
	@docker exec -i homelab-speedtest-server sqlite3 data/speedtest.db < test-env/seed.sql
	@echo "Test environment ready!"
	@echo "Server: http://localhost:8080"
	@echo "Node1: ssh root@localhost -p 2222 (Key: test-env/keys/id_rsa)"
	@echo "Node2: ssh root@localhost -p 2223 (Key: test-env/keys/id_rsa)"

clean-test:
	@echo "Cleaning up local test environment..."
	@docker compose -f docker-compose.test.yml down -v
	@rm -rf test-env
