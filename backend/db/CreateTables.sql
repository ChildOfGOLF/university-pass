/* ROLES */

CREATE TABLE IF NOT EXISTS roles (
	role_id SERIAL PRIMARY KEY,
	role varchar(16) /* студент/охранник/админ */
);

/* USERS */

CREATE TABLE IF NOT EXISTS users (
	user_id SERIAL PRIMARY KEY,
	role_id int NOT NULL,
	last_name varchar(50) NOT NULL,
	name varchar(30) NOT NULL,
	patronimic varchar(50),
	email varchar(50) UNIQUE NOT NULL,
	phone varchar(16), /* +7(999)999-99-99 */
	created_at timestamptz DEFAULT NOW()
);

/* GROUPS */

CREATE TABLE IF NOT EXISTS groups (
	group_id SERIAL PRIMARY KEY,
	group_name varchar(10)
);

/* STUDENTS */

CREATE TABLE IF NOT EXISTS students (
	student_id int PRIMARY KEY, /* users.user_id */
	group_id int NOT NULL
);

/* PASSWORDS */

CREATE TABLE IF NOT EXISTS passwords (
	password_id int PRIMARY KEY, /* users.user_id */
	password_hash varchar(128) NOT NULL
);

/* USER_DEVICES */

CREATE TABLE IF NOT EXISTS user_devices (
	user_id int PRIMARY KEY, /* users.user_id */
	device_id varchar(128) UNIQUE NOT NULL,
	created_at timestamptz DEFAULT NOW(), /* для отслеживания разницы мекжду updated_at */
	updated_at timestamptz DEFAULT NOW()
);

/* GUESTS */

CREATE TABLE IF NOT EXISTS guest_passes (
	uuid uuid PRIMARY KEY, /* или varchar(128) */
	last_name varchar(50) NOT NULL,
	name varchar(30) NOT NULL,
	patronimic varchar(50),
	email varchar(50) UNIQUE NOT NULL,
	phone varchar(16),
	valid_from timestamptz NOT NULL,
	valid_to timestamptz NOT NULL,
	is_used bool DEFAULT FALSE,
	created_at timestamptz DEFAULT NOW() /*,
	purpose text */
);

/* CAMPUSES */

CREATE TABLE IF NOT EXISTS campuses (
	campus_id SERIAL PRIMARY KEY,
	adress varchar(128)
);

/* ACCESS_POINTS */

CREATE TABLE IF NOT EXISTS access_points (
	point_id SERIAL PRIMARY KEY,
	campus_id int NOT NULL,
	scanner_id varchar(128) /* наподобие в user_devices */
);

/* VISIT_LOGS */

CREATE TABLE IF NOT EXISTS visit_logs (
	visit_id SERIAL PRIMARY KEY,
	guest_id uuid,
	user_id int,
	point_id int NOT NULL,
	visit_time timestamptz NOT NULL,
	direction bool NOT NULL, /* TRUE - enter, FALSE - out  */
	CONSTRAINT enter_only_one
		CHECK ((guest_id IS NOT NULL) != (user_id IS NOT NULL))
);
