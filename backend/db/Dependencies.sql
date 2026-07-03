/* USERS - ROLES */

ALTER TABLE users
	ADD CONSTRAINT fk_role_id FOREIGN KEY (role_id)
	REFERENCES roles(role_id);

/* USERS - STUDENTS */

ALTER TABLE students
	ADD CONSTRAINT fk_student_id FOREIGN KEY (student_id)
	REFERENCES users(user_id);

/* GROUPS - STUDENTS */

ALTER TABLE students
	ADD CONSTRAINT fk_group_id FOREIGN KEY (group_id)
	REFERENCES groups(group_id);

/* USERS - PASSWORDS */

ALTER TABLE passwords
	ADD CONSTRAINT fk_password_id FOREIGN KEY (password_id)
	REFERENCES users(user_id);

/* USERS - USER_DEVICES */

ALTER TABLE user_devices
	ADD CONSTRAINT fk_device_user_id FOREIGN KEY (user_id)
	REFERENCES users(user_id);

/* ACCESS_POINTS - CAMPUSES */

ALTER TABLE access_points
	ADD CONSTRAINT fk_campus_id FOREIGN KEY (campus_id)
	REFERENCES campuses(campus_id);

/* VISIT_LOGS - {GUEST_PASSES, USERS, ACCESS_POINTS} */

ALTER TABLE visit_logs
	ADD CONSTRAINT fk_visit_guest_id FOREIGN KEY (guest_id) REFERENCES guest_passes(uuid),
	ADD CONSTRAINT fk_visit_user_id FOREIGN KEY (user_id) REFERENCES users(user_id),
	ADD CONSTRAINT fk_visit_point_id FOREIGN KEY (point_id) REFERENCES access_points(point_id)